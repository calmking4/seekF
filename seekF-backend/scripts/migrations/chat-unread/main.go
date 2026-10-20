package main

import (
	"fmt"
	"log"

	"seekF-backend/internal/configs"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// main 升级私聊阅读进度，历史消息仅在首次迁移时初始化已读并核查中文注释。
func main() {
	conf := configs.GetConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", conf.User, conf.MysqlConfig.Password, conf.MysqlConfig.Host, conf.MysqlConfig.Port, conf.AppName)
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接迁移数据库失败: ", err)
	}
	var count int64
	if err := database.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'session' AND column_name = 'last_read_message_id'").Scan(&count).Error; err != nil {
		log.Fatal("核查阅读字段失败: ", err)
	}
	if count == 0 {
		// -1作为迁移中间态，异常中断后可继续初始化，不覆盖已经正常保存的进度。
		if err := database.Exec("ALTER TABLE session ADD COLUMN last_read_message_id BIGINT NOT NULL DEFAULT -1 COMMENT '本人已读入站私信的消息主键上界，0表示尚未阅读'").Error; err != nil {
			log.Fatal("新增阅读进度字段失败: ", err)
		}
	}
	if err := database.Exec(`UPDATE session AS s SET last_read_message_id = CASE WHEN s.receive_id LIKE 'U%' THEN
		(SELECT COALESCE(MAX(m.id), 0) FROM message AS m WHERE m.receive_id = s.send_id AND m.send_id = s.receive_id AND m.type IN (0, 2))
		ELSE 0 END WHERE s.last_read_message_id = -1`).Error; err != nil {
		log.Fatal("初始化历史私信阅读进度失败: ", err)
	}
	if err := database.Exec("ALTER TABLE session MODIFY COLUMN last_read_message_id BIGINT NOT NULL DEFAULT 0 COMMENT '本人已读入站私信的消息主键上界，0表示尚未阅读', COMMENT = '用户私聊及AI会话信息表'").Error; err != nil {
		log.Fatal("更新阅读字段及表注释失败: ", err)
	}
	var column struct {
		Comment      string
		DefaultValue string
	}
	if err := database.Raw("SELECT column_comment AS comment, column_default AS default_value FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'session' AND column_name = 'last_read_message_id'").Scan(&column).Error; err != nil {
		log.Fatal("验证阅读字段注释失败: ", err)
	}
	var tableComment string
	if err := database.Raw("SELECT table_comment FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'session'").Scan(&tableComment).Error; err != nil {
		log.Fatal("验证会话表注释失败: ", err)
	}
	if column.Comment != "本人已读入站私信的消息主键上界，0表示尚未阅读" || column.DefaultValue != "0" || tableComment != "用户私聊及AI会话信息表" {
		log.Fatal("阅读字段默认值或数据库中文注释核查失败")
	}
	fmt.Println("私聊阅读进度迁移完成，字段默认值及中文表、字段注释验证通过")
}
