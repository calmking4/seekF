package db

import (
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
)

// ESClient Elasticsearch客户端单例
var ESClient *elasticsearch.Client

func init() {
	cfg := configs.GetConfig()

	addresses := strings.Split(cfg.ESConfig.Addresses, ",")
	esCfg := elasticsearch.Config{
		Addresses: addresses,
	}
	if cfg.ESConfig.Username != "" {
		esCfg.Username = cfg.ESConfig.Username
		esCfg.Password = cfg.ESConfig.Password
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		zlog.Error("创建ES客户端失败: " + err.Error())
		return
	}

	// 测试连接
	res, err := client.Info()
	if err != nil {
		zlog.Error("连接ES失败: " + err.Error())
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		zlog.Error("ES连接返回错误: " + res.String())
		return
	}

	ESClient = client
	zlog.Info(fmt.Sprintf("已连接到Elasticsearch: %s", cfg.ESConfig.Addresses))

	// 自动创建索引
	if err := EnsureESIndices(); err != nil {
		zlog.Error("创建ES索引失败: " + err.Error())
	}
}
