package logging

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/yunfeiyang1916/toolkit/metrics"
	"github.com/yunfeiyang1916/toolkit/rolling"
	"github.com/yunfeiyang1916/toolkit/tool/gid"
)

// Logger 日志记录器
type Logger struct {
	*zap.SugaredLogger
	path string
	// 目录
	dir          string
	rolling      rolling.RollingFormat
	rollingFiles []io.Writer
	loglevel     zap.AtomicLevel
	prefix       string
	encoderCfg   zapcore.EncoderConfig
	callSkip     int
}

var defaultEncoderConfig = zapcore.EncoderConfig{
	CallerKey:      "caller",
	StacktraceKey:  "stack",
	LineEnding:     zapcore.DefaultLineEnding,
	TimeKey:        "time",
	MessageKey:     "msg",
	LevelKey:       "level",
	NameKey:        "logger",
	EncodeCaller:   zapcore.ShortCallerEncoder,
	EncodeLevel:    zapcore.CapitalColorLevelEncoder,
	EncodeTime:     MilliSecondTimeEncoder,
	EncodeDuration: zapcore.StringDurationEncoder,
	EncodeName:     zapcore.FullNameEncoder,
}

var (
	_defaultLogger  *Logger
	_jsonDataLogger *Logger
	hostIP          string
)

const (
	_jsonDataTaskKey = "service_name"
	localIP          = "local_ip"
	uniqID           = "uniq_id"
)

// Logger name for default loggers
const (
	DefaultLoggerName = "_default"
	SlowLoggerName    = "_slow"
	GenLoggerName     = "_gen"
	CrashLoggerName   = "_crash"
	BalanceLoggerName = "_balance"
)

func init() {
	_defaultLogger = New()
	logs[DefaultLoggerName] = _defaultLogger
	logs[SlowLoggerName] = slowlog
	logs[GenLoggerName] = genlog
	logs[CrashLoggerName] = crashlog
	logs[BalanceLoggerName] = balancelog
	if host, _ := gid.HostIP(); host != nil {
		hostIP = host.To4().String()
	}
}

var logs = map[string]*Logger{}
var logsMtx sync.RWMutex

func Log(name string) *Logger {
	logsMtx.RLock()
	defer logsMtx.RUnlock()
	return logs[name]
}
func New() *Logger {
	cfg := defaultEncoderConfig
	lvl := zap.NewAtomicLevelAt(zap.DebugLevel)
	return &Logger{
		SugaredLogger: zap.New(zapcore.NewCore(NewConsoleEncoder(&cfg), zapcore.Lock(os.Stderr), lvl)).WithOptions(zap.AddCaller(), zap.AddCallerSkip(1)).Sugar(),
		path:          "",
		dir:           "",
		rolling:       rolling.DailyRolling,
		rollingFiles:  nil,
		loglevel:      lvl,
		prefix:        "",
		encoderCfg:    cfg,
	}
}

// NewJSON build json data format logger
func NewJSON(path string, r rolling.RollingFormat) (*Logger, error) {
	cfg := defaultEncoderConfig
	cfg.LevelKey = ""
	cfg.MessageKey = "topic"
	lvl := zap.NewAtomicLevelAt(zap.DebugLevel)
	rollFile, err := rolling.NewRollingFile(path, r)
	if err != nil {
		return nil, err
	}
	return &Logger{
		SugaredLogger: zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(cfg), rollFile, lvl)).WithOptions(zap.AddCaller(), zap.AddCallerSkip(1)).Sugar(),
		path:          path,
		dir:           "",
		rolling:       rolling.DailyRolling,
		rollingFiles:  []io.Writer{rollFile},
		loglevel:      lvl,
		prefix:        "",
		encoderCfg:    cfg,
	}, nil
}

// InitData logger
func InitData(path string, rolling rolling.RollingFormat) error {
	if _jsonDataLogger != nil {
		return nil
	}
	l, err := NewJSON(path, rolling)
	if err != nil {
		return err
	}
	_jsonDataLogger = l
	return nil
}

