package main

import (
	"log"

	"github.com/alex-ant/storage-validator/internal/config"
	"github.com/alex-ant/storage-validator/internal/hasher"
)

func main() {
	// Read config.
	if *config.Directory == "" {
		log.Fatal("working directory is not specified")
	}

	if *config.Mode == "" {
		log.Fatal("mode is not specified")
	}

	// Init hasher.
	hasher, hasherErr := hasher.New(*config.Directory)
	if hasherErr != nil {
		log.Fatalf("failed to init hasher client: %v", hasherErr)
	}

	switch *config.Mode {
	case "init":
		initErr := hasher.Init()
		if initErr != nil {
			log.Fatalf("failed to init directory: %v", initErr)
		}
	case "validate":

	case "reset":
		hasher.Reset()
	}
}
