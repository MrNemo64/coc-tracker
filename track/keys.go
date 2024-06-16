package track

import (
	"bufio"
	"os"
	"sync"
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
	mu   sync.Mutex
}

func (kl *KeyList) GetKey() string {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	var bestKey *apiKey

	// Find the key with the most available tokens
	for i := range kl.keys {
		if bestKey == nil || (kl.keys[i].limiter.Allow() && kl.keys[i].limiter.Burst() > bestKey.limiter.Burst()) {
			bestKey = &kl.keys[i]
		}
	}

	// If a key was found with available tokens, consume one and return the key
	if bestKey != nil {
		bestKey.timesUsed++
		return bestKey.key
	}

	// If no key has available tokens, block until one becomes available
	for {
		for i := range kl.keys {
			reserve := kl.keys[i].limiter.Reserve()
			if reserve.OK() {
				time.Sleep(reserve.Delay())
				kl.keys[i].timesUsed++
				return kl.keys[i].key
			}
		}
		time.Sleep(10 * time.Millisecond) // Add a small sleep to prevent tight loop
	}
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
