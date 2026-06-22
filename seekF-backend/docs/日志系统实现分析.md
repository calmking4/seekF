# 日志系统实现分析

## 1. 功能概述

seekF 项目使用 **Zap** 作为日志库，实现了结构化日志记录、日志轮转、慢 SQL 日志分离等功能。

## 2. 技术架构

```
┌─────────────────────────────────────────────────────────────────┐
│                       应用层                                     │
│  zlog.Info() / zlog.Error() / zlog.Warn() / zlog.Debug()       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    zlog_util.go                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Zap Logger (主日志)                                      │   │
│  │  ┌─────────────┐  ┌─────────────┐                       │   │
│  │  │   Stdout    │  │  文件输出    │                       │   │
│  │  │  (控制台)    │  │ (按天轮转)  │                       │   │
│  │  └─────────────┘  └─────────────┘                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  slowSQLLogger (慢SQL专用)                                │   │
│  │  ┌─────────────┐  ┌─────────────┐                       │   │
│  │  │   Stdout    │  │  文件输出    │                       │   │
│  │  │  (控制台)    │  │ slowsql.log │                       │   │
│  │  └─────────────┘  └─────────────┘                       │   │
│  └─────────────────────────────────────────────────────────┘   │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    文件系统                                       │
│  ./logs/                                                        │
│  ├── app.20240101.log      # 应用日志                            │
│  ├── app.20240102.log                                           │
│  ├── slowsql.20240101.log  # 慢SQL日志                           │
│  └── slowsql.20240102.log                                       │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 核心实现

### 3.1 核心文件

| 文件路径 | 职责 |
|---------|------|
| `internal/pkg/zlog/zlog_util.go` | 日志工具封装 |
| `internal/pkg/db/mysql_util.go` | GORM 慢 SQL 日志适配 |
| `internal/configs/configs.go` | 日志配置定义 |
| `config/config.toml` | 日志配置文件 |

### 3.2 依赖库

```go
import (
    "go.uber.org/zap"                    // 高性能结构化日志库
    "go.uber.org/zap/zapcore"            // Zap 核心组件
    "github.com/lestrrat-go/file-rotatelogs"  // 日志轮转
)
```

## 4. 日志初始化

### 4.1 配置结构

```go
type LogConfig struct {
    LogDir       string `toml:"logDir"`       // 日志目录，如 "./logs"
    MaxAge       int    `toml:"maxAge"`       // 保留旧日志的最大天数，默认 30
    RotationTime int    `toml:"rotationTime"` // 轮转间隔（小时），默认 24
}
```

### 4.2 配置文件 (`config.toml`)

```toml
[logConfig]
logDir = "./logs"       # 日志目录
maxAge = 30             # 保留旧日志的最大天数
rotationTime = 24       # 轮转间隔（小时）
```

### 4.3 初始化流程

```go
var logger *zap.Logger
var slowSQLLogger *zap.Logger // 慢SQL专用logger

