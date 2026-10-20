package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	redisclient "github.com/redis/go-redis/v9"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/models"
)

// main 检查或执行私聊功能精简迁移，不启动任何业务服务。
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "私聊迁移失败: "+err.Error())
		os.Exit(1)
	}
}

// run 从后端工作目录加载连接配置并检查实际结构。
func run() error {
	apply := flag.Bool("apply", false, "备份后执行数据库和缓存清理")
	verifyOnly := flag.Bool("verify", false, "只验证迁移结果")
	backupRoot := flag.String("backup-root", "../plan/private-chat/backups", "备份目录")
	flag.Parse()
	cfg := configs.GetConfig()
	connection := mysql.NewConfig()
	connection.User = cfg.MysqlConfig.User
	connection.Passwd = cfg.MysqlConfig.Password
	connection.Net = "tcp"
	connection.Addr = fmt.Sprintf("%s:%d", cfg.MysqlConfig.Host, cfg.MysqlConfig.Port)
	connection.DBName = cfg.AppName
	connection.Timeout = 10 * time.Second
	connection.ReadTimeout = 30 * time.Second
	connection.WriteTimeout = 30 * time.Second
	connection.Params = map[string]string{"charset": "utf8mb4"}
	db, err := sql.Open("mysql", connection.FormatDSN())
	if err != nil {
		return fmt.Errorf("创建数据库连接失败: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	if *apply {
		return migrate(ctx, db, cfg, *backupRoot)
	}
	if *verifyOnly {
		result, err := verify(ctx, db)
		if err != nil {
			return err
		}
		return printJSON(result)
	}
	return inspect(ctx, db)
}

// querySource 允许备份查询绑定同一个一致性快照事务。
type querySource interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// readRows 将检查查询转换为可保存的结构化结果。
func readRows(ctx context.Context, db querySource, query string, args ...any) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("执行结构检查失败: %w", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		item := make(map[string]any)
		for i, name := range columns {
			if value, ok := values[i].([]byte); ok {
				item[name] = string(value)
			} else {
				item[name] = values[i]
			}
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// printJSON 输出不含凭证的迁移摘要。
func printJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// saveJSON 将检查记录写入备份目录，不向控制台输出业务内容。
func saveJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// countValue 查询单个校验计数。
func countValue(ctx context.Context, db *sql.DB, query string, args ...any) (int64, error) {
	var value int64
	err := db.QueryRowContext(ctx, query, args...).Scan(&value)
	return value, err
}

// tableExists 检查固定迁移范围内的表是否仍然存在。
func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	value, err := countValue(ctx, db, "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", table)
	return value > 0, err
}

// sqlValue 用十六进制字面量保存字符串和二进制内容，避免引号和字符集转义损失。
func sqlValue(value any) string {
	switch value := value.(type) {
	case nil:
		return "NULL"
	case []byte:
		return "X'" + hex.EncodeToString(value) + "'"
	case string:
		return "X'" + hex.EncodeToString([]byte(value)) + "'"
	default:
		return fmt.Sprint(value)
	}
}

// backupSQL 保存五张受影响表的完整结构和数据，返回每张表的精确记录数。
func backupSQL(ctx context.Context, db *sql.DB, path string) (map[string]int64, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	counts := make(map[string]int64)
	if _, err := io.WriteString(file, "-- 私聊精简迁移前的完整备份；仅在维护窗口人工恢复。\nSET NAMES utf8mb4;\nSET FOREIGN_KEY_CHECKS=0;\n"); err != nil {
		return nil, err
	}
	for _, table := range []string{"group_info", "contact_apply", "user_contact", "message", "session"} {
		exists, err := tableExistsInTx(ctx, tx, table)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		var tableName, ddl string
		if err := tx.QueryRowContext(ctx, "SHOW CREATE TABLE `"+table+"`").Scan(&tableName, &ddl); err != nil {
			return nil, err
		}
		if _, err := fmt.Fprintf(file, "\nDROP TABLE IF EXISTS `%s`;\n%s;\n", table, ddl); err != nil {
			return nil, err
		}
		rows, err := tx.QueryContext(ctx, "SELECT * FROM `"+table+"` ORDER BY id")
		if err != nil {
			return nil, err
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, err
		}
		counts[table] = 0
		for rows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				rows.Close()
				return nil, err
			}
			encoded := make([]string, len(values))
			for i, value := range values {
				encoded[i] = sqlValue(value)
			}
			if _, err := fmt.Fprintf(file, "INSERT INTO `%s` (`%s`) VALUES (%s);\n", table, strings.Join(columns, "`,`"), strings.Join(encoded, ",")); err != nil {
				rows.Close()
				return nil, err
			}
			counts[table]++
		}
		rowErr := rows.Err()
		rows.Close()
		if rowErr != nil {
			return nil, rowErr
		}
	}
	if _, err := io.WriteString(file, "\nSET FOREIGN_KEY_CHECKS=1;\n"); err != nil {
		return nil, err
	}
	if err := file.Sync(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return counts, nil
}

// tableExistsInTx 在备份快照内检查旧表，支持迁移重复执行。
func tableExistsInTx(ctx context.Context, tx *sql.Tx, table string) (bool, error) {
	var value int
	err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", table).Scan(&value)
	return value > 0, err
}

// retiredSQL 限定只删除群消息及已下线的通话消息，不包含正常私聊和 AI 对话。
const retiredSQL = "(receive_id LIKE 'G%' OR send_id LIKE 'G%' OR type=3 OR session_id IN (SELECT uuid FROM session WHERE receive_id LIKE 'G%' OR send_id LIKE 'G%'))"

// retainedRecords 记录保留消息及会话的逐行指纹，验证清理没有修改这些数据。
func retainedRecords(ctx context.Context, db *sql.DB) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)
	for table, where := range map[string]string{"message": "NOT " + retiredSQL, "session": "receive_id NOT LIKE 'G%' AND send_id NOT LIKE 'G%'"} {
		rows, err := readRows(ctx, db, "SELECT * FROM `"+table+"` WHERE "+where+" ORDER BY id")
		if err != nil {
			return nil, err
		}
		result[table] = make(map[string]string)
		for _, row := range rows {
			delete(row, "av_data")
			data, err := json.Marshal(row)
			if err != nil {
				return nil, err
			}
			result[table][fmt.Sprint(row["id"])] = fmt.Sprintf("%x", sha256.Sum256(data))
		}
	}
	return result, nil
}

