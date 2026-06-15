package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aluoty/probe.git/internal/client"
	"github.com/aluoty/probe.git/internal/config"
	"github.com/aluoty/probe.git/internal/download"
	"github.com/aluoty/probe.git/internal/github"
	"github.com/aluoty/probe.git/internal/output"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cfg, err := config.Parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if cfg.ShowHelp {
		config.PrintUsage(os.Stderr)
		return 0
	}
	if cfg.ShowVersion {
		fmt.Println(config.Name, config.Version)
		return 0
	}

	if cfg.GitHub != "" {
		if err := github.Download(cfg); err != nil {
			return fail(cfg, err)
		}
		return 0
	}

	exitCode := 0
	for _, url := range cfg.URLs {
		reqCfg := cfg.Clone(url)
		if err := fetchOne(reqCfg); err != nil {
			exitCode = fail(reqCfg, err)
			if cfg.FailOnError {
				return exitCode
			}
		}
	}
	return exitCode
}

func fetchOne(cfg *config.Config) error {
	body, err := client.PrepareRequestBody(cfg)
	if err != nil {
		return err
	}

	if err := download.PrepareResume(cfg); err != nil {
		return err
	}

	start := time.Now()
	resp, err := client.Fetch(cfg, body)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return output.HandleResponse(cfg, resp, time.Since(start))
}

func fail(cfg *config.Config, err error) int {
	if !cfg.Silent || cfg.ShowErrors {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return 1
}
