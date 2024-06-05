package main

import (
	"github.com/MrNemo64/coc-tracker/track"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	track.Run()
}
