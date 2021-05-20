package sql

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/yunfeiyang1916/toolkit/go-tls"

	"github.com/go-sql-driver/mysql"
	"github.com/xwb1989/sqlparser"
	log "github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/metrics"
)

var (
	sqlRegexp                = regexp.MustCompile(`\?`)
	numericPlaceHolderRegexp = regexp.MustCompile(`\$\d+`)
)

var (
	sqlGlobalLogger = &GlobalLogger{print: log.Info}
)

func newGlobalLogger(statLevel string, isMaster int, database, format string) *GlobalLogger {
	return &GlobalLogger{
		print:     log.Info,
		IsMaster:  isMaster,
		Database:  database,
		StatLevel: statLevel,
		LogFormat: format,
	}
}

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func (l *GlobalLogger) setSqlStat(code int, source, sql string, costTime time.Duration) {
	curdType, tableName, err := parseSql(sql)
	if (err != nil || tableName == "") && code == 0 { // 执行正确仍然解析sql失败就不再记录stat日志
		return
	}
	caller := funcName(source)
	if caller == "" {
		return
	}

	var name string
	switch l.StatLevel {
	case "database", "":
		if err == nil {
			name = fmt.Sprintf("DB.%s.%s", curdType, l.Database)
		} else {
			name = fmt.Sprintf("DB.ERROR.%s", l.Database)
		}
		metrics.TimerDuration(name, costTime, metrics.TagCode, code, "clientag", "sql", "ismaster", l.IsMaster)
	case "table":
		if err == nil {
			name = fmt.Sprintf("TABLE.%s.%s.%s", curdType, l.Database, tableName)
		} else {
			name = fmt.Sprintf("TABLE.ERROR.%s", l.Database)
		}
		metrics.TimerDuration(name, costTime, metrics.TagCode, code, "clientag", "sql", "ismaster", l.IsMaster)
	case "sql":
		if err == nil {
			name = fmt.Sprintf("SQL.%s.%s.%s.%s", curdType, l.Database, tableName, caller)
		} else {
			name = fmt.Sprintf("SQL.ERROR.%s.%s", l.Database, caller)
		}
		metrics.TimerDuration(name, costTime, metrics.TagCode, code, "clientag", "sql", "ismaster", l.IsMaster, "caller", caller)
	default:
		return
	}
}

func parseSql(sql string) (curdType string, tableName string, err error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return
	}
	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		curdType = "Select"
		tableName = extractTableNames(stmt.From)
	case *sqlparser.Insert:
		curdType = "Insert"
		tableName = parseTableStr(stmt.Table.Name.String())
	case *sqlparser.Update:
		curdType = "Update"
		tableName = extractTableNames(stmt.TableExprs)
	case *sqlparser.Delete:
		curdType = "Delete"
		tableName = extractTableNames(stmt.TableExprs)
	default:
		err = errors.New("sql parser result empty")
	}

	return
}

func extractTableNames(tbExprs sqlparser.TableExprs) string {
	var res []string
	for _, item := range tbExprs {
		if tb := extractTableName(item); tb != "" {
			res = append(res, tb)
		}
	}
	if len(res) > 0 {
		return strings.Join(res, "_")
	} else {
		return ""
	}
}

func extractTableName(tbExpr sqlparser.TableExpr) string {
	switch i := tbExpr.(type) {
	case *sqlparser.AliasedTableExpr:
		switch tb := i.Expr.(type) {
		case sqlparser.TableName:
			return tb.Name.String()
		case *sqlparser.Subquery:
			if s, ok := tb.Select.(*sqlparser.Select); ok {
				return extractTableNames(s.From)
			} else {
				return directPrintTableName(i)
			}
		default:
			return directPrintTableName(i)
		}
	case *sqlparser.JoinTableExpr:
		l := extractTableName(i.LeftExpr)
		r := extractTableName(i.RightExpr)
		if l != "" && r != "" {
			return strings.Join([]string{l, r}, "_")
		} else {
			return ""
		}
	case *sqlparser.ParenTableExpr:
		var tbs []string
		for _, item := range i.Exprs {
			if tb := extractTableName(item); tb != "" {
				tbs = append(tbs, tb)
			}
		}
		if len(tbs) > 0 {
			return strings.Join(tbs, "_")
		} else {
			return ""
		}
	}

	return ""
}

func directPrintTableName(tbExpr sqlparser.TableExpr) string {
	if tbExpr == nil {
		return ""
	}
	b := sqlparser.NewTrackedBuffer(nil)
	tbExpr.Format(b)
	return parseTableStr(b.String())
}

