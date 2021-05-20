package trace

import (
	"github.com/jinzhu/gorm"
)

func Open(source, name string) (*gorm.DB, error) {
	sqldb := OpenDB(source, name)
	db, err := gorm.Open("mysql", sqldb)
	if err != nil {
		return db, err
	}
	return db, err
}
