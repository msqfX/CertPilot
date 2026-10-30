package database

import (
	"github.com/msqfX/CertPilot/backend/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
)

func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Domain{},
		&model.Certificate{},
		&model.CertificateDomain{},
		&model.NotificationChannel{},
		&model.NotificationLog{},
		&model.OperationLog{},
	); err != nil {
		return nil, err
	}
	log.Println("[db] migrated")
	return db, nil
}
