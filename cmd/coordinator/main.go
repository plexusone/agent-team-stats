// Package main provides the coordinator entrypoint for the statistics team.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jessevdk/go-flags"
	worker "github.com/plexusone/omniagent-worker"
	"github.com/plexusone/omniagent-worker/server"
	"github.com/plexusone/omniobserve/agentops"

	// Register postgres provider for agentops
	_ "github.com/plexusone/omniobserve/agentops/postgres"

	"github.com/plexusone/agent-team-stats/coordinator"
	"github.com/plexusone/agent-team-stats/pkg/config"
)

// Options defines the CLI options structure
type Options struct {
	Port        int    `short:"p" long:"port" default:"8080" description:"HTTP server port"`
	Verbose     bool   `short:"v" long:"verbose" description:"Show verbose debug information"`
	Version     bool   `long:"version" description:"Show version information"`
	AgentOpsDSN string `long:"agentops-dsn" env:"AGENTOPS_DSN" description:"AgentOps PostgreSQL DSN for workflow tracing"`
}

const version = "2.0.0"

func main() {
	var opts Options

	parser := flags.NewParser(&opts, flags.Default)
	parser.LongDescription = `Statistics Coordinator - Multi-Agent Orchestration Server

This server runs the statistics team coordinator using omniagent-worker.
It manages three workers in-process:
  - Research: Web search for statistics sources
  - Synthesis: LLM-based statistics extraction
  - Verification: Source validation

ENDPOINTS:
  POST /execute    Execute the statistics workflow
  GET  /health     Health check
  GET  /ready      Readiness check
  GET  /info       Coordinator information

ENVIRONMENT VARIABLES:
  LLM_PROVIDER      LLM provider (gemini, claude, openai, ollama)
  GEMINI_API_KEY    API key for Gemini
  CLAUDE_API_KEY    API key for Claude
  OPENAI_API_KEY    API key for OpenAI
  SEARCH_PROVIDER   Search provider (serper, serpapi)
  SERPER_API_KEY    API key for Serper
  SERPAPI_API_KEY   API key for SerpAPI
  AGENTOPS_DSN      PostgreSQL DSN for AgentOps tracing (optional)

EXAMPLES:
  stats-coordinator                    # Run on default port 8080
  stats-coordinator -p 8000            # Run on port 8000
  stats-coordinator --verbose          # Run with debug logging
  stats-coordinator --agentops-dsn "postgres://user:pass@localhost/agentops"
`

	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			}
		}
		os.Exit(1)
	}

	if opts.Version {
		fmt.Printf("stats-coordinator version %s\n", version)
		fmt.Println("Built with omniagent-worker")
		os.Exit(0)
	}

	// Configure logging
	logLevel := slog.LevelInfo
	if opts.Verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg := config.LoadConfig()

	// Configure AgentOps if DSN provided
	var agentOpsStore agentops.Store
	agentOpsEnabled := false
	if opts.AgentOpsDSN != "" {
		var err error
		agentOpsStore, err = agentops.Open("postgres", agentops.WithDSN(opts.AgentOpsDSN))
		if err != nil {
			logger.Error("failed to open AgentOps store", "error", err)
			os.Exit(1)
		}
		defer agentOpsStore.Close()
		agentOpsEnabled = true
		logger.Info("AgentOps tracing enabled")
	}

	// Create coordinator
	coord := coordinator.New(coordinator.Config{
		AppConfig: cfg,
		AgentOps: &worker.AgentOpsConfig{
			Enabled: agentOpsEnabled,
			Store:   agentOpsStore,
		},
	})

	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("received shutdown signal")
		cancel()
	}()

	// Initialize coordinator
	logger.Info("initializing coordinator")
	if err := coord.Init(ctx); err != nil {
		logger.Error("failed to initialize coordinator", "error", err)
		os.Exit(1)
	}

	// Run server
	logger.Info("starting coordinator server",
		"port", opts.Port,
		"workers", coord.Pool().IDs(),
	)

	if err := server.Run(ctx, coord, server.Config{
		Port:   opts.Port,
		Logger: logger,
	}); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	// Shutdown coordinator
	logger.Info("shutting down coordinator")
	if err := coord.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("coordinator stopped")
}
