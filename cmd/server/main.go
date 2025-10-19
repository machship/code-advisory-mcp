package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github/machship/code-advisory-mcp/internal/embedder"
	"github/machship/code-advisory-mcp/internal/mcp"

	mcpServer "github.com/mark3labs/mcp-go/server"
	"github.com/philippgille/chromem-go"
)

//go:embed embedded/advisory.gob
var embeddedDBBytes []byte

//go:embed embedded/INSTRUCTIONS.md
var embeddedInstructions string

type config struct {
	dbPath             string
	backend            string
	azureEndpoint      string
	azureKey           string
	ollamaBaseURL      string
	model              string
	useEmbedded        bool
	printInstructions  bool
	queryLogDir        string
}

func main() {
	cfg := parseFlags()
	
	if cfg.printInstructions {
		printPromptInstructions()
		return
	}

	if err := run(cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() config {
	cfg := config{}

	flag.StringVar(&cfg.dbPath, "db", getEnv("DB_PATH", ""), "Path to advisory database (empty = use embedded)")
	flag.StringVar(&cfg.backend, "backend", getEnv("EMBEDDING_BACKEND", "ollama"), "Embedding backend: azure or ollama")
	flag.StringVar(&cfg.azureEndpoint, "azure-endpoint", getEnv("AZURE_OPENAI_ENDPOINT", ""), "Azure OpenAI endpoint (required for azure backend)")
	flag.StringVar(&cfg.azureKey, "azure-key", getEnv("AZURE_OPENAI_KEY", ""), "Azure OpenAI API key (required for azure backend)")
	flag.StringVar(&cfg.ollamaBaseURL, "ollama-url", getEnv("OLLAMA_BASE_URL", "http://localhost:11434/api"), "Ollama base URL (for ollama backend)")
	flag.StringVar(&cfg.model, "model", getEnv("EMBEDDING_MODEL", "nomic-embed-text"), "Embedding model name")
	flag.StringVar(&cfg.queryLogDir, "query-log-dir", getEnv("QUERY_LOG_DIR", ""), "Directory for query log files (empty = no logging)")
	flag.BoolVar(&cfg.printInstructions, "print-instructions", false, "Print prompt instructions and exit")

	flag.Parse()

	// Use embedded database if no path provided
	cfg.useEmbedded = (cfg.dbPath == "")

	// Normalize backend
	cfg.backend = strings.ToLower(cfg.backend)

	// Validate backend-specific requirements
	switch cfg.backend {
	case "azure":
		if cfg.azureEndpoint == "" {
			log.Fatal("azure-endpoint is required for azure backend")
		}
		if cfg.azureKey == "" {
			log.Fatal("azure-key is required for azure backend")
		}
	case "ollama":
		if cfg.model == "" {
			log.Fatal("model is required for ollama backend")
		}
	default:
		log.Fatalf("unsupported backend: %s (use 'azure' or 'ollama')", cfg.backend)
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadEmbeddedDB loads the embedded database from memory.
func loadEmbeddedDB() (*chromem.DB, error) {
	// Create in-memory database
	db := chromem.NewDB()

	// Create a reader from the embedded bytes
	reader := bytes.NewReader(embeddedDBBytes)

	// Import from reader
	if err := db.ImportFromReader(reader, ""); err != nil {
		return nil, fmt.Errorf("failed to import embedded database: %w", err)
	}

	return db, nil
}

func run(cfg config) error {
	// Determine backend type
	var backend embedder.Backend
	switch cfg.backend {
	case "azure":
		backend = embedder.BackendAzureOpenAI
	case "ollama":
		backend = embedder.BackendOllama
	default:
		return fmt.Errorf("unsupported backend: %s", cfg.backend)
	}

	// Create query log file if directory specified
	var queryLogFile *os.File
	if cfg.queryLogDir != "" {
		// Ensure directory exists
		if err := os.MkdirAll(cfg.queryLogDir, 0755); err != nil {
			return fmt.Errorf("failed to create query log directory: %w", err)
		}
		
		// Create unique log file with timestamp and PID
		timestamp := time.Now().Format("20060102-150405")
		logFilename := fmt.Sprintf("queries-%s-%d.log", timestamp, os.Getpid())
		logPath := fmt.Sprintf("%s/%s", cfg.queryLogDir, logFilename)
		
		var err error
		queryLogFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open query log file: %w", err)
		}
		defer func() {
			if err := queryLogFile.Close(); err != nil {
				log.Printf("Warning: failed to close query log file: %v", err)
			}
		}()
		log.Printf("Query logging enabled: %s", logPath)
	}

	// Initialize MCP handler with either embedded DB or path
	var handler *mcp.Handler
	var err error

	if cfg.useEmbedded {
		log.Printf("Loading embedded database")
		db, dbErr := loadEmbeddedDB()
		if dbErr != nil {
			return fmt.Errorf("failed to load embedded database: %w", dbErr)
		}

		handler, err = mcp.NewHandler(mcp.Config{
			DB:            db,
			Backend:       backend,
			Endpoint:      cfg.azureEndpoint,
			APIKey:        cfg.azureKey,
			Model:         cfg.model,
			OllamaBaseURL: cfg.ollamaBaseURL,
			QueryLog:      queryLogFile,
		})
	} else {
		log.Printf("Loading external database: %s", cfg.dbPath)
		handler, err = mcp.NewHandler(mcp.Config{
			DBPath:        cfg.dbPath,
			Backend:       backend,
			Endpoint:      cfg.azureEndpoint,
			APIKey:        cfg.azureKey,
			Model:         cfg.model,
			OllamaBaseURL: cfg.ollamaBaseURL,
			QueryLog:      queryLogFile,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to create MCP handler: %w", err)
	}

	// Create MCP server
	server := mcpServer.NewMCPServer(
		"code-advisory-mcp",
		"1.0.0",
	)

	// Register tools and prompts
	handler.RegisterTools(server)
	handler.RegisterPrompts(server)

	log.Printf("Starting Code Advisory MCP Server")
	if cfg.useEmbedded {
		log.Printf("  Database: embedded (extracted to temp)")
	} else {
		log.Printf("  Database: %s", cfg.dbPath)
	}
	log.Printf("  Backend: %s", cfg.backend)
	log.Printf("  Model: %s", cfg.model)

	// Start server with stdio transport
	if err := mcpServer.ServeStdio(server); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func printPromptInstructions() {
	fmt.Println(embeddedInstructions)
}
