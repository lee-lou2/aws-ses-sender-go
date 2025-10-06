package config

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbInstance *gorm.DB
	dbOnce     sync.Once
)

// GetDB 데이터베이스 인스턴스 반환 (싱글톤)
func GetDB() *gorm.DB {
	dbOnce.Do(func() {
		dbPath := GetEnv("DB_PATH", "./data/app.db")

		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger:      logger.Default.LogMode(logger.Error),
			PrepareStmt: false,
			NowFunc: func() time.Time {
				return time.Now().UTC()
			},
		})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("Failed to get database instance: %v", err)
		}

		maxOpenConns := getEnvAsInt("DB_MAX_OPEN_CONNS", 1)
		maxIdleConns := getEnvAsInt("DB_MAX_IDLE_CONNS", 1)
		connMaxLifetime := getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
		connMaxIdleTime := getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute)

		sqlDB.SetMaxOpenConns(maxOpenConns)
		sqlDB.SetMaxIdleConns(maxIdleConns)
		sqlDB.SetConnMaxLifetime(connMaxLifetime)
		sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

		// SQLite3 성능 최적화
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA synchronous=NORMAL")
		db.Exec("PRAGMA cache_size=-64000")
		db.Exec("PRAGMA busy_timeout=5000")
		db.Exec("PRAGMA foreign_keys=ON")

		if err := sqlDB.Ping(); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}

		log.Printf("Database connected successfully (path=%s, max_open=%d)", dbPath, maxOpenConns)
		dbInstance = db
	})
	return dbInstance
}

// getEnvAsInt 환경 변수를 정수로 변환
func getEnvAsInt(key string, defaultVal int) int {
	val := GetEnv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("Invalid integer value for %s: %v, using default: %d", key, err, defaultVal)
		return defaultVal
	}
	return intVal
}

// getEnvAsDuration 환경 변수를 Duration으로 변환
func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	val := GetEnv(key)
	if val == "" {
		return defaultVal
	}
	duration, err := time.ParseDuration(val)
	if err != nil {
		log.Printf("Invalid duration value for %s: %v, using default: %v", key, err, defaultVal)
		return defaultVal
	}
	return duration
}

// CloseDB 데이터베이스 연결 종료
func CloseDB() error {
	if dbInstance == nil {
		return nil
	}
	sqlDB, err := dbInstance.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	log.Println("Database connection closed successfully")
	return nil
}

// GetEnvAsInt 환경 변수를 정수로 변환 (외부 노출용)
func GetEnvAsInt(key string, defaultVal int) int {
	return getEnvAsInt(key, defaultVal)
}
