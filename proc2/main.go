package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type InputData struct {
	Name   string `json:"name"`
	Params []int  `json:"params"`
}

type OutputData struct {
	Name    string `json:"name"`
	Sum     int    `json:"sum"`
	Version string `json:"version"`
}

const version = "1.0.0"

func main() {
	// Parse command-line arguments
	logFile := flag.String("log-file", "", "Path to the log file (optional)")
	flag.Parse()

	// Configure logger
	var logWriter io.Writer = os.Stderr
	if *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		logWriter = io.MultiWriter(file, os.Stderr)
	}

	logger := slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("Application started", "version", version)

	// Read all input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		slog.Error("Failed to read from stdin", "error", err)
		os.Exit(1)
	}

	// Parse the input as JSON into InputData struct
	var inputData InputData
	err = json.Unmarshal(input, &inputData)
	if err != nil {
		slog.Error("Failed to parse JSON", "error", err)
		os.Exit(1)
	}

	slog.Info("Parsed input data", "data", inputData)

	// Calculate the sum of params
	sum := 0
	for _, num := range inputData.Params {
		sum += num
	}

	// Create output data
	outputData := OutputData{
		Name:    inputData.Name,
		Sum:     sum,
		Version: version,
	}

	// Marshal output data to JSON
	outputJSON, err := json.Marshal(outputData)
	if err != nil {
		slog.Error("Failed to marshal output JSON", "error", err)
		os.Exit(1)
	}

	// Write output JSON to stdout
	fmt.Println(string(outputJSON))

	slog.Info("Processing completed", "input_name", inputData.Name, "output_sum", sum)
}
