package main

import (
	"fmt"
	"os"

	"github.com/aluoty/probe.git/internal/client"
	"github.com/aluoty/probe.git/internal/config"
	"github.com/aluoty/probe.git/internal/download"
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

	body, err := client.LoadRequestBody(cfg)
	if err != nil {
		return fail(cfg, err)
	}

	if err := download.PrepareResume(cfg); err != nil {
		return fail(cfg, err)
	}

	resp, err := client.Fetch(cfg, body)
	if err != nil {
		return fail(cfg, fmt.Errorf("request failed: %w", err))
	}
	defer resp.Body.Close()

	if err := output.HandleResponse(cfg, resp); err != nil {
		return fail(cfg, err)
	}

	return 0
}

func fail(cfg *config.Config, err error) int {
	if !cfg.Silent {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return 1
}
