package track

import (
	"bufio"
	"os"
	"time"

	"golang.org/x/time/rate"
)

type apiKey struct {
	key       string
	limiter   *rate.Limiter
	timesUsed int
}

type KeyList struct {
	keys []apiKey
}

func LoadKeysFromFile(path string) (*KeyList, error) {
	readFile, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer readFile.Close()

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	keyList := &KeyList{
		keys: make([]apiKey, 0),
	}

	for fileScanner.Scan() {
		keyList.keys = append(keyList.keys, apiKey{
			key:       fileScanner.Text(),
			limiter:   rate.NewLimiter(rate.Every(1*time.Second), 35),
			timesUsed: 0,
		})
	}

	return keyList, nil
}