func init() {
    // 1. 配置编码器
    encoderConfig := zap.NewProductionEncoderConfig()
    encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder  // ISO8601 时间格式
    encoder := zapcore.NewJSONEncoder(encoderConfig)        // JSON 格式

    // 2. 加载配置
    conf := configs.GetConfig()
    logDir := conf.LogDir

    // 3. 确保日志目录存在
    if err := os.MkdirAll(logDir, 0755); err != nil {
        log.Printf("创建日志目录失败: %v", err)
    }

    // 4. 初始化主日志 (同时输出到 stdout 和文件)
    fileWriteSyncer := getFileLogWriter(logDir, conf)
    core := zapcore.NewTee(
        zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
        zapcore.NewCore(encoder, fileWriteSyncer, zapcore.DebugLevel),
    )
    logger = zap.New(core)

    // 5. 初始化慢SQL专用logger
    slowSQLCore := zapcore.NewTee(
        zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
        zapcore.NewCore(encoder, getSlowSQLLogWriter(logDir, conf), zapcore.InfoLevel),
    )
    slowSQLLogger = zap.New(slowSQLCore)
}
```

## 5. 日志轮转

### 5.1 主日志轮转

```go
func getFileLogWriter(logDir string, conf *configs.Config) (writeSyncer zapcore.WriteSyncer) {
    maxAge := time.Duration(conf.MaxAge) * 24 * time.Hour        // 保留天数
    rotationTime := time.Duration(conf.RotationTime) * time.Hour // 轮转间隔

    // 文件名模式: ./logs/app.%Y%m%d.log
    pattern := filepath.Join(logDir, "app.%Y%m%d.log")

    writer, err := rotatelogs.New(
        pattern,
        rotatelogs.WithMaxAge(maxAge),           // 清理超过 maxAge 的旧日志
        rotatelogs.WithRotationTime(rotationTime), // 轮转间隔
    )
    if err != nil {
        log.Printf("初始化日志轮转失败: %v，回退到仅输出到 stdout", err)
        return zapcore.AddSync(os.Stdout)
    }

    return zapcore.AddSync(writer)
}
```

### 5.2 慢 SQL 日志轮转

```go
func getSlowSQLLogWriter(logDir string, conf *configs.Config) (writeSyncer zapcore.WriteSyncer) {
    maxAge := time.Duration(conf.MaxAge) * 24 * time.Hour
    rotationTime := time.Duration(conf.RotationTime) * time.Hour

    // 慢SQL日志文件名: ./logs/slowsql.%Y%m%d.log
    pattern := filepath.Join(logDir, "slowsql.%Y%m%d.log")

    writer, err := rotatelogs.New(
        pattern,
        rotatelogs.WithMaxAge(maxAge),
        rotatelogs.WithRotationTime(rotationTime),
    )
    if err != nil {
        log.Printf("初始化慢SQL日志轮转失败: %v，回退到仅输出到 stdout", err)
        return zapcore.AddSync(os.Stdout)
    }

    return zapcore.AddSync(writer)
}
```

## 6. 日志 API

### 6.1 日志级别函数

```go
// Info 信息日志
func Info(message string, fields ...zap.Field) {
    callerFields := getCallerInfoForLog()
    fields = append(fields, callerFields...)
    logger.Info(message, fields...)
}

// Warn 警告日志
func Warn(message string, fields ...zap.Field) {
    callerFields := getCallerInfoForLog()
    fields = append(fields, callerFields...)
    logger.Warn(message, fields...)
}

// Error 错误日志
func Error(message string, fields ...zap.Field) {
    callerFields := getCallerInfoForLog()
    fields = append(fields, callerFields...)
    logger.Error(message, fields...)
}

// Fatal 致命错误日志（会调用 os.Exit(1)）
func Fatal(message string, fields ...zap.Field) {
    callerFields := getCallerInfoForLog()
    fields = append(fields, callerFields...)
    logger.Fatal(message, fields...)
}

// Debug 调试日志
func Debug(message string, fields ...zap.Field) {
    callerFields := getCallerInfoForLog()
    fields = append(fields, callerFields...)
    logger.Debug(message, fields...)
}

// SlowSQL 慢SQL日志（输出到单独文件）
func SlowSQL(message string, fields ...zap.Field) {
    callerFields := getCallerInfoForLog()
    fields = append(fields, callerFields...)
    slowSQLLogger.Info(message, fields...)
}
```

### 6.2 调用方信息追踪

```go
// getCallerInfoForLog 获得调用方的日志信息，包括函数名，文件名，行号
func getCallerInfoForLog() (callerFields []zap.Field) {
    pc, file, line, ok := runtime.Caller(2) // 回溯两层，拿到写日志的调用方的函数信息
    if !ok {
        return
    }
    funcName := runtime.FuncForPC(pc).Name()
    funcName = path.Base(funcName) // Base函数返回路径的最后一个元素，只保留函数名

    callerFields = append(callerFields, 
        zap.String("func", funcName), 
        zap.String("file", file), 
        zap.Int("line", line),
    )
    return
}
```

## 7. 慢 SQL 日志集成

### 7.1 GORM 适配器

```go
// GormWriter 适配器，将GORM日志转发到zlog
type GormWriter struct{}

