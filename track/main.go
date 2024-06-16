package track

import (
	"github.com/MrNemo64/coc-tracker/util"
)

func init() {
	util.LoadEnv()
}

func Run() {
	client := CreateCocClient()
	client.Run()
}
