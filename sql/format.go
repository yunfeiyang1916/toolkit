package sql

import (
	"fmt"
	"strings"
)

var (
	// Map from format's placeholders to printf verbs
	phfs map[string]string

	// Default format of log message
	defaultFmt = "(%[1]s) [%.2[2]fms] %[3]s [%[4]d rows affected or returned] %[6]s"
)

type SqlInfo struct {
	FileWithLine string
	Duration     float64
	Sql          string
	Rows         int64
	RowsSimple   int64
	ErrorMsg     string

	Format string
}

// init pkg
func init() {
	initFormatPlaceholders()
}

// Initializes the map of placeholders
func initFormatPlaceholders() {
	phfs = map[string]string{
		"%{file_with_line}": "(%[1]s)",
		"%{duration}":       "[%.2[2]fms]",
		"%{sql}":            "%[3]s",
		"%{rows}":           "[%[4]d rows affected or returned]",
		"%{rows_simple}":    "[%[5]d]",
		"%{error_msg}":      "%[6]s",
	}
}

func (r *SqlInfo) Output() string {
	msg := fmt.Sprintf(r.Format,
		r.FileWithLine, // %[1] // %{file_with_line}
		r.Duration,     // %[2] // %{duration}
		r.Sql,          // %[3] // %{sql}
		r.Rows,         // %[4] // %{rows}
		r.RowsSimple,   // %[5] // %{rows_simple}
		r.ErrorMsg,     // %[6] // %{error_msg}
	)
	// Ignore printf errors if len(args) > len(verbs)
	if i := strings.LastIndex(msg, "%!(EXTRA"); i != -1 {
		return msg[:i]
	}
	return msg
}

func (r *SqlInfo) SetCustomFormat(format string) {
	r.Format = parseFormat(format)
}

// Analyze and represent format string as printf format string and time format
func parseFormat(format string) (msgfmt string) {
	if len(format) < 6 /* (len of "%{sql} */ {
		return defaultFmt
	}
	idx := strings.IndexRune(format, '%')
	for idx != -1 {
		msgfmt += format[:idx]
		format = format[idx:]
		if len(format) > 2 {
			if format[1] == '{' {
				// end of curr verb pos
				if jdx := strings.IndexRune(format, '}'); jdx != -1 {
					// next verb pos
					idx = strings.Index(format[1:], "%{")
					// incorrect verb found ("...%{wefwef ...") but after
					// this, new verb (maybe) exists ("...%{inv %{verb}...")
					if idx != -1 && idx < jdx {
						msgfmt += "%%"
						format = format[1:]
						continue
					}
					// get verb and arg
					verb := ph2verb(format[:jdx+1])
					msgfmt += verb

					format = format[jdx+1:]
				} else {
					format = format[1:]
				}
			} else {
				msgfmt += "%%"
				format = format[1:]
			}
		}
		idx = strings.IndexRune(format, '%')
	}
	msgfmt += format
	return
}

func ph2verb(ph string) (verb string) {
	n := len(ph)
	if n < 4 {
		return ``
	}
	if ph[0] != '%' || ph[1] != '{' || ph[n-1] != '}' {
		return ``
	}
	return phfs[ph]
}