func parseTableStr(tables string) string {
	tables = strings.Replace(tables, " ", "_", -1)
	tables = strings.Replace(tables, ",", "", -1)
	return tables
}

func (l *GlobalLogger) logFormatter(values ...interface{}) (messages []interface{}) {
	if len(values) > 1 {
		var (
			sql             string
			formattedValues []string
			level           = values[0]
		)

		info := SqlInfo{
			FileWithLine: fmt.Sprintf("%s", values[1]),
		}

		if level == "sql" {
			// duration
			info.Duration = float64(values[2].(time.Duration).Nanoseconds()/1e4) / 100.0

			// sql
			for _, value := range values[4].([]interface{}) {
				indirectValue := reflect.Indirect(reflect.ValueOf(value))
				if indirectValue.IsValid() {
					value = indirectValue.Interface()
					if t, ok := value.(time.Time); ok {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
					} else if b, ok := value.([]byte); ok {
						if str := string(b); isPrintable(str) {
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", str))
						} else {
							formattedValues = append(formattedValues, "'<binary>'")
						}
					} else if r, ok := value.(driver.Valuer); ok {
						if value, err := r.Value(); err == nil && value != nil {
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
						} else {
							formattedValues = append(formattedValues, "NULL")
						}
					} else {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
					}
				} else {
					formattedValues = append(formattedValues, "NULL")
				}
			}

			if numericPlaceHolderRegexp.MatchString(values[3].(string)) {
				sql = values[3].(string)
				for index, value := range formattedValues {
					placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
					sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
				}
			} else {
				formattedValuesLength := len(formattedValues)
				for index, value := range sqlRegexp.Split(values[3].(string), -1) {
					sql += value
					if index < formattedValuesLength {
						sql += formattedValues[index]
					}
				}
			}

			info.Sql = sql
			info.Rows = values[5].(int64)
			info.RowsSimple = values[5].(int64)

			if val, exists := tls.Get("error_message"); exists {
				errorMessage := val.([]interface{})
				info.ErrorMsg = fmt.Sprintf("%v", errorMessage[0])
				l.setSqlStat(getCodeFromError(errorMessage[0]), fmt.Sprintf("%v", values[1]), values[3].(string), values[2].(time.Duration))
				tls.Delete("error_message")
			} else {
				l.setSqlStat(0, fmt.Sprintf("%v", values[1]), values[3].(string), values[2].(time.Duration))
			}

			// print info
			info.SetCustomFormat(l.LogFormat)
			messages = []interface{}{info.Output()}
		} else {
			messages = []interface{}{fmt.Sprintf("(%v) ", values[1])}
			messages = append(messages, values[2:]...)
		}
	}

	return
}

type GlobalLogger struct {
	mu        sync.RWMutex
	print     func(...interface{})
	IsMaster  int
	Database  string
	StatLevel string
	LogFormat string
}

func (l *GlobalLogger) Print(message ...interface{}) {
	if len(message) > 1 {
		if message[0] == "log" {
			tls.Set("error_message", message[2:])
			return
		}
	}
	message = l.logFormatter(message...)
	if len(message) >= 1 && l.print != nil {
		l.mu.RLock()
		f := l.print
		l.mu.RUnlock()
		if f != nil {
			f(message...)
		}
	}
}

func (l *GlobalLogger) setPrint(print func(...interface{})) {
	l.mu.Lock()
	l.print = print
	l.mu.Unlock()
}

// SetLoggerFunc设置logger
func SetLoggerFunc(print func(...interface{})) {
	sqlGlobalLogger.setPrint(print)
}

var errCodeMap = map[error]int{
	mysql.ErrInvalidConn:       220,
	mysql.ErrMalformPkt:        221,
	mysql.ErrNoTLS:             222,
	mysql.ErrCleartextPassword: 223,
	mysql.ErrNativePassword:    224,
	mysql.ErrOldPassword:       225,
	mysql.ErrUnknownPlugin:     226,
	mysql.ErrOldProtocol:       227,
	mysql.ErrPktSync:           228,
	mysql.ErrPktSyncMul:        229,
	mysql.ErrPktTooLarge:       230,
	mysql.ErrBusyBuffer:        231,
}

func getCodeFromError(e interface{}) (code int) {
	switch err := e.(type) {
	case *mysql.MySQLError:
		code = int(err.Number)
	case error:
		if _, ok := errCodeMap[err]; ok {
			code = errCodeMap[err]
		} else {
			code = 241
		}
	default:
		code = 241
	}
	return
}
