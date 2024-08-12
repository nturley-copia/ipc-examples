package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
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
	// Start the Datadog tracer
	tracer.Start(
		tracer.WithServiceName("proc1"),
		tracer.WithEnv("production"),
		tracer.WithServiceVersion(version),
	)
	defer tracer.Stop()

	// Create a root context
	ctx := context.Background()

	// Start a span for the entire program execution
	span, ctx := tracer.StartSpanFromContext(ctx, "proc1.execute")
	defer span.Finish()

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

	slog.InfoContext(ctx, "Application started", "version", version)

	// Read all input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to read from stdin", "error", err)
		span.SetTag("error", err.Error())
		os.Exit(1)
	}

	// Parse the input as JSON into InputData struct
	var inputData InputData
	parseSpan, _ := tracer.StartSpanFromContext(ctx, "proc1.parse_input")
	err = json.Unmarshal(input, &inputData)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse JSON", "error", err)
		parseSpan.SetTag("error", err.Error())
		parseSpan.Finish()
		os.Exit(1)
	}
	parseSpan.SetTag("params.count", len(inputData.Params))
	parseSpan.Finish()

	slog.InfoContext(ctx, "Parsed input data", "data", inputData)

	// Calculate the sum of params
	calcSpan, _ := tracer.StartSpanFromContext(ctx, "proc1.calculate_sum")
	sum := 0
	for _, num := range inputData.Params {
		sum += num
	}
	calcSpan.SetTag("sum", sum)
	calcSpan.Finish()

	// Create output data
	outputData := OutputData{
		Name:    inputData.Name,
		Sum:     sum,
		Version: version,
	}

	// Marshal output data to JSON
	marshalSpan, _ := tracer.StartSpanFromContext(ctx, "proc1.marshal_output")
	outputJSON, err := json.Marshal(outputData)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal output JSON", "error", err)
		marshalSpan.SetTag("error", err.Error())
		marshalSpan.Finish()
		os.Exit(1)
	}
	marshalSpan.Finish()

	// Write output JSON to stdout
	fmt.Println(string(outputJSON))

	slog.InfoContext(ctx, "Processing completed", "input_name", inputData.Name, "output_sum", sum)
}
