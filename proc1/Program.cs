using Datadog.Trace;
using Serilog;
using Serilog.Context;
using Serilog.Formatting.Json;
using System.CommandLine;
using System.Text.Json;

class Program
{
    static async Task<int> Main(string[] args)
    {
        var logFileOption = new Option<string>(
            name: "--log-file",
            description: "The file path for the log file.",
            getDefaultValue: () => "proc1.log");

        var rootCommand = new RootCommand("Process JSON input and output result");
        rootCommand.AddOption(logFileOption);

        rootCommand.SetHandler(async (string logFile) =>
        {
            ConfigureLogger(logFile);
            Environment.ExitCode = await ProcessInput();
        }, logFileOption);

        return await rootCommand.InvokeAsync(args);
    }

    static void ConfigureLogger(string logFile)
    {
        Log.Logger = new LoggerConfiguration()
            .Enrich.FromLogContext()
            .WriteTo.Console(new JsonFormatter(), standardErrorFromLevel: Serilog.Events.LogEventLevel.Verbose)
            .WriteTo.File(new JsonFormatter(), logFile, rollOnFileSizeLimit: true, fileSizeLimitBytes: 1000000)
            .CreateLogger();
    }

    static async Task<int> ProcessInput()
    {
        Log.Information("Hello from proc1");

        Log.Information("Reading input JSON from stdin...");
        string json = await Console.In.ReadToEndAsync();

        try
        {
            var inp = JsonSerializer.Deserialize<Input>(json);
            if (inp == null)
            {
                Log.Error("Failed to parse JSON: deserialized object is null");
                return 1; // Return non-zero status code for parsing error
            }

            SpanContext? parentContext = null;
            if (!string.IsNullOrEmpty(inp.TraceId) && !string.IsNullOrEmpty(inp.ParentSpanId))
            {
                if (ulong.TryParse(inp.TraceId, out ulong traceId) && ulong.TryParse(inp.ParentSpanId, out ulong spanId))
                {
                    parentContext = new SpanContext(traceId, spanId);
                }
                else
                {
                    Log.Warning("Invalid TraceId or ParentSpanId format. Using new trace context.");
                }
            }

            var startSpanOptions = new SpanCreationSettings
            {
                Parent = parentContext
            };

            using (var scope = Tracer.Instance.StartActive("proc1.process", startSpanOptions))
            {
                // Push trace and span IDs to the log context
                using (LogContext.PushProperty("dd.trace_id", scope.Span.TraceId.ToString()))
                using (LogContext.PushProperty("dd.span_id", scope.Span.SpanId.ToString()))
                using (LogContext.PushProperty("Params", inp.Params))
                using (LogContext.PushProperty("Negate", inp.Negate))
                {
                    Log.Information("Successfully input JSON");
                    scope.Span.SetTag("input.params.count", inp.Params.Length);
                    scope.Span.SetTag("input.negate", inp.Negate.ToString());

                    Log.Information("Calculating result...");
                    using (var calcScope = Tracer.Instance.StartActive("proc1.calculate_result"))
                    using (LogContext.PushProperty("dd.span_id", calcScope.Span.SpanId.ToString()))
                    {
                        var outp = new Output { Result = inp.Params.Sum() };
                        if (inp.Negate)
                        {
                            outp.Result = -outp.Result;
                        }
                        calcScope.Span.SetTag("result", outp.Result);
                        Log.Information("Result: {Result}", outp.Result);
                        Console.WriteLine(JsonSerializer.Serialize(outp));
                    }
                }
            }
        }
        catch (JsonException ex)
        {
            Log.Error(ex, "Failed to parse JSON");
            return 2; // Return non-zero status code for JSON parsing exception
        }
        catch (Exception ex)
        {
            Log.Error(ex, "An unexpected error occurred");
            return 3; // Return non-zero status code for unexpected exceptions
        }

        return 0; // Return 0 for successful execution
    }
}

class Input
{
    public required int[] Params { get; set; }
    public bool Negate { get; set; }
    public string? TraceId { get; set; }
    public string? ParentSpanId { get; set; }
}

class Output
{
    public int Result { get; set; }
}