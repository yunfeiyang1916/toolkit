package sql

import "testing"

func TestSqlInfo_Output(t *testing.T) {
	type fields struct {
		FileWithLine string
		Duration     float64
		Sql          string
		Rows         int64
		RowsSimple   int64
		ErrorMsg     string
		Format       string
	}
	f := "%{file_with_line} %{duration} %{sql} %{rows} %{error_msg}"
	f2 := "%{file_with_line} %{duration} %{sql} %{rows}"
	f3 := "%{duration} %{sql} %{rows}"
	f4 := "%{duration} (query %{sql}) %{rows}"
	f5 := "%{duration} %{rows_simple} (query %{sql})"
	f6 := "%s %{file_with_line} %{duration} %{sql} %{rows} %{error_msg}"
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"test1", fields{"/a.go", 1.0, "select * from test", 0, 0, "invalid connection", f}, "(/a.go) [1.00ms] select * from test [0 rows affected or returned] invalid connection"},
		{"test2", fields{"/a.go", 1.0, "select * from test", 0, 0, "", f2}, "(/a.go) [1.00ms] select * from test [0 rows affected or returned]"},
		{"test3", fields{"/a.go", 1.0, "select * from test", 0, 0, "", f3}, "[1.00ms] select * from test [0 rows affected or returned]"},
		{"test4", fields{"/a.go", 1.0, "select * from test", 0, 0, "", f4}, "[1.00ms] (query select * from test) [0 rows affected or returned]"},
		{"test5", fields{"/a.go", 2.0, "select * from test", 0, 0, "", f5}, "[2.00ms] [0] (query select * from test)"},
		{"test6", fields{"/a.go", 2.0, "select * from test", 0, 0, "", f6}, "%s (/a.go) [2.00ms] select * from test [0 rows affected or returned] "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &SqlInfo{
				FileWithLine: tt.fields.FileWithLine,
				Duration:     tt.fields.Duration,
				Sql:          tt.fields.Sql,
				Rows:         tt.fields.Rows,
				RowsSimple:   tt.fields.RowsSimple,
				ErrorMsg:     tt.fields.ErrorMsg,
			}
			r.SetCustomFormat(tt.fields.Format)
			if got := r.Output(); got != tt.want {
				t.Errorf("SqlInfo.Output() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSqlInfo_SetCustomFormat(t *testing.T) {
	type fields struct {
		FileWithLine string
		Duration     float64
		Sql          string
		Rows         int64
		RowsSimple   int64
		ErrorMsg     string
		Format       string
	}
	type args struct {
		format string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"test", fields{"/a.go", 2.0, "select * from test", 0, 0, "", "%{sql}"}, args{"%{sql}"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &SqlInfo{
				FileWithLine: tt.fields.FileWithLine,
				Duration:     tt.fields.Duration,
				Sql:          tt.fields.Sql,
				Rows:         tt.fields.Rows,
				RowsSimple:   tt.fields.RowsSimple,
				ErrorMsg:     tt.fields.ErrorMsg,
				Format:       tt.fields.Format,
			}
			r.SetCustomFormat(tt.args.format)
			if got := r.Format; got != parseFormat(tt.args.format) {
				t.Errorf("SqlInfo.Format = %q, want %q", got, tt.args.format)
			}
		})
	}
}

func Test_parseFormat(t *testing.T) {
	type args struct {
		format string
	}
	tests := []struct {
		name       string
		args       args
		wantMsgfmt string
	}{
		{"test1", args{"%{sql}"}, "%[3]s"},
		{"test2", args{"%{sql} %{sql}"}, "%[3]s %[3]s"},
		{"test3", args{"%{sql} %{duration}"}, "%[3]s [%.2[2]fms]"},
		{"test4", args{"%{sql} %{rows_simple} %{error_msg}"}, "%[3]s [%[5]d] %[6]s"},
		{"test6", args{"%{file_with_line} %{duration} %{sql} %{rows} %{error_msg}"}, "(%[1]s) [%.2[2]fms] %[3]s [%[4]d rows affected or returned] %[6]s"},
		{"test7", args{"DEBUG %{file_with_line} %{duration} %{sql} %{rows} %{error_msg}"}, "DEBUG (%[1]s) [%.2[2]fms] %[3]s [%[4]d rows affected or returned] %[6]s"},
		{"test8", args{"%s %{file_with_line} %{duration} %{sql} %{rows} %{error_msg}"}, "%%s (%[1]s) [%.2[2]fms] %[3]s [%[4]d rows affected or returned] %[6]s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotMsgfmt := parseFormat(tt.args.format); gotMsgfmt != tt.wantMsgfmt {
				t.Errorf("parseFormat() = %v, want %v", gotMsgfmt, tt.wantMsgfmt)
			}
		})
	}
}

func Test_ph2verb(t *testing.T) {
	type args struct {
		ph string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"test", args{"%{sql}"}, "%[3]s"},
		{"test2", args{"%{}"}, ""},
		{"test3", args{"%{s}"}, ""},
		{"test4", args{"%{"}, ""},
		{"test5", args{"{}"}, ""},
		{"test6", args{"%|}"}, ""},
		{"test7", args{"%{sql} %{}"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVerb := ph2verb(tt.args.ph)
			if gotVerb != tt.want {
				t.Errorf("ph2verb() gotVerb = %v, want %v", gotVerb, tt.want)
			}
		})
	}
}
