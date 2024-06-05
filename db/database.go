package db

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
)

func ConnectToDatabase() (*sqlx.DB, error) {
	host := os.Getenv("DB_HOST")
	password := os.Getenv("DB_PASSWORD")
	user := os.Getenv("DB_USER")
	database := os.Getenv("DB_DATABASE")
	ssl := os.Getenv("DB_SSL")
	return sqlx.Connect("postgres", fmt.Sprintf("host=%s password=%s user=%s database=%s sslmode=%s", host, password, user, database, ssl))
}
