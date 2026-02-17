package config

import (
	"log"

	"github.com/spf13/viper"
)

// Configuration 映射配置文件内容
type Configuration struct {
	LBAddr               string `mapstructure:"lb_addr"`
	OssAddr              string `mapstructure:"oss_addr"`
	OssKey               string `mapstructure:"oss_key"`
	OssSecret            string `mapstructure:"oss_secret"`
	OssBucket            string `mapstructure:"oss_bucket"`
	OssRegion            string `mapstructure:"oss_region"`
	DbAddr               string `mapstructure:"db_addr"`
	DbUser               string `mapstructure:"db_user"`
	DbPwd                string `mapstructure:"db_pwd"`
	DbName               string `mapstructure:"db_name"`
	VideoDeleteDelayTime int    `mapstructure:"video_delete_delay_time"`
	RedisAddr            string `mapstructure:"redis_addr"`
	RedisPwd             string `mapstructure:"redis_pwd"`
	RedisDB              int    `mapstructure:"redis_db"`
}

var AppConfig Configuration

func init() {
	// 设置文件名（不带扩展名）
	viper.SetConfigName("config")
	// 设置搜索路径
	viper.AddConfigPath("./config")
	// 设置文件类型
	viper.SetConfigType("json")

	// 允许环境变量覆盖 (类似 Spring Boot)
	// 例如 export OSS_KEY=xyz 会覆盖文件中的 oss_key
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}
}
