package zlog

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"seekF-backend/internal/configs"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger
var logPath string

// 自动调用
func init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	// 设置日志记录中时间格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// 日志encoder还是JSONEncoder，把日志行格式化成JSON格式
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	conf := configs.GetConfig()
	logPath = conf.LogPath

	// 确保日志目录存在
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("创建日志目录失败: %v", err)
	}

	// 使用 file-rotatelogs 实现按天日志轮转
	fileWriteSyncer := getFileLogWriter()
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
		zapcore.NewCore(encoder, fileWriteSyncer, zapcore.DebugLevel),
	)
	logger = zap.New(core)
}

func getFileLogWriter() (writeSyncer zapcore.WriteSyncer) {
	conf := configs.GetConfig()

	// 配置轮转参数
	maxAge := time.Duration(conf.MaxAge) * 24 * time.Hour       // 保留天数
	rotationTime := time.Duration(conf.RotationTime) * time.Hour // 轮转间隔

	// 生成带日期的文件名模式: ./logs/app.log -> ./logs/app.%Y%m%d.log
	ext := filepath.Ext(logPath)
	pattern := strings.TrimSuffix(logPath, ext) + ".%Y%m%d" + ext

	writer, err := rotatelogs.New(
		pattern,
		rotatelogs.WithMaxAge(maxAge),         // 清理超过 maxAge 的旧文件
		rotatelogs.WithRotationTime(rotationTime), // 轮转间隔
	)
	if err != nil {
		// 轮转日志初始化失败，回退到 stdout，避免程序崩溃
		log.Printf("初始化日志轮转失败: %v，回退到仅输出到 stdout", err)
		return zapcore.AddSync(os.Stdout)
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
