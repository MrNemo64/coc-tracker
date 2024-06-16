package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	env, err := FindFileInRoot(".env")
	if err != nil {
		panic(fmt.Errorf("error finding .env file: %w", err))
	}

	err = godotenv.Load(env)
	if err != nil {
		panic(fmt.Errorf("error loading .env file: %w", err))
	}
}

// https://github.com/joho/godotenv/issues/43
func FindFileInRoot(name string) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			break
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			return "", fmt.Errorf("go.mod not found")
		}
		currentDir = parent
	}

	return filepath.Join(currentDir, name), nil
}