func (w *GormWriter) Printf(format string, args ...interface{}) {
    zlog.SlowSQL(fmt.Sprintf(format, args...))
}
```

### 7.2 GORM 配置

```go
func init() {
    conf := configs.GetConfig()
    // ... DSN 配置 ...

    // 配置GORM日志 - 使用zlog的慢SQL专用logger
    newLogger := logger.New(
        &GormWriter{},
        logger.Config{
            SlowThreshold:             200 * time.Millisecond, // 慢SQL阈值
            LogLevel:                  logger.Warn,            // Warn级别会记录慢SQL
            IgnoreRecordNotFoundError: true,                   // 忽略RecordNotFound错误
            Colorful:                  false,
        },
    )

    GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
        SkipDefaultTransaction: true,
        Logger:                 newLogger,
    })
}
```

## 8. 日志输出格式

### 8.1 JSON 格式示例

```json
{
    "level": "info",
    "ts": "2024-01-01T12:00:00.000+0800",
    "msg": "用户登录成功",
    "func": "auth_service.Login",
    "file": "D:/develop/GoRelated/project/seekF/seekF-backend/internal/services/user_service/auth_service.go",
    "line": 42
}
```

### 8.2 慢 SQL 日志示例

```json
{
    "level": "info",
    "ts": "2024-01-01T12:00:00.000+0800",
    "msg": "[205.12ms] SELECT * FROM user_info WHERE id = ?",
    "func": "user_dao.GetUserInfo",
    "file": "D:/develop/GoRelated/project/seekF/seekF-backend/internal/dao/user_dao/user_dao.go",
    "line": 15
}
```

## 9. 使用示例

### 9.1 基本使用

```go
import "seekF-backend/internal/pkg/zlog"

// 信息日志
zlog.Info("用户登录成功: " + userId)

// 错误日志
zlog.Error("创建AI会话失败: " + err.Error())

// 警告日志
zlog.Warn("用户会话即将过期: " + sessionId)

// 调试日志
zlog.Debug("请求参数: " + string(jsonBytes))
```

### 9.2 带额外字段

```go
zlog.Info("消息发送成功",
    zap.String("messageId", msgId),
    zap.String("senderId", senderId),
    zap.String("receiverId", receiverId),
)
```

### 9.3 错误处理模式

```go
if err != nil {
    zlog.Error("操作失败: " + err.Error())
    return err
}
```

## 10. 日志规范

### 10.1 语言要求

**所有日志信息和错误信息必须使用中文**，包括：
- `zlog.Info/Error/Warn/Debug` 日志
- `fmt.Errorf` 错误信息
- `errors.New` 错误信息

```go
// ✅ 正确
zlog.Error("创建AI会话失败: " + err.Error())
zlog.Info("用户登录成功: " + userId)
fmt.Errorf("用户不存在")

// ❌ 错误
zlog.Error("create AI session failed: " + err.Error())
zlog.Info("user login success: " + userId)
fmt.Errorf("user not found")
```

### 10.2 日志级别使用场景

| 级别 | 使用场景 |
|------|----------|
| `Debug` | 开发调试信息，生产环境不输出 |
| `Info` | 关键业务流程节点、操作成功 |
| `Warn` | 非致命异常、降级处理 |
| `Error` | 操作失败、需要关注的异常 |
| `Fatal` | 致命错误，程序无法继续运行 |

### 10.3 日志内容要求

1. **包含上下文**：日志应包含关键业务标识（如 userId、sessionId）
2. **错误链**：记录原始错误信息 `": " + err.Error()`
3. **简洁明了**：一句话描述清楚，避免冗余

```go
// ✅ 推荐
zlog.Error("保存用户消息失败: " + err.Error())
zlog.Info("删除AI会话成功: " + sessionId)

// ❌ 避免
zlog.Error("error")
zlog.Info("success")
```

## 11. 配置项说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `logDir` | string | `./logs` | 日志文件存储目录 |
| `maxAge` | int | `30` | 旧日志保留天数 |
| `rotationTime` | int | `24` | 日志轮转间隔（小时） |
| `SlowThreshold` | duration | `200ms` | GORM 慢 SQL 阈值 |

## 12. 文件结构

```
seekF-backend/
├── logs/                          # 日志目录
│   ├── app.20240101.log          # 应用日志（按天）
│   ├── app.20240102.log
│   ├── slowsql.20240101.log      # 慢SQL日志（按天）
│   └── slowsql.20240102.log
├── internal/
│   ├── pkg/
│   │   └── zlog/
│   │       └── zlog_util.go      # 日志工具
│   └── pkg/
│       └── db/
│           └── mysql_util.go     # GORM适配
└── config/
    └── config.toml                # 日志配置
```

## 13. 注意事项

1. **性能考虑**: Zap 是高性能日志库，生产环境建议使用 JSON 格式
2. **磁盘空间**: 合理设置 `maxAge`，避免日志文件占用过多磁盘空间
3. **日志分离**: 慢 SQL 日志单独存储，便于性能分析
4. **调用方信息**: 自动记录函数名、文件名、行号，便于排查问题
5. **Fatal 级别**: 使用 `Fatal` 会调用 `os.Exit(1)`，谨慎使用
6. **日志目录**: 程序启动时自动创建，无需手动创建
