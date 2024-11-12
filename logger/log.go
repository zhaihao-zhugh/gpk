package log

import (

	// "github.com/mattn/go-colorable"

	// log "github.com/sirupsen/logrus"

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

var logLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
var logger *zap.Logger = zap.New(
	zapcore.NewCore(zapcore.NewConsoleEncoder(engineConfig), multipleWriter, logLevel),
	zap.AddCaller(),
	zap.AddCallerSkip(1),
)
var sugaredLogger *zap.SugaredLogger = logger.Sugar()

func NameEncoder(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(Colorize(loggerName, WhiteFg|BlackBg).String())
}

type Logger struct {
	*zap.SugaredLogger
}

func NewLogger(name string) *Logger {
	return &Logger{
		sugaredLogger.Named(name),
	}
}

func SetLogLevel(level string) {
	set_level, err := zap.ParseAtomicLevel(level)
	if err != nil {
		sugaredLogger.Error(err)
		return
	}
	logLevel = set_level
	logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewCore(zapcore.NewConsoleEncoder(engineConfig), multipleWriter, logLevel)
	}))
	sugaredLogger = logger.Sugar()
	// loglevel, err := zapcore.ParseLevel(level)
	//
	//	if err != nil {
	//		sugaredLogger.Error(err)
	//		return
	//	}
	//
	// logLevel.SetLevel(loglevel)
}

func (l *Logger) AddCallerSkip(i int) {
	l.SugaredLogger = l.SugaredLogger.WithOptions(zap.AddCallerSkip(i))
}

func (l *Logger) Debug(args ...interface{}) {
	l.SugaredLogger.Debug(args...)
}

func (l *Logger) Info(args ...interface{}) {
	l.SugaredLogger.Info(args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.SugaredLogger.Warn(args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.SugaredLogger.Error(args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.SugaredLogger.Fatal(args...)
}

func (l *Logger) Panic(args ...interface{}) {
	l.SugaredLogger.Panic(args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.SugaredLogger.Debugf(format, args...)
}

// Infof logs a message at level Info on the standard logger.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.SugaredLogger.Infof(format, args...)
}

// Warnf logs a message at level Warn on the standard logger.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.SugaredLogger.Warnf(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.SugaredLogger.Errorf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger.
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.SugaredLogger.Panicf(format, args...)
}

// Fatalf logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.SugaredLogger.Fatalf(format, args...)
}
