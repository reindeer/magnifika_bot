package main

import (
	"fmt"
	"os"

	"github.com/reindeer/magnifika_bot/config"

	"gitlab.com/gorib/waffle/app"
)

func main() {
	config.Init()
	application, err := app.New()
	if err != nil {
		fmt.Printf("Failed to create an application: %v\n", err)
		os.Exit(1)
	}
	application.RunUntilStop()
}
