package testutil

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/MrNemo64/coc-tracker/db"
	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDatabase struct {
	container *postgres.PostgresContainer
	DB        *sqlx.DB
}

func (tb *TestDatabase) Shutdown() error {
	dbErr := tb.DB.Close()
	conErr := tb.container.Terminate(context.Background())
	if dbErr != nil || conErr != nil {
		return fmt.Errorf("error shuting down test database, database: %v, container: %v", dbErr, conErr)
	}
	return nil
}

func CreatePostgresContainer() (*TestDatabase, error) {
	ctx := context.Background()

	dbName := "test_container_database"
	dbUser := "test_container_user"
	dbPassword := "test_container_password"

	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("docker.io/postgres:16-alpine"),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to startup test database: %v", err)
	}

	netPort, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get test database port: %v", err)
	}

	port, err := strconv.Atoi(netPort.Port())
	if err != nil {
		return nil, fmt.Errorf("failed parse the test database post: %v", err)
	}

	dbCon, err := db.ConnectToDatabase(db.DatabaseConfiguration{
		Host:     "localhost",
		Port:     port,
		Database: dbName,
		User:     dbUser,
		Password: dbPassword,
		SSL:      "disable",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %v", err)
	}

	testContainer := &TestDatabase{
		container: postgresContainer,
		DB:        dbCon,
	}

	err = db.Migrate(dbCon)
	if err != nil {
		testContainer.Shutdown()
		return nil, fmt.Errorf("failed to migrate test database: %v", err)
	}

	return testContainer, nil
}
