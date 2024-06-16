package db

import (
	"fmt"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DatabaseConfiguration struct {
	Host     string
	Port     int
	Database string
	Password string
	User     string
	SSL      string
}

func DatabaseConfigurationFromEnv() DatabaseConfiguration {
	port, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		panic(err)
	}
	return DatabaseConfiguration{
		Host:     os.Getenv("DB_HOST"),
		Port:     port,
		Database: os.Getenv("DB_DATABASE"),
		Password: os.Getenv("DB_PASSWORD"),
		User:     os.Getenv("DB_USER"),
		SSL:      os.Getenv("DB_SSL"),
	}
}

func ConnectToDatabase(conf DatabaseConfiguration) (*sqlx.DB, error) {
	return sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%d database=%s user=%s password=%s sslmode=%s", conf.Host, conf.Port, conf.Database, conf.User, conf.Password, conf.SSL))
}

func Migrate(db *sqlx.DB) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return err
	}
	migration, err := migrate.NewWithDatabaseInstance("file://"+os.Getenv("MIGRATIONS_DIR"), "postgres", driver)
	if err != nil {
		return err
	}
	return migration.Up()
}
