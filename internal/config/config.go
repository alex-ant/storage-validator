package config

import (
	"flag"
	"log"

	"github.com/alex-ant/envs"
)

const (
	batchSize200Mb = 200 * 1024 * 1024
)

var (
	Mode      = flag.String("m", "", "operation mode (init/validate/reset)")
	Directory = flag.String("d", "", "Working directory path")
)

func init() {
	// Parse flags if not parsed already.
	if !flag.Parsed() {
		flag.Parse()
	}

	// Determine and read environment variables.
	flagsErr := envs.GetAllFlags()
	if flagsErr != nil {
		log.Fatal(flagsErr)
	}
}
