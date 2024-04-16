package config

import (
	"os"

	"github.com/pborman/getopt/v2"
)

func Init() {
	var (
		db string
	)
	getopt.FlagLong(&db, "db", 'd', "Specify path to registry database")
	_ = getopt.Getopt(nil)
	if db != "" {
		_ = os.Setenv("DB_PATH", db)
	}
	os.Args = append([]string{getopt.CommandLine.Program()}, getopt.Args()...)

	InitDi()
}
