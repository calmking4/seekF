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

func (w *GormWriter) Printf(format string, args ...interface{}) {
	zlog.SlowSQL(fmt.Sprintf(format, args...))
}

func init() {
	conf := configs.GetConfig()
	user := conf.User
	password := conf.MysqlConfig.Password
	host := conf.MysqlConfig.Host
	port := conf.MysqlConfig.Port
	appName := conf.AppName
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, appName)
	// dsn := fmt.Sprintf("%s@unix(/var/run/mysqld/mysqld.sock)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, appName) //linux连接数据库

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

	var err error
	GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true, // 禁用默认事务，提高性能
		Logger:                 newLogger,
	})
	if err != nil {
		zlog.Fatal(err.Error())
	}
}
