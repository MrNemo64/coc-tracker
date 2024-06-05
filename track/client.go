package track

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type CocClient struct {
	keys      *KeyList
	ctx       context.Context
	cancelCtx context.CancelFunc
}

func CreateCocClient(keys *KeyList) *CocClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &CocClient{
		keys:      keys,
		ctx:       ctx,
		cancelCtx: cancel,
	}
}

func (client *CocClient) Run() {
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	client.cancelCtx()
}
