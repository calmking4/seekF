package zlog

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"seekF-backend/internal/configs"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger = zap.NewNop()
var slowSQLLogger = zap.NewNop() // 慢SQL专用logger

// InitLogger 显式初始化日志，控制台输出到stderr以保留MCP的stdio协议。
func InitLogger() error {
	conf := configs.GetConfig()
	if conf == nil {
		return fmt.Errorf("配置未初始化，无法初始化日志")
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	// 设置日志记录中时间格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// 日志encoder还是JSONEncoder，把日志行格式化成JSON格式
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	logDir := conf.LogDir

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("创建日志目录失败: %v", err)
	}

	// 使用 file-rotatelogs 实现按天日志轮转
	fileWriteSyncer := getFileLogWriter(logDir, conf)
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), zapcore.DebugLevel),
		zapcore.NewCore(encoder, fileWriteSyncer, zapcore.DebugLevel),
	)
	logger = zap.New(core)

	// 初始化慢SQL专用logger
	slowSQLCore := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), zapcore.InfoLevel),
		zapcore.NewCore(encoder, getSlowSQLLogWriter(logDir, conf), zapcore.InfoLevel),
	)
	slowSQLLogger = zap.New(slowSQLCore)
	return nil
}

// getFileLogWriter 创建按天轮转的应用日志输出器。
func getFileLogWriter(logDir string, conf *configs.Config) (writeSyncer zapcore.WriteSyncer) {
	// 配置轮转参数
	maxAge := time.Duration(conf.MaxAge) * 24 * time.Hour        // 保留天数
	rotationTime := time.Duration(conf.RotationTime) * time.Hour // 轮转间隔

	// 生成带日期的文件名模式: ./logs/app.%Y%m%d.log
	pattern := filepath.Join(logDir, "app.%Y%m%d.log")

	writer, err := rotatelogs.New(
		pattern,
		rotatelogs.WithMaxAge(maxAge), // 清理超过 maxAge 的旧日志
		rotatelogs.WithRotationTime(rotationTime), // 轮转间隔
	)
	if err != nil {
		// 轮转日志初始化失败，回退到控制台，避免程序崩溃
		log.Printf("初始化日志轮转失败: %v，回退到仅输出到控制台", err)
		return zapcore.AddSync(os.Stderr)
	}

	return zapcore.AddSync(writer)
}

// getSlowSQLLogWriter 创建按天轮转的慢SQL日志输出器。
func getSlowSQLLogWriter(logDir string, conf *configs.Config) (writeSyncer zapcore.WriteSyncer) {
	// 配置轮转参数
	maxAge := time.Duration(conf.MaxAge) * 24 * time.Hour        // 保留天数
	rotationTime := time.Duration(conf.RotationTime) * time.Hour // 轮转间隔

	// 生成慢SQL日志文件名: ./logs/slowsql.%Y%m%d.log
	pattern := filepath.Join(logDir, "slowsql.%Y%m%d.log")

	writer, err := rotatelogs.New(
		pattern,
		rotatelogs.WithMaxAge(maxAge),
		rotatelogs.WithRotationTime(rotationTime),
	)
	if err != nil {
		log.Printf("初始化慢SQL日志轮转失败: %v，回退到仅输出到控制台", err)
		return zapcore.AddSync(os.Stderr)
	}

	return zapcore.AddSync(writer)
}

// getCallerInfoForLog 获得调用方的日志信息，包括函数名，文件名，行号
func getCallerInfoForLog() (callerFields []zap.Field) {
	pc, file, line, ok := runtime.Caller(2) // 回溯两层，拿到写日志的调用方的函数信息
	if !ok {
		return
	}
	funcName := runtime.FuncForPC(pc).Name()
	funcName = path.Base(funcName) // Base函数返回路径的最后一个元素，只保留函数名

	callerFields = append(callerFields, zap.String("func", funcName), zap.String("file", file), zap.Int("line", line))
	return
}

func Info(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Info(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Warn(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Fatal(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Debug(message, fields...)
}

// SlowSQL 记录慢SQL日志，输出到 slowsql.日期.log
func SlowSQL(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	slowSQLLogger.Info(message, fields...)
}
