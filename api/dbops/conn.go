package dbops

import (
	"fmt"
	"time"
	"video-server/api/cache"
	"video-server/api/utils"
	"video-server/internal/config"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Db *gorm.DB

// Init 初始化数据库连接和Redis缓存
// 为什么不用 init()？
// 1. init() 在 import 时自动执行，但此时 config 和 logger 可能还没初始化
// 2. 手动初始化可以更好地控制执行顺序和错误处理
// 3. 可以在 main 函数中按需初始化
func Init() error {
	// 1. 初始化数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.DbUser,
		config.AppConfig.DbPwd,
		config.AppConfig.DbAddr,
		config.AppConfig.DbName,
	)

	utils.Logger.Info("正在连接数据库",
		zap.String("user", config.AppConfig.DbUser),
		zap.String("addr", config.AppConfig.DbAddr),
		zap.String("database", config.AppConfig.DbName))

	var err error
	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		utils.Logger.Error("数据库连接失败",
			zap.String("addr", config.AppConfig.DbAddr),
			zap.Error(err))
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := Db.DB()
	if err != nil {
		utils.Logger.Error("获取数据库实例失败", zap.Error(err))
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	utils.Logger.Info("数据库连接成功",
		zap.String("addr", config.AppConfig.DbAddr),
		zap.Int("max_open_conns", 100),
		zap.Int("max_idle_conns", 10))

	// 2. 初始化Redis缓存
	utils.Logger.Info("正在连接Redis",
		zap.String("addr", config.AppConfig.RedisAddr))

	if err := cache.InitRedis(
		config.AppConfig.RedisAddr,
		config.AppConfig.RedisPwd,
		config.AppConfig.RedisDB,
	); err != nil {
		utils.Logger.Warn("Redis连接失败（将继续运行，但无缓存）",
			zap.String("addr", config.AppConfig.RedisAddr),
			zap.Error(err))
		// 不返回错误，允许服务在没有缓存的情况下运行
	} else {
		utils.Logger.Info("Redis连接成功",
			zap.String("addr", config.AppConfig.RedisAddr))
	}

	return nil
}
