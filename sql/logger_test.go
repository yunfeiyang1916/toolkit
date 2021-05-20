package sql

import (
	"errors"
	"testing"
)

func TestParseSql_Result(t *testing.T) {
	type fields struct {
		Sql string
	}
	type wants struct {
		CurdType  string
		TableName string
		Err       error
	}

	sqlErr := "select * from on id = 1"

	sqlSelect1 := "select * from t1"
	sqlSelect2 := "select * from t1 a, t2 b where a.id = b.id"
	sqlSelect3 := "select * from t1 left join t2 on t1.id = t2.id"
	sqlSelect4 := "select * from (select * from t1) t2"
	sqlSelect5 := "select * from t1 where id = (select id from t2)"

	sqlInsert1 := "insert into t (id) values (1)"

	sqlUpdate1 := "update t1 set id = 1"
	sqlUpdate2 := "update t1 a, t2 b set a.id = 1"

	sqlDelete1 := "delete from t1 where id = 1"

	tests := []struct {
		name   string
		fields fields
		want   wants
	}{
		{"test_error", fields{sqlErr}, wants{"", "", errors.New("error")}},
		{"test_select_1", fields{sqlSelect1}, wants{"Select", "t1", nil}},
		{"test_select_2", fields{sqlSelect2}, wants{"Select", "t1_t2", nil}},
		{"test_select_3", fields{sqlSelect3}, wants{"Select", "t1_t2", nil}},
		{"test_select_4", fields{sqlSelect4}, wants{"Select", "t1", nil}},
		{"test_select_5", fields{sqlSelect5}, wants{"Select", "t1", nil}},

		{"test_insert_1", fields{sqlInsert1}, wants{"Insert", "t", nil}},

		{"test_update_1", fields{sqlUpdate1}, wants{"Update", "t1", nil}},
		{"test_update_2", fields{sqlUpdate2}, wants{"Update", "t1_t2", nil}},

		{"test_delete_1", fields{sqlDelete1}, wants{"Delete", "t1", nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if curdType, tableName, err := parseSql(tt.fields.Sql); curdType != tt.want.CurdType || tableName != tt.want.TableName ||
				(err == nil && tt.want.Err != nil) || (err != nil && tt.want.Err == nil) {
				t.Errorf("parseSql(tt.fields.Sql) return %s, %s, %v, want %s, %s, %v",
					curdType, tableName, err,
					tt.want.CurdType, tt.want.TableName, tt.want.Err)
			}
		})
	}
}
