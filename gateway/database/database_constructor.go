package database

import (
	"app/dto"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	db *gorm.DB
}

func New(config dto.Config) (*Database, error) {
	connectString := ""
	var db *gorm.DB
	var err error

	dbConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	if config.DB.Debug {
		newLogger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold: time.Second, // Slow SQL threshold
				LogLevel:      logger.Info, // Log level
				Colorful:      true,        // Disable color
			},
		)
		dbConfig.Logger = newLogger
	}

	connectString = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
		config.DB.Host,
		config.DB.Port,
		config.DB.User,
		config.DB.Password,
		config.DB.Name,
	)
	if config.DB.SSLMode != "" {
		connectString += " sslmode=" + config.DB.SSLMode
	}
	db, err = gorm.Open(postgres.Open(connectString), dbConfig)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(150)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Автомиграция базы, тут нужно указать все модели
	err = db.AutoMigrate(
		&User{},
	)
	if err != nil {
		return nil, err
	}

	return &Database{
		db: db,
	}, nil
}

func (d *Database) DB() *gorm.DB {
	return d.db
}
