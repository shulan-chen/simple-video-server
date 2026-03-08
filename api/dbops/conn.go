package dbops

import (
	"fmt"
	"log"
	"time"
	"video-server/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Db *gorm.DB

// Init 初始化数据库连接
// 为什么不用 init()？
// 1. init() 在 import 时自动执行，但此时 config 和 logger 可能还没初始化
// 2. 手动初始化可以更好地控制执行顺序和错误处理
// 3. 可以在 main 函数中按需初始化
func Init() error {
	// 构建 DSN: user:password@tcp(addr)/dbname...
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.DbUser,
		config.AppConfig.DbPwd,
		config.AppConfig.DbAddr,
		config.AppConfig.DbName,
	)

	log.Printf("[Database] 正在连接数据库: %s@%s/%s",
		config.AppConfig.DbUser,
		config.AppConfig.DbAddr,
		config.AppConfig.DbName)

	var err error
	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := Db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期

	log.Printf("[Database] 数据库连接成功")
	return nil
}
