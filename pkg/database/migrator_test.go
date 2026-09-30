package database

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMigration(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("写入 %s 失败: %v", name, err)
	}
}

func TestLoadMigrationsOrdersAndHashes(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "0002_add_index.up.sql", "ALTER TABLE t ADD INDEX i (c);\n")
	writeMigration(t, dir, "0002_add_index.down.sql", "ALTER TABLE t DROP INDEX i;\n")
	writeMigration(t, dir, "0001_init_schema.up.sql", "CREATE TABLE IF NOT EXISTS t (id INT);\n")
	writeMigration(t, dir, "0001_init_schema.down.sql", "DROP TABLE IF EXISTS t;\n")

	migrations, err := LoadMigrations(dir)
	if err != nil {
		t.Fatalf("LoadMigrations 失败: %v", err)
	}
	if len(migrations) != 2 {
		t.Fatalf("期望 2 个版本，实际 %d", len(migrations))
	}
	if migrations[0].Version != "0001" || migrations[1].Version != "0002" {
		t.Fatalf("版本未按升序排列: %s, %s", migrations[0].Version, migrations[1].Version)
	}
	if migrations[0].Name != "init_schema" {
		t.Fatalf("版本名解析错误: %s", migrations[0].Name)
	}

	sum := sha256.Sum256([]byte("CREATE TABLE IF NOT EXISTS t (id INT);\n"))
	if migrations[0].Checksum != hex.EncodeToString(sum[:]) {
		t.Fatalf("checksum 计算错误: %s", migrations[0].Checksum)
	}
	if !IsBaseline(migrations, "0001") || IsBaseline(migrations, "0002") {
		t.Fatal("IsBaseline 判定错误")
	}
}

func TestLoadMigrationsRejectsIncompletePairs(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "0001_init_schema.up.sql", "SELECT 1;\n")

	_, err := LoadMigrations(dir)
	if err == nil || !strings.Contains(err.Error(), "成对") {
		t.Fatalf("缺少 down 脚本时应报错，实际: %v", err)
	}
}

func TestLoadMigrationsRejectsInvalidNames(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "init.sql", "SELECT 1;\n")

	if _, err := LoadMigrations(dir); err == nil {
		t.Fatal("非法文件名应当报错")
	}
}

func TestLoadMigrationsRejectsDescriptionMismatch(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "0001_init_schema.up.sql", "SELECT 1;\n")
	writeMigration(t, dir, "0001_other_name.down.sql", "SELECT 1;\n")

	if _, err := LoadMigrations(dir); err == nil {
		t.Fatal("同一版本的 up/down 描述不一致时应当报错")
	}
}

func TestSplitStatements(t *testing.T) {
	script := `-- 头部注释
CREATE TABLE IF NOT EXISTS a (
	id INT
);

-- 中间注释
ALTER TABLE a ADD COLUMN b INT;

SET @sql = IF(EXISTS(SELECT 1), 'SELECT 1', 'DO 0');
`
	statements := splitStatements(script)
	if len(statements) != 3 {
		t.Fatalf("期望 3 条语句，实际 %d: %#v", len(statements), statements)
	}
	if !strings.HasPrefix(statements[0], "CREATE TABLE") || !strings.Contains(statements[0], "\n") {
		t.Fatalf("多行语句未被完整保留: %q", statements[0])
	}
	if statements[1] != "ALTER TABLE a ADD COLUMN b INT" {
		t.Fatalf("语句切分错误: %q", statements[1])
	}
	if !strings.HasPrefix(statements[2], "SET @sql") {
		t.Fatalf("末条无分号语句未保留: %q", statements[2])
	}
}

func TestSplitStatementsIgnoresCommentsAndBlankLines(t *testing.T) {
	script := "-- 只有注释\n\n   \n-- 另一行注释\n"
	if got := splitStatements(script); len(got) != 0 {
		t.Fatalf("注释与空行不应产生语句，实际: %#v", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("abcdef", 10); got != "abcdef" {
		t.Fatalf("短字符串不应被截断: %q", got)
	}
	if got := truncate("abcdef", 3); got != "abc..." {
		t.Fatalf("长字符串截断错误: %q", got)
	}
}

func TestSelectRollbackTargetsValidatesBaselineUpfront(t *testing.T) {
	migrations := []Migration{
		{Version: "0001", Name: "init_schema"},
		{Version: "0002", Name: "dedupe"},
		{Version: "0003", Name: "add_column"},
	}
	applied := map[string]appliedMigration{
		"0001": {Version: "0001"},
		"0002": {Version: "0002"},
		"0003": {Version: "0003"},
	}

	// 不含基线：正常选出最近 N 个已应用版本（倒序）
	targets, err := selectRollbackTargets(migrations, applied, 2, false)
	if err != nil {
		t.Fatalf("不含基线时不应报错: %v", err)
	}
	if len(targets) != 2 || targets[0].Version != "0003" || targets[1].Version != "0002" {
		t.Fatalf("回滚目标选择错误: %#v", targets)
	}

	// 含基线且未显式允许：必须在执行前整体拒绝，且不返回任何目标
	targets, err = selectRollbackTargets(migrations, applied, 3, false)
	if err == nil || !strings.Contains(err.Error(), "基线") {
		t.Fatalf("含基线时应拒绝，实际: %v", err)
	}
	if len(targets) != 0 {
		t.Fatalf("拒绝时不应返回任何待执行目标，实际: %#v", targets)
	}

	// 显式允许后可包含基线
	targets, err = selectRollbackTargets(migrations, applied, 3, true)
	if err != nil {
		t.Fatalf("显式允许基线时不应报错: %v", err)
	}
	if len(targets) != 3 || targets[2].Version != "0001" {
		t.Fatalf("显式允许基线时目标错误: %#v", targets)
	}

	// 未应用的版本不进入目标
	targets, err = selectRollbackTargets(migrations, map[string]appliedMigration{"0002": {Version: "0002"}}, 5, false)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(targets) != 1 || targets[0].Version != "0002" {
		t.Fatalf("只应回滚已应用版本: %#v", targets)
	}
}
