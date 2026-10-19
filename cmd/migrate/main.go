// cmd/migrate 是数据库结构变更的唯一入口。
//
//	migrate up               应用所有未执行的迁移
//	migrate down --steps N   回滚最近 N 个版本（破坏性操作，需 --yes）
//	migrate status           查看各版本状态（存在 checksum 不一致时返回非 0）
//
// 连接参数取自 DB_* 环境变量；迁移目录默认 ./migrations，可用 MIGRATIONS_DIR 覆盖。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/lixiandea/video_server/internal/config"
	"github.com/lixiandea/video_server/internal/models"
	"github.com/lixiandea/video_server/pkg/database"
	"github.com/lixiandea/video_server/pkg/logging"
)

const defaultMigrationsDir = "migrations"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	if err := logging.InitLogger("info", "stdout"); err != nil {
		fmt.Fprintln(os.Stderr, "初始化日志失败:", err)
		os.Exit(1)
	}
	defer logging.Sync()

	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = defaultMigrationsDir
	}

	cfg := config.LoadConfig()

	var err error
	switch os.Args[1] {
	case "up":
		err = runUp(cfg, dir)
	case "down":
		err = runDown(cfg, dir, os.Args[2:])
	case "status":
		err = runStatus(cfg, dir)
	case "check":
		err = runCheck(cfg)
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "未知子命令 %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "❌ 迁移失败:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `用法: migrate <子命令> [参数]

子命令:
  up                       应用所有未执行的迁移
  down [--steps N] --yes   回滚最近 N 个版本（默认 1），需 --yes 确认
  status                   查看各版本状态
  check                    校验模型定义与库结构是否一致

环境变量:
  DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME  数据库连接
  MIGRATIONS_DIR                               迁移目录（默认 ./migrations）
  MIGRATE_ALLOW_BASELINE_DOWN=1                允许回滚基线迁移（会删除全部业务表）
`)
}

func runUp(cfg *config.Config, dir string) error {
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		return err
	}

	applied, err := database.MigrateUp(db, dir)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		fmt.Println("no pending migrations")
		return nil
	}
	for _, m := range applied {
		fmt.Printf("✅ applied %s_%s\n", m.Version, m.Name)
	}

	return nil
}

func runDown(cfg *config.Config, dir string, args []string) error {
	fs := flag.NewFlagSet("down", flag.ExitOnError)
	steps := fs.Int("steps", 1, "回滚的版本数量")
	yes := fs.Bool("yes", false, "确认执行破坏性回滚")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*yes {
		return fmt.Errorf("回滚会修改数据库结构，请加 --yes 确认")
	}

	db, err := database.Connect(&cfg.Database)
	if err != nil {
		return err
	}

	allowBaseline := os.Getenv("MIGRATE_ALLOW_BASELINE_DOWN") == "1"
	rolled, err := database.MigrateDown(db, dir, *steps, allowBaseline)
	if err != nil {
		return err
	}

	if len(rolled) == 0 {
		fmt.Println("nothing to roll back")
		return nil
	}
	for _, m := range rolled {
		fmt.Printf("↩️  rolled back %s_%s\n", m.Version, m.Name)
	}

	return nil
}

func runStatus(cfg *config.Config, dir string) error {
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		return err
	}

	states, err := database.MigrationStatus(db, dir)
	if err != nil {
		return err
	}

	mismatched := 0
	fmt.Printf("%-8s %-34s %-18s %s\n", "VERSION", "NAME", "STATUS", "APPLIED AT")
	for _, state := range states {
		status := "pending"
		appliedAt := "-"
		switch {
		case state.ChecksumMismatch:
			status = "checksum-mismatch"
			appliedAt = state.AppliedAt.Format("2006-01-02 15:04:05")
			mismatched++
		case state.Applied:
			status = "applied"
			appliedAt = state.AppliedAt.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%-8s %-34s %-18s %s\n", state.Version, state.Name, status, appliedAt)
	}

	if mismatched > 0 {
		return fmt.Errorf("%d 个已应用迁移的脚本内容发生变化，请新增版本而不是修改历史脚本", mismatched)
	}

	return nil
}

func runCheck(cfg *config.Config) error {
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		return err
	}

	modelsToCheck := []interface{}{
		&models.User{},
		&models.Video{},
		&models.VideoLike{},
		&models.Comment{},
		&models.Session{},
		&models.VideoDeletionRecord{},
		&models.AuditLog{},
	}
	issues, err := database.CheckModelSchema(db, modelsToCheck...)
	if err != nil {
		return err
	}

	drift := 0
	for _, issue := range issues {
		switch {
		case issue.MissingTable:
			fmt.Printf("❌ %s: 表不存在\n", issue.Table)
			drift++
		case len(issue.MissingColumns) > 0:
			fmt.Printf("❌ %s: 缺少列 %s\n", issue.Table, strings.Join(issue.MissingColumns, ", "))
			drift++
		}
		if len(issue.ExtraColumns) > 0 {
			fmt.Printf("ℹ️  %s: 多出列 %s（历史遗留，不影响运行）\n", issue.Table, strings.Join(issue.ExtraColumns, ", "))
		}
	}

	if drift > 0 {
		return fmt.Errorf("%d 张表的库结构落后于模型定义，请新增迁移补齐", drift)
	}
	fmt.Printf("✅ 模型与库结构一致（%d 个模型）\n", len(modelsToCheck))
	return nil
}
