package dao

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Db *gorm.DB

func init() {
	dsn := "root:@tcp(127.0.0.1:3306)/tiktok?charset=utf8mb4&parseTime=True"
	var err error
	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Panicln("failed to connect target database from MySQL, detail:", err)
	}
}
