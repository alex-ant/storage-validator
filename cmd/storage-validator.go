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

	// Run selected hasher mode.
	switch *config.Mode {
	case "init":
		initErr := hasher.Init()
		if initErr != nil {
			log.Fatalf("failed to init directory: %v", initErr)
		}
	case "validate":
		validateErr := hasher.Validate()
		if validateErr != nil {
			log.Fatalf("failed to validate directory: %v", validateErr)
		}
	case "reset":
		hasher.Reset()
	default:
		log.Fatal("invalid mode")
	}
}