// esRequest 使用配置中的连接访问聊天索引，不引入业务包初始化副作用。
func esRequest(ctx context.Context, cfg *configs.Config, method, path string, body any) (map[string]any, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	address := strings.TrimRight(strings.TrimSpace(strings.Split(cfg.ESConfig.Addresses, ",")[0]), "/")
	req, err := http.NewRequestWithContext(ctx, method, address+path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("创建搜索索引请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.ESConfig.Username != "" {
		req.SetBasicAuth(cfg.ESConfig.Username, cfg.ESConfig.Password)
	}
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("访问搜索索引失败: %w", err)
	}
	defer response.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusNotFound {
		return map[string]any{"索引不存在": true}, nil
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("搜索索引操作失败，状态码=%d", response.StatusCode)
	}
	return result, nil
}

// retiredESQuery 只匹配群聊目标或通话类型的搜索文档。
func retiredESQuery() map[string]any {
	return map[string]any{"query": map[string]any{"bool": map[string]any{"minimum_should_match": 1, "should": []any{
		map[string]any{"prefix": map[string]any{"receive_id": "G"}},
		map[string]any{"prefix": map[string]any{"send_id": "G"}},
		map[string]any{"term": map[string]any{"message_type": 3}},
	}}}}
}

// backupES 分页保存将删除的搜索文档，避免只备份第一页。
func backupES(ctx context.Context, cfg *configs.Config, path string) (int, error) {
	query := retiredESQuery()
	query["size"] = 500
	query["sort"] = []string{"_doc"}
	result, err := esRequest(ctx, cfg, http.MethodPost, "/chat_messages/_search?scroll=1m", query)
	if err != nil {
		return 0, err
	}
	var documents []any
	scrollID := ""
	defer func() {
		if scrollID != "" {
			_, _ = esRequest(ctx, cfg, http.MethodDelete, "/_search/scroll", map[string]any{"scroll_id": []string{scrollID}})
		}
	}()
	for {
		if result["索引不存在"] == true {
			break
		}
		if id, ok := result["_scroll_id"].(string); ok {
			scrollID = id
		}
		hits, ok := result["hits"].(map[string]any)
		if !ok {
			return 0, fmt.Errorf("搜索文档备份响应缺少命中信息")
		}
		batch, ok := hits["hits"].([]any)
		if !ok {
			return 0, fmt.Errorf("搜索文档备份响应格式错误")
		}
		documents = append(documents, batch...)
		if len(batch) == 0 || scrollID == "" {
			break
		}
		result, err = esRequest(ctx, cfg, http.MethodPost, "/_search/scroll", map[string]any{"scroll": "1m", "scroll_id": scrollID})
		if err != nil {
			return 0, err
		}
	}
	return len(documents), saveJSON(path, documents)
}

