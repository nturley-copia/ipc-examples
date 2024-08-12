import tracer from 'dd-trace';
import winston from 'winston';
import { Command } from 'commander';
import { z } from 'zod';
import fs from 'fs';
import { AsyncLocalStorage } from 'async_hooks';

// Initialize Datadog tracer
tracer.init();

// Create an AsyncLocalStorage instance for our trace context
const traceContext = new AsyncLocalStorage<{ traceId: string; spanId: string }>();

// Define schema for input validation
const InputSchema = z.object({
  Params: z.array(z.number()),
  Negate: z.boolean(),
  TraceId: z.string().optional(),
  ParentSpanId: z.string().optional(),
});

type Input = z.infer<typeof InputSchema>;

interface Output {
  Result: number;
}

let logger: winston.Logger;

function configureLogger(logFile: string) {
  const logFormat = winston.format.combine(
    winston.format.timestamp({
      format: 'YYYY-MM-DD HH:mm:ss.SSS'
    }),
    winston.format.errors({ stack: true }),
    winston.format.splat(),
    winston.format.json(),
    winston.format((info) => {
      const context = traceContext.getStore();
      if (context) {
        info['dd.trace_id'] = context.traceId;
        info['dd.span_id'] = context.spanId;
      }
      return info;
    })()
  );

  logger = winston.createLogger({
    level: 'info',
    format: logFormat,
    defaultMeta: { service: 'proc1' },
    transports: [
      new winston.transports.Console({
        level: 'verbose',
      }),
      new winston.transports.File({
        filename: logFile,
        maxsize: 1000000,
        maxFiles: 5,
      }),
    ],
  });
}

function readStdin(): string {
  try {
    return fs.readFileSync(process.stdin.fd, 'utf-8');
  } catch (error) {
    logger.error('Failed to read from stdin', { error: error instanceof Error ? error.message : String(error) });
    process.exit(1);
  }
}

async function processInput(): Promise<number> {
  logger.info('Reading input JSON from stdin...');
  const json = readStdin();

  try {
    const inp = InputSchema.parse(JSON.parse(json));

    let spanOptions: any = {};
    if (inp.TraceId && inp.ParentSpanId) {
      spanOptions.childOf = {
        traceId: inp.TraceId,
        spanId: inp.ParentSpanId
      };
    }

    const span = tracer.startSpan('proc1.process', spanOptions);

    return await traceContext.run({ traceId: span.context().toTraceId(), spanId: span.context().toSpanId() }, async () => {
      span.setTag('input.params.count', inp.Params.length);
      span.setTag('input.negate', inp.Negate.toString());

      logger.info('Successfully input JSON', {
        Params: inp.Params,
        Negate: inp.Negate,
      });

      logger.info('Calculating result...');

      const calcSpan = tracer.startSpan('proc1.calculate_result', { childOf: span });

      return await traceContext.run({ traceId: calcSpan.context().toTraceId(), spanId: calcSpan.context().toSpanId() }, async () => {
        const outp: Output = { Result: inp.Params.reduce((sum, num) => sum + num, 0) };
        if (inp.Negate) {
          outp.Result = -outp.Result;
        }

        calcSpan.setTag('result', outp.Result);

        logger.info(`Result: ${outp.Result}`);

        console.log(JSON.stringify(outp));

        calcSpan.finish();
        span.finish();

        return 0; // Return 0 for successful execution
      });
    });

  } catch (error) {
    if (error instanceof z.ZodError) {
      logger.error('Input validation failed', { error: error.errors });
      return 2;
    } else if (error instanceof SyntaxError) {
      logger.error('Failed to parse JSON', { error: error.message });
      return 3;
    } else {
      logger.error('An unexpected error occurred', { error: error instanceof Error ? error.message : String(error) });
      return 4;
    }
  }
}

async function main() {
  const program = new Command();

  program
    .option('--log-file <path>', 'The file path for the log file', 'proc1.log')
    .action(async (options) => {
      configureLogger(options.logFile);
      const exitCode = await processInput();
      process.exit(exitCode);
    });

  await program.parseAsync(process.argv);
}

main().catch((error) => {
  console.error('Unhandled error:', error);
  process.exit(1);
});