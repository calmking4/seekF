package db

import (
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var GormDB *gorm.DB

func init() {
	conf := configs.GetConfig()
	user := conf.User
	password := conf.MysqlConfig.Password
	host := conf.MysqlConfig.Host
	port := conf.MysqlConfig.Port
	appName := conf.AppName
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, appName)
	// dsn := fmt.Sprintf("%s@unix(/var/run/mysqld/mysqld.sock)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, appName) //linux连接数据库
	var err error
	GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		zlog.Fatal(err.Error())
	}
}
