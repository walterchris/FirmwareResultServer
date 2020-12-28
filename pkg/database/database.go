package database

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Init Database
func Init() (*sql.DB, error) {
	db, err := sql.Open("mysql", "poster:poster@tcp(127.0.0.1:6603)/testing?parseTime=true")
	if err != nil {
		return nil, err
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	return db, nil
}

// Reconnect to Database
func Reconnect(c *sql.DB) error {
	con, err := Init()
	if err != nil {
		return err
	}
	c = con
	return nil
}
