package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/SeakMengs/AutoCert/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectReturnGormDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v TimeZone=Asia/Phnom_Penh", cfg.HOST, cfg.USERNAME, cfg.PASSWORD, cfg.DATABASE, cfg.PORT)

	return gorm.Open(postgres.New(postgres.Config{
		DSN: dsn,
		// PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		TranslateError: true,
	})
}

func Connect(cfg config.DatabaseConfig) (*sql.DB, error) {
	_db, err := ConnectReturnGormDB(cfg)
	if err != nil {
		log.Panic("Failed to connect database")
	}

	db, err := _db.DB()
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	duration, err := time.ParseDuration(cfg.MaxIdleTime)
	if err != nil {
		fmt.Print("Failed to parse database max idle time")
		log.Panic(err)
	} else {
		db.SetConnMaxIdleTime(duration)
	}

	return db, nil
}
