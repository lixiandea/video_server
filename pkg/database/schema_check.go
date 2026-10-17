package database

import (
	"fmt"
	"sort"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// SchemaIssue 描述某个模型对应表与库实际结构的差异。
// MissingTable / MissingColumns 会让代码在运行期报错；ExtraColumns 只是历史遗留提示。
type SchemaIssue struct {
	Table          string
	MissingTable   bool
	MissingColumns []string
	ExtraColumns   []string
}

// Drift 表示该差异会导致模型无法正常读写数据库。
func (i SchemaIssue) Drift() bool {
	return i.MissingTable || len(i.MissingColumns) > 0
}

// CheckModelSchema 对比 GORM 模型定义与实际库结构，返回有差异的表。
// 只检查「模型 → 库」方向：库里多出来的列不视为错误（旧库可能保留废弃结构）。
func CheckModelSchema(db *gorm.DB, values ...interface{}) ([]SchemaIssue, error) {
	issues := make([]SchemaIssue, 0, len(values))
	for _, value := range values {
		table, modelColumns, err := modelTableColumns(value)
		if err != nil {
			return nil, err
		}

		issue := SchemaIssue{Table: table}
		if !db.Migrator().HasTable(value) {
			issue.MissingTable = true
			issues = append(issues, issue)
			continue
		}

		columnTypes, err := db.Migrator().ColumnTypes(value)
		if err != nil {
			return nil, fmt.Errorf("读取 %s 列信息失败: %w", table, err)
		}
		dbColumns := make([]string, 0, len(columnTypes))
		for _, columnType := range columnTypes {
			dbColumns = append(dbColumns, columnType.Name())
		}

		issue.MissingColumns = missingColumns(modelColumns, dbColumns)
		issue.ExtraColumns = missingColumns(dbColumns, modelColumns)
		issues = append(issues, issue)
	}

	return issues, nil
}

// modelTableColumns 解析模型对应的表名与列名（含 gorm:"column:..." 重命名）。
func modelTableColumns(value interface{}) (string, []string, error) {
	parsed, err := schema.Parse(value, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return "", nil, fmt.Errorf("解析模型 %T 失败: %w", value, err)
	}

	columns := make([]string, 0, len(parsed.Fields))
	for _, field := range parsed.Fields {
		if field.DBName != "" {
			columns = append(columns, field.DBName)
		}
	}
	return parsed.Table, columns, nil
}

// missingColumns 返回 expected 中存在、actual 中缺失的列名（升序）。
func missingColumns(expected, actual []string) []string {
	present := make(map[string]struct{}, len(actual))
	for _, column := range actual {
		present[column] = struct{}{}
	}

	missing := make([]string, 0)
	for _, column := range expected {
		if _, ok := present[column]; !ok {
			missing = append(missing, column)
		}
	}
	sort.Strings(missing)
	return missing
}
