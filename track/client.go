package track

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/MrNemo64/coc-tracker/db"
	"github.com/MrNemo64/coc-tracker/util"
	"github.com/jmoiron/sqlx"
)

type CocClient struct {
	keys      *KeyList
	ctx       context.Context
	cancelCtx context.CancelFunc
	logger    *slog.Logger
	db        *sqlx.DB
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

	db, err := db.ConnectToDatabase()
	if err != nil {
		panic(err)
	}

	logger.Info("Conected to database")

	ctx, cancel := context.WithCancel(context.Background())
	return &CocClient{
		keys:      keys,
		ctx:       ctx,
		cancelCtx: cancel,
		logger:    logger,
		db:        db,
	}
}

func (client *CocClient) Run() {
	sigChan := make(chan os.Signal, 1)

	client.logger.Info("Starting tracker")
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	client.logger.Info("Started tracker")
	<-sigChan

	client.logger.Info("Stopping tracker")
	client.cancelCtx()
	client.logger.Info("Stopped tracker")
}
