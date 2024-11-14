package logger

import (

	// "github.com/mattn/go-colorable"

	// log "github.com/sirupsen/logrus"

	"os"
	"path"
	"time"

	zaprotatelogs "github.com/lestrrat-go/file-rotatelogs"
	. "github.com/logrusorgru/aurora/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogConfig struct {
	LogLevel string `json:"loglevel" yaml:"loglevel"`
	LogFile  string `json:"logfile" yaml:"logfile"`
}

var engineConfig = zapcore.EncoderConfig{
	// Keys can be anything except the empty string.
	TimeKey:        "T",
	LevelKey:       "L",
	NameKey:        "N",
	CallerKey:      "C",
	FunctionKey:    zapcore.OmitKey,
	MessageKey:     "M",
	StacktraceKey:  "S",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.CapitalColorLevelEncoder,
	EncodeTime:     zapcore.TimeEncoderOfLayout("2006/01/02 15:04:05"),
	EncodeDuration: zapcore.StringDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
	EncodeName:     NameEncoder,
	// NewReflectedEncoder: func(w io.Writer) zapcore.ReflectedEncoder {
	// 	return yaml.NewEncoder(w)
	// },
}

func NameEncoder(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(Colorize(loggerName, WhiteFg|BlackBg).String())
}

func NewLogger(name string) *zap.SugaredLogger {
	return logger.Named(name)
}

var logger *zap.SugaredLogger
var logLevel = zap.NewAtomicLevelAt(zap.DebugLevel)

func Init(config *LogConfig) {
	logger = zap.New(
		zapcore.NewCore(zapcore.NewConsoleEncoder(engineConfig), multipleWriter, logLevel),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	).Sugar()

	SetLogLevel(config.LogLevel)
	SetSavePath(config.LogFile)
}

func SetLogLevel(level string) {
	if level == "" {
		return
	}
	set_level, err := zap.ParseAtomicLevel(level)
	if err != nil {
		logger.Error(err)
		return
	}
	logLevel = set_level
	logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewCore(zapcore.NewConsoleEncoder(engineConfig), multipleWriter, logLevel)
	}))
}

// writer
var multipleWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout))

func SetSavePath(file string) {
	if file == "" {
		return
	}
	fileWriter, err := zaprotatelogs.New(
		path.Join(file, "%Y-%m-%d.log"), //日志的路径和文件名
		// zaprotatelogs.WithLinkName(CONFIG.Zap.LinkName), // 生成软链，指向最新日志文件
		zaprotatelogs.WithMaxAge(time.Duration(7*24)*time.Hour), //保存日期的时间
		zaprotatelogs.WithRotationTime(24*time.Hour),            //每天分割一次日志
	)
	if err != nil {
		logger.Error(err)
		return
	}
	multipleWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(fileWriter))

	logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewCore(zapcore.NewConsoleEncoder(engineConfig), multipleWriter, logLevel)
	}))
}

func Debug(args ...any) {
	logger.Debug(args...)
}

func Info(args ...any) {
	logger.Info(args...)
}

func Warn(args ...any) {
	logger.Warn(args...)
}

func Error(args ...any) {
	logger.Error(args...)
}

func Panic(args ...any) {
	logger.Panic(args...)
}

func Fatal(args ...any) {
	logger.Fatal(args...)
}

func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger.
func Panicf(format string, args ...interface{}) {
	logger.Panicf(format, args...)
}

// Fatalf logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}
