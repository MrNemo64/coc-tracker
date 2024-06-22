package testutil

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/MrNemo64/coc-tracker/db"
	"github.com/MrNemo64/coc-tracker/track/jobs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

func (tdb *TestDatabase) AssertJobsTableEquals(t testing.TB, expected []jobs.DBJob) {
	t.Helper()
	rows, err := tdb.DB.Queryx("SELECT * FROM jobs ORDER BY id ASC")
	if err != nil {
		t.Fatalf("Failed to query table jobs: %v", err)
	}
	t.Cleanup(func() { rows.Close() })

	var results []jobs.DBJob
	for rows.Next() {
		var row jobs.DBJob
		if err := rows.StructScan(&row); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		results = append(results, row)
	}

	if diff := cmp.Diff(expected, results, cmpopts.EquateApproxTime(time.Millisecond*500)); diff != "" {
		t.Errorf("Table jobs does not match expected rows:\n%s", diff)
	}
}