// clearCaches 仅清理聊天查询及旧联系人/群缓存，不触碰登录会话或其他业务数据。
func clearCaches(ctx context.Context, client *redisclient.Client) (int64, error) {
	var deleted int64
	for _, pattern := range []string{"contact:*", "chat:sessions:*", "message_list_*", "group_messagelist_*"} {
		var cursor uint64
		var keys []string
		for {
			batch, next, err := client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return deleted, err
			}
			keys = append(keys, batch...)
			cursor = next
			if cursor == 0 {
				break
			}
		}
		if len(keys) > 0 {
			count, err := client.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, err
			}
			deleted += count
		}
	}
	return deleted, nil
}

// verify 从 information_schema 和业务表检查删除结果及中文注释。
func verify(ctx context.Context, db *sql.DB) (map[string]any, error) {
	checks := map[string]string{
		"旧表剩余数":     "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('group_info','contact_apply','user_contact')",
		"通话字段剩余数":   "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='message' AND COLUMN_NAME='av_data'",
		"旧消息剩余数":    "SELECT COUNT(*) FROM message WHERE " + retiredSQL,
		"群会话剩余数":    "SELECT COUNT(*) FROM session WHERE receive_id LIKE 'G%' OR send_id LIKE 'G%'",
		"消息字段缺失注释数": "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='message' AND COLUMN_COMMENT=''",
	}
	result := make(map[string]any)
	for name, query := range checks {
		value, err := countValue(ctx, db, query)
		if err != nil {
			return nil, err
		}
		if value != 0 {
			return nil, fmt.Errorf("校验%s失败，剩余=%d", name, value)
		}
		result[name] = value
	}
	comments, err := readRows(ctx, db, "SELECT t.TABLE_COMMENT,c.COLUMN_COMMENT AS TYPE_COMMENT FROM information_schema.TABLES t JOIN information_schema.COLUMNS c ON c.TABLE_SCHEMA=t.TABLE_SCHEMA AND c.TABLE_NAME=t.TABLE_NAME AND c.COLUMN_NAME='type' WHERE t.TABLE_SCHEMA=DATABASE() AND t.TABLE_NAME='message'")
	if err != nil {
		return nil, err
	}
	if len(comments) != 1 || comments[0]["TABLE_COMMENT"] != "聊天消息表，保存用户私聊和AI对话消息" || comments[0]["TYPE_COMMENT"] != "消息类型：0文本，2文件；用户私聊和AI对话共用" {
		return nil, fmt.Errorf("消息表或类型字段的中文注释未生效")
	}
	result["实际中文注释"] = comments
	columns, err := readRows(ctx, db, "SELECT COLUMN_NAME,COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='message'")
	if err != nil {
		return nil, err
	}
	databaseFields := make(map[string]string)
	for _, column := range columns {
		databaseFields[fmt.Sprint(column["COLUMN_NAME"])] = fmt.Sprint(column["COLUMN_COMMENT"])
	}
	model := reflect.TypeOf(models.Message{})
	if len(databaseFields) != model.NumField() {
		return nil, fmt.Errorf("消息模型字段数量与数据库不一致")
	}
	for i := 0; i < model.NumField(); i++ {
		tags := make(map[string]string)
		for _, item := range strings.Split(model.Field(i).Tag.Get("gorm"), ";") {
			pair := strings.SplitN(item, ":", 2)
			if len(pair) == 2 {
				tags[pair[0]] = pair[1]
			}
		}
		if actual, ok := databaseFields[tags["column"]]; !ok || actual != tags["comment"] {
			return nil, fmt.Errorf("消息模型字段 %s 或中文注释与数据库不一致", model.Field(i).Name)
		}
	}
	result["模型字段及注释一致数"] = model.NumField()
	return result, nil
}

