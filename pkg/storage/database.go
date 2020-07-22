/*
Package storage makes the database connection
*/
package storage

import (
	"fmt"

	"github.com/GGP1/palo/internal/cfg"
	"github.com/GGP1/palo/pkg/auth/email"
	"github.com/GGP1/palo/pkg/model"

	"github.com/jinzhu/gorm"
)

// NewDatabase creates a database and the tables required by the api
// It returns a pointer to the gorm.DB struct, the close function and an error
func NewDatabase() (*gorm.DB, func() error, error) {
	db, err := gorm.Open("postgres", cfg.DBURL)
	if err != nil {
		return nil, nil, fmt.Errorf("could not open the database: %w", err)
	}

	err = db.DB().Ping()
	if err != nil {
		return nil, nil, fmt.Errorf("connection to the database died: %w", err)
	}

	err = tableExists(db, &model.Product{}, &model.User{}, &model.Review{}, &model.Shop{}, &model.Location{})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create the table: %w", err)
	}

	if db.Table("pending_list").HasTable(&email.List{}) != true {
		db.Table("pending_list").CreateTable(&email.List{}).AutoMigrate(&email.List{})
	}
	if db.Table("validated_list").HasTable(&email.List{}) != true {
		db.Table("validated_list").CreateTable(&email.List{}).AutoMigrate(&email.List{})
	}

	return db, db.Close, nil
}

// Check if a table is already created, if not, create it
// Plus model automigration
func tableExists(db *gorm.DB, models ...interface{}) error {
	for _, model := range models {
		db.AutoMigrate(model)
		if db.HasTable(model) != true {
			err := db.CreateTable(model).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}
