package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/philippgille/chromem-go"
)

type config struct {
	inputDB  string
	outputDB string
	compress bool
}

func main() {
	cfg := parseFlags()

	if err := run(cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() config {
	cfg := config{}

	flag.StringVar(&cfg.inputDB, "input", "advisory-ollama.db", "Input database path (directory)")
	flag.StringVar(&cfg.outputDB, "output", "advisory.gob", "Output database file")
	flag.BoolVar(&cfg.compress, "compress", false, "Compress output with gzip")

	flag.Parse()

	return cfg
}

func run(cfg config) error {
	// Load existing persistent database
	log.Printf("Loading database from: %s", cfg.inputDB)
	db, err := chromem.NewPersistentDB(cfg.inputDB, false)
	if err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Export to single file
	log.Printf("Exporting to: %s (compress=%v)", cfg.outputDB, cfg.compress)
	file, err := os.Create(cfg.outputDB)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	if err := db.ExportToWriter(file, cfg.compress, ""); err != nil {
		return fmt.Errorf("failed to export: %w", err)
	}

	// Get file size
	info, err := file.Stat()
	if err == nil {
		log.Printf("Successfully exported database: %d bytes", info.Size())
	}

	return nil
}