// InitData logger
func InitDataWithKey(path string, rolling rolling.RollingFormat, task string) error {
	err := InitData(path, rolling)
	if err != nil {
		return err
	}

	_jsonDataLogger.SugaredLogger = _jsonDataLogger.SugaredLogger.With(_jsonDataTaskKey, task)
	return nil
}

func NewLogger(opt *Options, paths ...string) *Logger {
	opt.init()
	var res *Logger
	if len(paths) == 0 {
		res = New()
		normalizeLoggerWithOption(res, opt)
		return res
	}
	for _, path := range paths {
		logger := New()
		normalizeLoggerWithOption(logger, opt)
		logger.SetOutputByName(path)
		if res == nil {
			res = logger
		}
		s := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		logsMtx.Lock()
		logs[s] = logger
		logsMtx.Unlock()
	}
	return res
}

func (l *Logger) SetOutput(out io.Writer) {
	l.SugaredLogger = zap.New(zapcore.NewCore(NewConsoleEncoder(&l.encoderCfg), zapcore.Lock(zapcore.AddSync(out)), zap.DebugLevel)).WithOptions(zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
	l.SugaredLogger.Named(l.prefix)
}

func (l *Logger) GetOutput() io.Writer {
	return nil
}

func (l *Logger) SetColors(color bool) {
	if !color {
		l.encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	} else {
		l.encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	}
}

func (l *Logger) With(args ...interface{}) *Logger {
	return &Logger{SugaredLogger: l.SugaredLogger.With(args...).Desugar().WithOptions(zap.AddCallerSkip(0)).Sugar()}
}

func (l *Logger) SetLogPrefix(prefix string) {
	l.prefix = prefix
	l.SugaredLogger.Named(prefix)
}

func (l *Logger) SetRotateByDay() {
	l.rolling = rolling.DailyRolling
	l.refreshRotate()
}

func (l *Logger) refreshRotate() {
	for _, w := range l.rollingFiles {
		r, ok := w.(*rolling.RollingFile)
		if ok {
			r.SetRolling(l.rolling)
		}
	}
}

func (l *Logger) SetRotateByHour() {
	l.rolling = rolling.HourlyRolling
	l.refreshRotate()
}

func (l *Logger) SetRotateBySecond() {
	l.rolling = rolling.SecondlyRolling
	l.refreshRotate()
}

func (l *Logger) SetFlags(flags int) {
	if flags == 0 {
		l.encoderCfg = zapcore.EncoderConfig{
			CallerKey:     "",
			StacktraceKey: "",
			LineEnding:    zapcore.DefaultLineEnding,
			TimeKey:       "",
			MessageKey:    "msg",
			LevelKey:      "",
			NameKey:       "",
		}
	}
}

func (l *Logger) SetHighlighting(highlighting bool) {
	l.SetColors(highlighting)
}

func (l *Logger) SetPrintLevel(printLevel bool) {
	if !printLevel {
		l.encoderCfg.LevelKey = ""
	} else {
		l.encoderCfg.LevelKey = "level"
	}
}

func (l *Logger) SetTimeFmt(fmt string) error {
	l.encoderCfg.EncodeTime = NewTimeEncoder(fmt)
	return nil
}

func (l *Logger) SetOutputByName(path string) error {
	if l.path == path {
		return nil
	}
	l.closeFiles()
	l.path = path
	l.dir = ""
	debugFile, err := rolling.NewRollingFile(path, l.rolling)
	if err != nil {
		return err
	}
	core := zapcore.NewTee(
		zapcore.NewCore(NewConsoleEncoder(&l.encoderCfg), debugFile, l.loglevel),
	)
	l.rollingFiles = []io.Writer{debugFile}
	l.SugaredLogger = zap.New(core).WithOptions(zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
	l.SugaredLogger.Named(l.prefix)
	return nil
}

func (l *Logger) closeFiles() {
	for _, w := range l.rollingFiles {
		r, ok := w.(*rolling.RollingFile)
		if ok {
			r.Close()
		}
	}
	l.rollingFiles = nil
}

func (l *Logger) SetOutputPath(path string) error {
	if l.dir == path {
		return nil
	}
	l.closeFiles()
	l.path = ""
	l.dir = path
	debugFile, err := rolling.NewRollingFile(path+"/debug.log", l.rolling)
	if err != nil {
		return err
	}
	infoFile, err := rolling.NewRollingFile(path+"/info.log", l.rolling)
	if err != nil {
		return err
	}
	errorFile, err := rolling.NewRollingFile(path+"/error.log", l.rolling)
	if err != nil {
		return err
	}
	debugLogEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		if l.loglevel.Level() > zapcore.DebugLevel {
			return false
		}
		return l.loglevel.Enabled(lvl)
	})
	errorlogEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel
	})
	infologEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return l.loglevel.Level() <= zapcore.InfoLevel && zapcore.InfoLevel == lvl
	})
	core := zapcore.NewTee(
		zapcore.NewCore(NewConsoleEncoder(&l.encoderCfg), debugFile, debugLogEnabler),
		zapcore.NewCore(NewConsoleEncoder(&l.encoderCfg), infoFile, infologEnabler),
		zapcore.NewCore(NewConsoleEncoder(&l.encoderCfg), errorFile, errorlogEnabler),
	)
	l.rollingFiles = []io.Writer{debugFile, infoFile, errorFile}
	l.SugaredLogger = zap.New(core).WithOptions(zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
	l.SugaredLogger.Named(l.prefix)
	return nil
}

