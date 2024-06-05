package main

import (
	"github.com/MrNemo64/coc-tracker/track"
	"github.com/MrNemo64/coc-tracker/util"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
	util.SetupLog()
	track.Run()
}
