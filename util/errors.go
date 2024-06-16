package util

import "errors"

var ErrCancelled = errors.New("cancelled operation")

func IsCancelledError(err error) bool {
	return err == ErrCancelled
}
