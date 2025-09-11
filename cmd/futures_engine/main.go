package main

import (
	"flag"
	"fmt"
	"frizo/futures_engine/internal/version"
	"os"
	"os/signal"
	"syscall"

	"frizo/futures_engine/internal/config"
	"frizo/futures_engine/internal/logger"
)

func main() {
	// Command line flags
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
		healthCheck = flag.Bool("health-check", false, "Perform health check")
		configFile  = flag.String("config", ".env.local", "Path to configuration file")
		logLevel    = flag.String("log-level", "", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		fmt.Printf("Futures Engine %s\n\n", version.Short())
		fmt.Println("Usage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Handle health check
	if *healthCheck {
		fmt.Println("OK")
		os.Exit(0)
	}

	// Load configuration
	cfg := config.Load()

	// Override log level from command line
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel)
	logger.SetDefault(log)

	// Log startup information
	log.Info("Starting Futures Engine",
		"version", version.Short(),
		"environment", cfg.Environment,
		"host", cfg.Host,
		"port", cfg.Port,
	)

	// Handle unused config file flag
	if *configFile != "" {
		log.Warn("Configuration file support not implemented yet", "file", *configFile)
	}

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start your application here
	log.Info("Futures Engine is running", "address", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))

	// Example of your main application logic
	if err := run(cfg, log); err != nil {
		log.Error("Application error", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	<-quit
	log.Info("Shutting down Futures Engine...")

	// Perform cleanup here
	cleanup(log)

	log.Info("Futures Engine stopped")
}

// run contains your main application logic
func run(cfg *config.Config, log *logger.Logger) error {
	// TODO: Implement your application logic here
	log.Info("Application started successfully")
	return nil
}

// cleanup performs cleanup operations
func cleanup(log *logger.Logger) {
	// TODO: Implement cleanup logic here
	log.Debug("Cleanup completed")
}
