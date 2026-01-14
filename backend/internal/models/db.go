package models

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&Market{},
		&Order{},
		&Trade{},
		&UserBalance{},
		&BalanceLog{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
