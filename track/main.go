package track

import "os"

func Run() {
	keysFile := os.Getenv("KEYS_FILE")
	if keysFile == "" {
		panic("Keys file not specified")
	}

	keys, err := LoadKeysFromFile(keysFile)
	if err != nil {
		panic(err)
	}

	client := CreateCocClient(keys)
	client.Run()
}
