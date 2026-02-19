package dbops

import (
	"fmt"
	"video-server/api/utils"
	"video-server/config"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Db *gorm.DB

func init() {

	// 构建 DSN: user:password@tcp(addr)/dbname...
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.DbUser,
		config.AppConfig.DbPwd,
		config.AppConfig.DbAddr,
		config.AppConfig.DbName,
	)

	var err error
	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		utils.Logger.Error("open mysql failed,", zap.Error(err))
		panic(err)
	}
}
