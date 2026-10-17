package database

import (
	"reflect"
	"testing"
)

type schemaCheckSample struct {
	ID         uint   `gorm:"primaryKey"`
	UUID       string `gorm:"column:external_id"`
	Ignored    string `gorm:"-"`
	NoOverride string
}

func TestModelTableColumns(t *testing.T) {
	table, columns, err := modelTableColumns(&schemaCheckSample{})
	if err != nil {
		t.Fatalf("modelTableColumns 失败: %v", err)
	}
	if table != "schema_check_samples" {
		t.Fatalf("表名解析错误: %s", table)
	}

	want := []string{"id", "external_id", "no_override"}
	if !reflect.DeepEqual(columns, want) {
		t.Fatalf("列名解析错误: %v, 期望 %v", columns, want)
	}
}

func TestMissingColumns(t *testing.T) {
	got := missingColumns([]string{"id", "uuid", "description"}, []string{"id", "uuid"})
	if !reflect.DeepEqual(got, []string{"description"}) {
		t.Fatalf("缺失列计算错误: %v", got)
	}

	if got := missingColumns([]string{"id"}, []string{"legacy", "id"}); len(got) != 0 {
		t.Fatalf("不应把多余列算作缺失: %v", got)
	}
}

func TestSchemaIssueDrift(t *testing.T) {
	cases := []struct {
		name  string
		issue SchemaIssue
		want  bool
	}{
		{"missing table", SchemaIssue{MissingTable: true}, true},
		{"missing columns", SchemaIssue{MissingColumns: []string{"uuid"}}, true},
		{"extra columns only", SchemaIssue{ExtraColumns: []string{"legacy"}}, false},
		{"clean", SchemaIssue{}, false},
	}
	for _, tc := range cases {
		if got := tc.issue.Drift(); got != tc.want {
			t.Errorf("%s: Drift() = %v, 期望 %v", tc.name, got, tc.want)
		}
	}
}
