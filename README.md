# IPC Examples

When you are writing a one-shot CLI that executes a single function and then terminates

## Guidelines

### TLDR

|                      | Typescript          | Go            | Python             | C#                 |
|----------------------|---------------------|---------------|--------------------|--------------------|
| Logging              | winston             | slog          | python-json-logger | Serilog            |
| CLI Argument Parsing | Commander           | flag          | argparse           | System.Commandline |
| JSON Parsing         | JSON.parse with Zod | encoding/json | pydantic           | System.Text.Json   |


### Development/Documentation

Make it easy for developers unfamiliar with this CLI to run and debug this code

 * README.md explains how to run the CLI locally for development and debugging purposes 
 * Isolate OS-dependent code. Keep as much business logic out of that code as possible, so that it can be developed and tested on Windows, Linux, and OSX
 * Add a `launch.json` file to make it easy to run in debug mode from VS Code
 * example test input files should be in the repo and we should validate those examples as test cases in CI to keep them up-to-date
 * Fatal errors should set the status code to a non-zero value

### Configuration

 * Configuration parameters are the data that is the same between repeated invocations such as log path, log level, environment name, etc
 * Configuration parameters should have reasonable defaults, but modifiable through the CLI args or environment variables

### Inputs

 * Input parameters are data that frequently change, for example, every time the executable is invoked
 * Pass input parameters through stdin in JSON format
   * JSON is standardized, OS and shell independent, and easily parsed in any language
   * It's trivial to pipe a file into stdin, so it's equivalent to a config file
   * It's faster to read from stdin than disk
   * You can chain together executables by piping the stdout of one tool into the stdin of another tool (ie jq)
 * You should be able to provide trace id, parent span id, and other log context data as input data
 * Default to serializing and deserializing classes to JSON instead of working with JSON document object hierarchies.

### Outputs

 * Pass structured output of the app to stdout in JSON format
 * The structured output should include the version of this CLI (release number for release builds, commit SHA for dev builds)
 * binary output can be written to a file, and structured output would define the path of that file

### Logging

 * All logs should be JSON
 * All logs should include the version of the CLI (release number for release builds, commit SHA for dev builds)
 * Logs can go to stderr and a file but never stdout.
   * stderr is usually not buffered and so it's less likely to break up the log statements
 * Lower log levels can be filtered from stderr if it's too noisy but ALL log levels must always be in the file
 * Each CLI invocation should just append onto the same log file
 * By default the log file name is `<CLI NAME>.log` and the log rotates by time or file size
 * The log file path is configurable so that we can relocate it, if necessary, due to user permissions issues
  * If you are calling into third party executables that generate unstructured logs, it is your responsibility to parse them and convert them into JSON
  * Give them their own span ID, and source name, but it's okay to include them in your own log file
 * If a third party library is dumping unstructured text to your stdout, find a way to redirect it into your log stream somehow
   * Otherwise, you'll need to put delimiters around it so the process that invoked you can separate your output vs the unstructured output from the library
 
 ### Traces

 * You should be generating traces, not just logs.
 * If you are not using a tracer, all logs should include a parent trace id and span id if they are given one in their input parameters

# Language Specific Guidelines

## C# Guidelines

* Use as modern of .NET like .NET 8.0. Try to stay away from .NET Framework if possible.
* Avoid relying on Visual Studio, make it buildable from the `dotnet` CLI
* Serilog for logging
* Parse JSON with `System.Text.Json`, if you have to use an old version of .NET, then use the NewtonSoft JSON library
* System.Commandline for parsing arguments

## Go Guidelines

* If possible, use the built-in structured logger from the standard library (slog), otherwise use logrus
* Use `encoding/json` to deserialize and serialize JSON
* Use `flag` to parse command line arguments

## Typescript Guidelines

* Winston for logging
  * Use the AsyncLocalStorage API to attach trace ids and span ids to all log statements in a code block
* Commander for argument parsing
* Use JSON.parse and zod for JSON parsing and validation

## Python Guidelines
* Poetry for virtual environment management
* argparse for command line argument parsing
* python-json-logger for JSON formatted logs
* pydantic for parsing and validating JSON