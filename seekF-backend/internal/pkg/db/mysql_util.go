package db

import (
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var GormDB *gorm.DB

// GormWriter 适配器，将GORM日志转发到zlog
type GormWriter struct{}

// Printf 将GORM日志交给慢SQL日志输出器。
func (w *GormWriter) Printf(format string, args ...interface{}) {
	zlog.SlowSQL(fmt.Sprintf(format, args...))
}

// InitMySQL 显式创建并检查MySQL连接池，失败由启动入口处理。
func InitMySQL() error {
	if GormDB != nil {
		return nil
	}
	conf := configs.GetConfig()
	if conf == nil {
		return fmt.Errorf("配置未初始化，无法初始化MySQL")
	}
	user := conf.User
	password := conf.MysqlConfig.Password
	host := conf.MysqlConfig.Host
	port := conf.MysqlConfig.Port
	appName := conf.AppName
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s", user, password, host, port, appName)
	// dsn := fmt.Sprintf("%s@unix(/var/run/mysqld/mysqld.sock)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s", user, appName) //linux连接数据库

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

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true, // 禁用默认事务，提高性能
		Logger:                 newLogger,
	})
	if err != nil {
		if database != nil {
			if pool, poolErr := database.DB(); poolErr == nil {
				_ = pool.Close()
			}
		}
		return fmt.Errorf("初始化MySQL失败: %w", err)
	}
	GormDB = database
	zlog.Info("MySQL初始化完成")
	return nil
}
