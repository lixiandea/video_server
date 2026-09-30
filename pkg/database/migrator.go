package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/lixiandea/video_server/pkg/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 迁移脚本文件名规范：<4 位序号>_<snake_case 描述>.{up,down}.sql
var migrationFilePattern = regexp.MustCompile(`^(\d{4})_([a-z0-9_]+)\.(up|down)\.sql$`)

// migrationTableName 记录已应用版本的元数据表
const migrationTableName = "schema_migrations"

// Migration 表示一个版本化迁移，up / down 必须成对存在。
type Migration struct {
	Version  string
	Name     string
	UpPath   string
	DownPath string
	// Checksum 是 up 文件的 SHA-256，用于发现「已应用的迁移被事后修改」，
	// 避免库结构与仓库里的脚本悄悄分叉。
	Checksum string
}

// appliedMigration 映射 schema_migrations 表。
type appliedMigration struct {
	Version    string    `gorm:"primaryKey;type:varchar(20)"`
	Name       string    `gorm:"type:varchar(191);not null"`
	Checksum   string    `gorm:"type:char(64);not null"`
	AppliedAt  time.Time `gorm:"not null"`
	DurationMs int       `gorm:"not null"`
}

func (appliedMigration) TableName() string { return migrationTableName }

// MigrationState 是 status 命令的一行输出。
type MigrationState struct {
	Migration
	Applied          bool
	AppliedAt        time.Time
	ChecksumMismatch bool
}

// LoadMigrations 读取迁移目录，校验 up/down 成对并按版本号升序返回。
func LoadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取迁移目录 %s 失败: %w", dir, err)
	}

	byVersion := make(map[string]*Migration)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			return nil, fmt.Errorf("非法迁移文件名 %q（应形如 0001_init_schema.up.sql）", entry.Name())
		}
		version, name, kind := matches[1], matches[2], matches[3]

		m := byVersion[version]
		if m == nil {
			m = &Migration{Version: version, Name: name}
			byVersion[version] = m
		}
		if m.Name != name {
			return nil, fmt.Errorf("版本 %s 的描述不一致: %s / %s", version, m.Name, name)
		}

		switch kind {
		case "up":
			m.UpPath = filepath.Join(dir, entry.Name())
		case "down":
			m.DownPath = filepath.Join(dir, entry.Name())
		}
	}

	if len(byVersion) == 0 {
		return nil, fmt.Errorf("迁移目录 %s 下没有迁移脚本", dir)
	}

	migrations := make([]Migration, 0, len(byVersion))
	for _, m := range byVersion {
		if m.UpPath == "" || m.DownPath == "" {
			return nil, fmt.Errorf("版本 %s(%s) 的 up/down 脚本不完整，两者必须成对存在", m.Version, m.Name)
		}
		checksum, err := fileChecksum(m.UpPath)
		if err != nil {
			return nil, err
		}
		m.Checksum = checksum
		migrations = append(migrations, *m)
	}

	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}

// IsBaseline 判断该版本是否为基线（序号最小的迁移）。
// 基线的 down 会删除全部业务表，执行器默认拒绝回滚它。
func IsBaseline(migrations []Migration, version string) bool {
	return len(migrations) > 0 && migrations[0].Version == version
}

// MigrateUp 应用所有未执行的迁移，返回本次实际应用的版本。
// 整个过程持有 MySQL 命名锁，多实例并发时会明确失败而不是重复执行 DDL。
func MigrateUp(db *gorm.DB, dir string) ([]Migration, error) {
	migrations, err := LoadMigrations(dir)
	if err != nil {
		return nil, err
	}

	release, err := acquireMigrationLock(db)
	if err != nil {
		return nil, err
	}
	defer release()

	if err := ensureMigrationTable(db); err != nil {
		return nil, err
	}

	applied, err := loadApplied(db)
	if err != nil {
		return nil, err
	}
	if err := validateChecksums(applied, migrations); err != nil {
		return nil, err
	}

	var done []Migration
	for _, m := range migrations {
		if _, ok := applied[m.Version]; ok {
			continue
		}
		if err := applyMigration(db, m); err != nil {
			return done, err
		}
		done = append(done, m)
	}

	return done, nil
}

// MigrateDown 按版本倒序回滚最近 steps 个已应用的迁移。
// allowBaseline 为 false 时拒绝回滚基线版本。
func MigrateDown(db *gorm.DB, dir string, steps int, allowBaseline bool) ([]Migration, error) {
	if steps <= 0 {
		return nil, fmt.Errorf("回滚步数必须为正整数，当前为 %d", steps)
	}

	migrations, err := LoadMigrations(dir)
	if err != nil {
		return nil, err
	}

	release, err := acquireMigrationLock(db)
	if err != nil {
		return nil, err
	}
	defer release()

	if err := ensureMigrationTable(db); err != nil {
		return nil, err
	}

	applied, err := loadApplied(db)
	if err != nil {
		return nil, err
	}
	if err := validateChecksums(applied, migrations); err != nil {
		return nil, err
	}

	targets := make([]Migration, 0, steps)
	for i := len(migrations) - 1; i >= 0 && len(targets) < steps; i-- {
		if _, ok := applied[migrations[i].Version]; ok {
			targets = append(targets, migrations[i])
		}
	}

	var rolled []Migration
	for _, m := range targets {
		if IsBaseline(migrations, m.Version) && !allowBaseline {
			return rolled, fmt.Errorf(
				"拒绝回滚基线迁移 %s_%s：该操作会删除全部业务表；确需执行请设置 MIGRATE_ALLOW_BASELINE_DOWN=1",
				m.Version, m.Name)
		}
		if err := rollbackMigration(db, m); err != nil {
			return rolled, err
		}
		rolled = append(rolled, m)
	}

	return rolled, nil
}

