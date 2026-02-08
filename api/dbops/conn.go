package dbops

import (
	"fmt"
	"video-server/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var Db *sqlx.DB

func init() {

	// 构建 DSN: user:password@tcp(addr)/dbname...
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.DbUser,
		config.AppConfig.DbPwd,
		config.AppConfig.DbAddr,
		config.AppConfig.DbName,
	)

	database, err := sqlx.Open("mysql", dsn)
	if err != nil {
		fmt.Println("open mysql failed,", err)
		panic(err)
	}
	Db = database
}
