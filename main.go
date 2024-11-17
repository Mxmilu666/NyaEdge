package main

import (
	"context"
	"fmt"
	"nyaedge-center/source"
	"nyaedge-center/source/helper"
	"nyaedge-center/source/server"
	"nyaedge-center/source/zaplogger"
	"os"

	"go.uber.org/zap"
)

var Config *source.Config

func main() {
	fmt.Println("NyaEdge-Center v0.0.1")
	logger, err := zaplogger.Setup()
	if err != nil {
		return
	}

	configFile := "config.yml"

	// 检查配置文件是否存在，如果不存在则创建默认配置文件
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		err := source.CreateDefaultConfig(configFile)
		if err != nil {
			logger.Error("Error creating default config file: %v", zap.Error(err))
			return
		}
		logger.Info("Created default config file. Please edit it with your configuration.")
		return
	}

	// 读取配置文件
	Config, err := source.ReadConfig(configFile)
	if err != nil {
		logger.Error("Error reading config file: %v", zap.Error(err))
		return
	}

	// 初始化数据库
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d",
		Config.Database.Username,
		Config.Database.Password,
		Config.Database.Address,
		Config.Database.Port,
	)
	database, err := source.SetupDatabase(uri)
	if err != nil {
		logger.Error("Error setting up database: %v", zap.Error(err))
		return
	}

	defer func() {
		if err = database.Disconnect(context.TODO()); err != nil {
			logger.Error("Error disconnecting from database: %v", zap.Error(err))
		}
	}()

	// 确保需要的集合存在
	err = source.EnsureCollection(database, source.DatabaseName, source.NodeCollection)
	if err != nil {
		logger.Error("Error ensuring nodes collection: %v", zap.Error(err))
		return
	}

	// 初始化 JWT
	helper.GetInstance()

	if err := server.StartServer(Config.Server.Address, Config.Server.Port, logger, database); err != nil {
		logger.Error("Server startup failed: %v", zap.Error(err))
	}

}