// MigrationStatus 返回全部版本的应用状态。
func MigrationStatus(db *gorm.DB, dir string) ([]MigrationState, error) {
	migrations, err := LoadMigrations(dir)
	if err != nil {
		return nil, err
	}

	applied := make(map[string]appliedMigration)
	if db.Migrator().HasTable(&appliedMigration{}) {
		if applied, err = loadApplied(db); err != nil {
			return nil, err
		}
	}

	states := make([]MigrationState, 0, len(migrations))
	for _, m := range migrations {
		state := MigrationState{Migration: m}
		if row, ok := applied[m.Version]; ok {
			state.Applied = true
			state.AppliedAt = row.AppliedAt
			state.ChecksumMismatch = row.Checksum != m.Checksum
		}
		states = append(states, state)
	}

	return states, nil
}

// applyMigration 逐条执行 up 脚本，全部成功后才登记版本。
// MySQL 的 DDL 会隐式提交、无法用事务包裹，因此失败时不登记版本，
// 修复后重跑即可（脚本规范要求尽量幂等）。
func applyMigration(db *gorm.DB, m Migration) error {
	content, err := os.ReadFile(m.UpPath)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", m.UpPath, err)
	}

	start := time.Now()
	for _, stmt := range splitStatements(string(content)) {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("迁移 %s_%s 执行失败: %w\n语句: %s", m.Version, m.Name, err, truncate(stmt, 200))
		}
	}

	record := appliedMigration{
		Version:    m.Version,
		Name:       m.Name,
		Checksum:   m.Checksum,
		AppliedAt:  time.Now(),
		DurationMs: int(time.Since(start).Milliseconds()),
	}
	if err := db.Create(&record).Error; err != nil {
		return fmt.Errorf("登记迁移版本 %s 失败: %w", m.Version, err)
	}

	logging.GetLogger().Info("Migration applied",
		zap.String("version", m.Version),
		zap.String("name", m.Name),
		zap.Int("duration_ms", record.DurationMs))

	return nil
}

// rollbackMigration 执行 down 脚本并删除版本记录。
func rollbackMigration(db *gorm.DB, m Migration) error {
	content, err := os.ReadFile(m.DownPath)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", m.DownPath, err)
	}

	for _, stmt := range splitStatements(string(content)) {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("回滚 %s_%s 失败: %w\n语句: %s", m.Version, m.Name, err, truncate(stmt, 200))
		}
	}

	if err := db.Where("version = ?", m.Version).Delete(&appliedMigration{}).Error; err != nil {
		return fmt.Errorf("删除迁移版本记录 %s 失败: %w", m.Version, err)
	}

	logging.GetLogger().Info("Migration rolled back",
		zap.String("version", m.Version),
		zap.String("name", m.Name))

	return nil
}

// splitStatements 按「行首 -- 为注释、语句以分号结尾」切分脚本。
// 规范：迁移脚本不得在字符串字面量中出现分号，否则会被误切。
func splitStatements(script string) []string {
	var (
		statements []string
		current    strings.Builder
	)

	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		current.WriteString(line)
		current.WriteString("\n")

		if strings.HasSuffix(trimmed, ";") {
			if stmt := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(current.String()), ";")); stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	// 最后一条语句允许省略分号
	if rest := strings.TrimSpace(current.String()); rest != "" {
		statements = append(statements, strings.TrimSuffix(rest, ";"))
	}

	return statements
}

func ensureMigrationTable(db *gorm.DB) error {
	if db.Migrator().HasTable(&appliedMigration{}) {
		return nil
	}
	if err := db.Migrator().CreateTable(&appliedMigration{}); err != nil {
		return fmt.Errorf("创建 %s 表失败: %w", migrationTableName, err)
	}

	return nil
}

func loadApplied(db *gorm.DB) (map[string]appliedMigration, error) {
	var rows []appliedMigration
	if err := db.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", migrationTableName, err)
	}

	applied := make(map[string]appliedMigration, len(rows))
	for _, row := range rows {
		applied[row.Version] = row
	}

	return applied, nil
}

// validateChecksums 拒绝执行「已应用但文件被改动」的版本。
func validateChecksums(applied map[string]appliedMigration, migrations []Migration) error {
	for _, m := range migrations {
		row, ok := applied[m.Version]
		if !ok {
			continue
		}
		if row.Checksum != m.Checksum {
			return fmt.Errorf(
				"迁移 %s_%s 已被应用但脚本内容发生变化（checksum %s → %s）：请新增一个版本而不是修改历史脚本",
				m.Version, m.Name, truncate(row.Checksum, 12), truncate(m.Checksum, 12))
		}
	}

	return nil
}

func fileChecksum(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取 %s 失败: %w", path, err)
	}

	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
