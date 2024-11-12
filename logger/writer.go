package log

import (
	"os"
	"path"
	"time"

	zaprotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// type MultipleWriter struct {
// 	io.Writer // 默认输出到标准输出
// 	sync.Map  // 用于存储多个输出
// }

// func (m *MultipleWriter) Write(p []byte) (n int, err error) {
// 	n, err = m.Writer.Write(p)
// 	m.Range(func(key, value any) bool {
// 		if _, err := key.(io.Writer).Write(p); err != nil {
// 			m.Delete(key)
// 		}
// 		return true
// 	})
// 	return
// }

// func (m *MultipleWriter) Add(writer io.Writer) {
// 	m.Map.Store(writer, struct{}{})
// }

// func AddWriter(writer io.Writer) {
// 	multipleWriter.Add(writer)
// }
// func DeleteWriter(writer io.Writer) {
// 	multipleWriter.Delete(writer)
// }

var multipleWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout))

func SetSavePath(file string) {
	fileWriter, err := zaprotatelogs.New(
		path.Join(file, "%Y-%m-%d.log"), //日志的路径和文件名
		// zaprotatelogs.WithLinkName(CONFIG.Zap.LinkName), // 生成软链，指向最新日志文件
		zaprotatelogs.WithMaxAge(time.Duration(7*24)*time.Hour), //保存日期的时间
		zaprotatelogs.WithRotationTime(24*time.Hour),            //每天分割一次日志
	)
	if err != nil {
		sugaredLogger.Error(err)
		return
	}
	multipleWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(fileWriter))

	logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewCore(zapcore.NewConsoleEncoder(engineConfig), multipleWriter, logLevel)
	}))
	sugaredLogger = logger.Sugar()
}
