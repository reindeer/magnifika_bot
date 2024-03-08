package main

import (
	"github.com/reindeer/magnifika_bot/config"

	"gitlab.com/gorib/waffle/app"
)

func main() {
	config.Init()
	app.New().RunUntilStop()
}