// migrate 在备份完成后清理旧数据与结构，最后逐行核对正常消息和会话仍然完整。
func migrate(ctx context.Context, db *sql.DB, cfg *configs.Config, backupRoot string) error {
	var locked int
	if err := db.QueryRowContext(ctx, "SELECT GET_LOCK('seekf_private_chat_cleanup',10)").Scan(&locked); err != nil || locked != 1 {
		return fmt.Errorf("无法取得私聊迁移锁")
	}
	defer db.ExecContext(context.Background(), "SELECT RELEASE_LOCK('seekf_private_chat_cleanup')")
	for _, table := range []string{"message", "session"} {
		exists, err := tableExists(ctx, db, table)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("当前数据库缺少预期表 %s，停止迁移", table)
		}
	}
	if _, err := db.ExecContext(ctx, "SET SESSION lock_wait_timeout=15"); err != nil {
		return err
	}
	foreignKeys, err := countValue(ctx, db, "SELECT COUNT(*) FROM information_schema.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA=DATABASE() AND REFERENCED_TABLE_NAME IN ('group_info','contact_apply','user_contact')")
	if err != nil {
		return err
	}
	if foreignKeys != 0 {
		return fmt.Errorf("旧表仍有外键引用，停止自动清理")
	}
	dependencies, err := countValue(ctx, db, "SELECT (SELECT COUNT(*) FROM information_schema.VIEWS WHERE TABLE_SCHEMA=DATABASE() AND VIEW_DEFINITION REGEXP 'group_info|contact_apply|user_contact|av_data') + (SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND (EVENT_OBJECT_TABLE IN ('group_info','contact_apply','user_contact') OR ACTION_STATEMENT REGEXP 'group_info|contact_apply|user_contact|av_data'))")
	if err != nil {
		return err
	}
	if dependencies > 0 {
		return fmt.Errorf("旧结构仍被视图或触发器引用，停止自动清理")
	}
	client := redisclient.NewClient(&redisclient.Options{Addr: fmt.Sprintf("%s:%d", cfg.RedisConfig.Host, cfg.RedisConfig.Port), Password: cfg.RedisConfig.Password, DB: cfg.RedisConfig.Db})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("检查聊天缓存连接失败: %w", err)
	}
	dir := filepath.Join(backupRoot, time.Now().Format("20060102-150405.000000000"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	counts, err := backupSQL(ctx, db, filepath.Join(dir, "restore.sql"))
	if err != nil {
		return fmt.Errorf("完整备份失败，未执行清理: %w", err)
	}
	retained, err := retainedRecords(ctx, db)
	if err != nil {
		return err
	}
	esCount, err := backupES(ctx, cfg, filepath.Join(dir, "retired-es-documents.json"))
	if err != nil {
		return fmt.Errorf("搜索文档备份失败，未执行清理: %w", err)
	}
	backupData, err := os.ReadFile(filepath.Join(dir, "restore.sql"))
	if err != nil {
		return err
	}
	before := map[string]any{"备份时间": time.Now().Format(time.RFC3339), "备份表记录数": counts, "保留记录指纹": retained, "待删除搜索文档数": esCount, "SQL备份SHA256": fmt.Sprintf("%x", sha256.Sum256(backupData))}
	if err := saveJSON(filepath.Join(dir, "before.json"), before); err != nil {
		return err
	}
	fmt.Println("备份已完成: " + dir)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	deletedMessages, err := tx.ExecContext(ctx, "DELETE FROM message WHERE "+retiredSQL)
	if err != nil {
		return err
	}
	deletedSessions, err := tx.ExecContext(ctx, "DELETE FROM session WHERE receive_id LIKE 'G%' OR send_id LIKE 'G%'")
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// MySQL DDL 会隐式提交，数据删除与结构变更分别执行，并依赖完整备份恢复。
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS group_info,contact_apply,user_contact"); err != nil {
		return fmt.Errorf("删除旧表失败，备份路径=%s: %w", dir, err)
	}
	columnCount, err := countValue(ctx, db, "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='message' AND COLUMN_NAME='av_data'")
	if err != nil {
		return err
	}
	alter := "ALTER TABLE message "
	if columnCount > 0 {
		alter += "DROP COLUMN av_data, "
	}
	alter += "MODIFY COLUMN type TINYINT NOT NULL COMMENT '消息类型：0文本，2文件；用户私聊和AI对话共用', COMMENT='聊天消息表，保存用户私聊和AI对话消息'"
	if _, err := db.ExecContext(ctx, alter); err != nil {
		return fmt.Errorf("清理消息字段失败，备份路径=%s: %w", dir, err)
	}
	esResult, err := esRequest(ctx, cfg, http.MethodPost, "/chat_messages/_delete_by_query?refresh=true&conflicts=abort", retiredESQuery())
	if err != nil {
		return err
	}
	if conflicts, _ := esResult["version_conflicts"].(float64); conflicts != 0 {
		return fmt.Errorf("清理搜索索引出现版本冲突")
	}
	if failures, _ := esResult["failures"].([]any); len(failures) > 0 {
		return fmt.Errorf("清理搜索索引存在失败项")
	}
	esRemaining, err := esRequest(ctx, cfg, http.MethodPost, "/chat_messages/_count", retiredESQuery())
	if err != nil {
		return err
	}
	if count, _ := esRemaining["count"].(float64); count != 0 {
		return fmt.Errorf("搜索索引仍存在旧功能文档")
	}
	cacheCount, err := clearCaches(ctx, client)
	if err != nil {
		return err
	}
	result, err := verify(ctx, db)
	if err != nil {
		return err
	}
	after, err := retainedRecords(ctx, db)
	if err != nil {
		return err
	}
	for table, rows := range retained {
		for id, hash := range rows {
			if after[table][id] != hash {
				return fmt.Errorf("保留数据校验失败，表=%s，记录=%s", table, id)
			}
		}
	}
	messageCount, err := deletedMessages.RowsAffected()
	if err != nil {
		return err
	}
	sessionCount, err := deletedSessions.RowsAffected()
	if err != nil {
		return err
	}
	result["删除群或通话消息数"] = messageCount
	result["删除群会话数"] = sessionCount
	result["清理搜索索引"] = esResult
	result["清理缓存键数"] = cacheCount
	result["保留消息校验数"] = len(retained["message"])
	result["保留会话校验数"] = len(retained["session"])
	result["备份路径"] = dir
	if err := saveJSON(filepath.Join(dir, "after.json"), result); err != nil {
		return err
	}
	return printJSON(result)
}

