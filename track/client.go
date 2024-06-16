package track

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/MrNemo64/coc-tracker/db"
	"github.com/MrNemo64/coc-tracker/track/jobs"
	"github.com/MrNemo64/coc-tracker/track/jobs/update"
	"github.com/MrNemo64/coc-tracker/util"
	"github.com/jmoiron/sqlx"
)

type CocClient struct {
	keys      *KeyList
	jobs      *jobs.RegisteredJobs
	ctx       context.Context
	cancelCtx context.CancelFunc
	logger    *slog.Logger
	db        *sqlx.DB
	client    *http.Client
}

func (c *CocClient) Get(ctx context.Context, url string) (response *http.Response, cacheHit bool, err error) {
	cacheHit = false

	response, err = c.client.Get(util.BaseUrl + url)

	return
}

func (c *CocClient) GetDB() *sqlx.DB {
	return c.db
}

func CreateCocClient() *CocClient {
	logger := util.GetLogger("client")
	keysFile := os.Getenv("KEYS_FILE")
	if keysFile == "" {
		panic("No keys file")
	}

	keys, err := LoadKeysFromFile(keysFile)
	if err != nil {
		panic(err)
	}

	if len(keys.keys) == 0 {
		panic("No keys loaded")
	}

	logger.Info(fmt.Sprintf("Loaded %d keys", len(keys.keys)))

	jobQueue := jobs.NewJobQueue()
	addAllJobKinds(jobQueue)

	db, err := db.ConnectToDatabase(db.DatabaseConfigurationFromEnv())
	if err != nil {
		panic(err)
	}

	logger.Info("Conected to database")

	logger.Info("Checking job status")

	ctx, cancel := context.WithCancel(context.Background())

	return &CocClient{
		jobs:      jobQueue,
		keys:      keys,
		ctx:       ctx,
		cancelCtx: cancel,
		logger:    logger,
		db:        db,
		client:    &http.Client{},
	}
}

func (client *CocClient) Run() {
	sigChan := make(chan os.Signal, 1)

	client.logger.Info("Migrating database")
	if err := db.Migrate(client.db); err != nil {
		client.logger.Error("Error migrating database", "err", err)
		panic(err)
	}

	client.logger.Info("Checking jobs")
	if err := client.jobs.CheckJobs(client.db); err != nil {
		client.logger.Error("Error checking jobs", "err", err)
		panic(err)
	}

	client.logger.Info("Started tracker")
	client.logger.Info("Starting tracker")
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go client.jobs.RunJobLoop(client, client.logger.With("name", "job loop"), client.ctx)
	<-sigChan

	client.logger.Info("Stopping tracker")
	client.cancelCtx()
	if err := client.db.Close(); err != nil {
		client.logger.Error("Error closing database connection", "err", err)
	}
	client.logger.Info("Stopped tracker")
}

func addAllJobKinds(queue *jobs.RegisteredJobs) {
	queue.RegisterJobKind(update.NewFetchCapitalLeaguesProvider())
}