func (l *Logger) SetLevel(level int) {
	l.loglevel.SetLevel(zapcore.Level(level))
}

func (l *Logger) SetLevelByString(level string) {
	l.loglevel.SetLevel(stringToLogLevel(level))
}

func (l *Logger) Logger() *log.Logger {
	stdLogger := log.New(logWriter{
		logFunc: func() func(msg string, fileds ...interface{}) {
			// fmt.Printf("logFunc %v.\n", l)
			logger := &Logger{SugaredLogger: l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(3)).Sugar()}
			return logger.Debugf
		},
	}, "", 0)
	return stdLogger
}

func GetLogger() *log.Logger {
	return _defaultLogger.Logger()
}

func SetRotateByHour() {
	_defaultLogger.SetRotateByHour()
}

func SetRotateByDay() {
	_defaultLogger.SetRotateByDay()
}

func SetLevelByString(level string) {
	_defaultLogger.SetLevelByString(level)
}

func SetOutputPath(dir string) {
	_defaultLogger.SetOutputPath(dir)
}

func SetOutputByName(path string) {
	_defaultLogger.SetOutputByName(path)
}

func Debug(v ...interface{}) {
	_defaultLogger.Debug(v...)
}

func Info(v ...interface{}) {
	_defaultLogger.Info(v...)
}

func Warn(v ...interface{}) {
	_defaultLogger.Warn(v...)
}

func Warning(v ...interface{}) {
	_defaultLogger.Warn(v...)
}

func Error(v ...interface{}) {
	_defaultLogger.Error(v...)
}

func Fatal(v ...interface{}) {
	_defaultLogger.Fatal(v...)
}

func Debugf(format string, v ...interface{}) {
	_defaultLogger.Debugf(format, v...)
}

func Infof(format string, v ...interface{}) {
	_defaultLogger.Infof(format, v...)
}

func Warnf(format string, v ...interface{}) {
	_defaultLogger.Warnf(format, v...)
}

func Warningf(format string, v ...interface{}) {
	_defaultLogger.Warnf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	_defaultLogger.Errorf(format, v...)
}

func Fatalf(format string, v ...interface{}) {
	_defaultLogger.Fatalf(format, v...)
}