// inspect 只读检查废弃表、字段、历史消息和外键依赖。
func inspect(ctx context.Context, db *sql.DB) error {
	checks := map[string]string{
		"废弃表结构": "SELECT TABLE_NAME, TABLE_ROWS, ENGINE, TABLE_COMMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND (TABLE_NAME REGEXP 'group|contact' OR TABLE_NAME IN ('message','session')) ORDER BY TABLE_NAME",
		"关联字段":  "SELECT TABLE_NAME,COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,COLUMN_DEFAULT,COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND (TABLE_NAME IN ('message','session') OR COLUMN_NAME REGEXP 'group|friend|contact|av_data') ORDER BY TABLE_NAME,ORDINAL_POSITION",
		"外键依赖":  "SELECT TABLE_NAME,CONSTRAINT_NAME,COLUMN_NAME,REFERENCED_TABLE_NAME FROM information_schema.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA=DATABASE() AND (TABLE_NAME IN ('group_info','contact_apply','user_contact') OR REFERENCED_TABLE_NAME IN ('group_info','contact_apply','user_contact')) AND REFERENCED_TABLE_NAME IS NOT NULL",
		"消息分布":  "SELECT LEFT(send_id,1) AS send_kind,LEFT(receive_id,1) AS receive_kind,type,COUNT(*) AS total FROM message GROUP BY LEFT(send_id,1),LEFT(receive_id,1),type",
		"会话分布":  "SELECT LEFT(send_id,1) AS send_kind,LEFT(receive_id,1) AS receive_kind,COUNT(*) AS total FROM session GROUP BY LEFT(send_id,1),LEFT(receive_id,1)",
	}
	result := make(map[string]any)
	for name, query := range checks {
		rows, err := readRows(ctx, db, query)
		if err != nil {
			return fmt.Errorf("检查%s失败: %w", name, err)
		}
		result[name] = rows
	}
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
