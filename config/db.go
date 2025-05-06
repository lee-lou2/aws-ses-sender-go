package config

import (
	"fmt" // DSN 문자열 포맷팅을 위해 추가
	"log"
	"sync"
	"time"

	"gorm.io/driver/postgres" // SQLite 드라이버 대신 PostgreSQL 드라이버를 임포트합니다.
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbInstance *gorm.DB
	dbOnce     sync.Once
)

// GetDB Database instance
func GetDB() *gorm.DB {
	dbOnce.Do(func() {
		host := GetEnv("DB_HOST", "localhost")
		port := GetEnv("DB_PORT", "5432")
		user := GetEnv("DB_USER", "postgres")
		password := GetEnv("DB_PASSWORD", "postgres")
		dbname := GetEnv("DB_NAME", "postgres")
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Seoul",
			host, user, password, dbname, port)
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Error),
		})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("Failed to get database instance: %v", err)
		}
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(time.Hour)
		dbInstance = db
	})
	return dbInstance
}