func With(args ...interface{}) *Logger {
	return &Logger{SugaredLogger: _defaultLogger.SugaredLogger.With(args...).Desugar().WithOptions(zap.AddCallerSkip(-1)).Sugar()}
}

func For(ctx context.Context, args ...interface{}) *Logger {
	tid := extraTraceID(ctx)
	var fields []interface{}
	if len(tid) != 0 {
		fields = make([]interface{}, 0, len(args)+2)
		fields = append(fields, traceIDKey, extraTraceID(ctx))
	} else {
		fields = make([]interface{}, 0, len(args))
	}
	fields = append(fields, args...)
	return &Logger{SugaredLogger: _defaultLogger.With(fields...).Desugar().WithOptions(zap.AddCallerSkip(-1)).Sugar()}
}

func Debugw(msg string, keysAndValues ...interface{}) {
	_defaultLogger.Debugw(msg, keysAndValues...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	_defaultLogger.Infow(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	_defaultLogger.Errorw(msg, keysAndValues...)
}

func Warningw(msg string, keysAndValues ...interface{}) {
	_defaultLogger.Warnw(msg, keysAndValues...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	_defaultLogger.Warnw(msg, keysAndValues...)
}

func DataWith(args ...interface{}) *Logger {
	return &Logger{SugaredLogger: _defaultLogger.With(args...).Desugar().WithOptions(zap.AddCallerSkip(-1)).Sugar()}
}

func DataLog(topic string, keysAndValues ...interface{}) {
	kvLen := len(keysAndValues) + 4
	values := make([]interface{}, kvLen)
	prefixValues := []interface{}{localIP, hostIP, uniqID, strconv.FormatUint(gid.New(), 16)}
	copy(values[:4], prefixValues)
	copy(values[4:], keysAndValues)
	_jsonDataLogger.Debugw(topic, values...)
	metrics.Meter("datalog.Counter", 1, "topic", topic)
}

func DataLogCtx(ctx context.Context, topic string, keysAndValues ...interface{}) {
	for _, cb := range contextList {
		k, v := cb(ctx)
		if len(k) != 0 && len(v) != 0 {
			keysAndValues = append(keysAndValues, k, v)
		}
	}
	kvLen := len(keysAndValues) + 4
	values := make([]interface{}, kvLen)
	prefixValues := []interface{}{localIP, hostIP, uniqID, strconv.FormatUint(gid.New(), 16)}
	copy(values[:4], prefixValues)
	copy(values[4:], keysAndValues)
	_jsonDataLogger.Debugw(topic, values...)
	metrics.Meter("datalog.Counter", 1, "topic", topic)
}

// Sync all log data
func Sync() {
	logsMtx.RLock()
	defer logsMtx.RUnlock()
	for _, l := range logs {
		l.Sync()
	}
	if _jsonDataLogger != nil {
		_jsonDataLogger.Sync()
	}
}

func stringToLogLevel(level string) zapcore.Level {
	switch level {
	case "fatal":
		return zap.FatalLevel
	case "error":
		return zap.ErrorLevel
	case "warn":
		return zap.WarnLevel
	case "warning":
		return zap.WarnLevel
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	}
	return zap.DebugLevel
}

func normalizeLoggerWithOption(res *Logger, opt *Options) {
	if opt.DisableColors {
		res.SetColors(false)
	}
	if opt.DisableLevel {
		res.encoderCfg.LevelKey = ""
	}
	if opt.DisableFullTimestamp {
		res.encoderCfg.TimeKey = ""
	}
	if opt.Level != "" {
		res.SetLevelByString(opt.Level)
	}
	if opt.Rolling != "" {
		res.rolling = rolling.RollingFormat(opt.Rolling)
	}
}

type logWriter struct {
	logFunc func() func(msg string, fileds ...interface{})
}

func (l logWriter) Write(p []byte) (int, error) {
	p = bytes.TrimSpace(p)
	if l.logFunc != nil {
		l.logFunc()(string(p))
	}
	return len(p), nil
}
