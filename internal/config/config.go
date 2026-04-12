package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Configuration 映射配置文件内容
type Configuration struct {
	// 服务配置
	ServiceName string `mapstructure:"service_name"` // 服务名称（用于日志和追踪）
	APIAddr     string `mapstructure:"api_addr"`     // API服务地址
	WebAddr     string `mapstructure:"web_addr"`     // Web服务地址
	StreamAddr  string `mapstructure:"stream_addr"`  // Stream服务地址

	// OSS配置
	OssAddr   string `mapstructure:"oss_addr"`
	OssKey    string `mapstructure:"oss_key"`
	OssSecret string `mapstructure:"oss_secret"`
	OssBucket string `mapstructure:"oss_bucket"`
	OssRegion string `mapstructure:"oss_region"`

	// 数据库配置
	DbAddr string `mapstructure:"db_addr"`
	DbUser string `mapstructure:"db_user"`
	DbPwd  string `mapstructure:"db_pwd"`
	DbName string `mapstructure:"db_name"`

	// Redis配置
	RedisAddr string `mapstructure:"redis_addr"`
	RedisPwd  string `mapstructure:"redis_pwd"`
	RedisDB   int    `mapstructure:"redis_db"`

	// 定时任务配置
	VideoDeleteDelayTime int `mapstructure:"video_delete_delay_time"` // 视频删除延迟时间（秒）
	VideoDeleteBatchSize int `mapstructure:"video_delete_batch_size"` // 每次处理的视频数量
}

var AppConfig Configuration

// Load 加载配置
// 为什么要这么做？
// 1. 支持从环境变量覆盖配置（容器化部署必需）
// 2. 支持不同环境的配置文件（dev/test/prod）
// 3. 配置加载失败应该让服务启动失败（快速失败原则）
func Load(configPath string) error {
	// 优先从环境变量读取配置文件路径
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}
	if configPath == "" {
		configPath = "config.json" // 默认配置文件路径
	}

	// 设置配置文件路径
	viper.SetConfigFile(configPath)
	viper.SetConfigType("json")

	// 允许环境变量覆盖
	// 例如：export DB_ADDR=localhost:3306 会覆盖配置文件中的 db_addr
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析到结构体
	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 此处不使用logger，因为logger在config加载后才初始化
	// 在main.go中使用zap记录配置加载成功
	return nil
}

// MustLoad 加载配置，失败则panic
// 为什么要有这个方法？
// 配置加载失败时，服务不应该继续运行（快速失败 > 默默运行错误配置）
func MustLoad(configPath string) {
	if err := Load(configPath); err != nil {
		// 配置加载失败时，logger还未初始化，使用panic
		panic(fmt.Sprintf("[Config] 配置加载失败: %v", err))
	}
}
