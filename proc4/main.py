import sys
import argparse
from typing import List, Optional
import logging
from pythonjsonlogger import jsonlogger
from ddtrace import tracer
from ddtrace.context import Context
from pydantic import BaseModel

class Input(BaseModel):
    Params: List[int]
    Negate: bool
    TraceId: Optional[str] = None
    ParentSpanId: Optional[str] = None

class Output(BaseModel):
    Result: int

def configure_logger(log_file: str):
    logger = logging.getLogger('proc4')
    logger.setLevel(logging.INFO)
    
    formatter = jsonlogger.JsonFormatter('%(timestamp)s %(levelname)s %(name)s %(message)s')

    # Console handler
    ch = logging.StreamHandler()
    ch.setFormatter(formatter)
    logger.addHandler(ch)

    # File handler
    fh = logging.FileHandler(log_file)
    fh.setFormatter(formatter)
    logger.addHandler(fh)

    return logger

def read_stdin() -> str:
    return sys.stdin.read()

def process_input(logger: logging.Logger) -> int:
    logger.info("Hello from proc4")

    logger.info("Reading input JSON from stdin")
    json_input = read_stdin()
    logger.info("Read input JSON from stdin", extra={'input': json_input})

    try:
        inp = Input.model_validate_json(json_input)

        span_kwargs = {}
        if inp.TraceId and inp.ParentSpanId:
            context = Context(trace_id=int(inp.TraceId, 16), span_id=int(inp.ParentSpanId, 16))
            span_kwargs['child_of'] = context

        with tracer.start_span('proc4.process', **span_kwargs) as span:
            span.set_tag('input.params.count', len(inp.Params))
            span.set_tag('input.negate', str(inp.Negate))

            logger.info("Successfully parsed input JSON", extra={
                'dd.trace_id': span.trace_id,
                'dd.span_id': span.span_id,
                'params_count': len(inp.Params),
                'negate': inp.Negate,
            })

            logger.info("Calculating result", extra={
                'dd.trace_id': span.trace_id,
                'dd.span_id': span.span_id,
            })

            with tracer.start_span('proc4.calculate_result', child_of=span) as calc_span:
                result = sum(inp.Params)
                if inp.Negate:
                    result = -result

                calc_span.set_tag('result', result)

                logger.info("Result calculated", extra={
                    'dd.trace_id': calc_span.trace_id,
                    'dd.span_id': calc_span.span_id,
                    'result': result
                })

                outp = Output(Result=result)
                print(outp.model_dump_json())

    except ValueError as e:
        logger.error("Failed to parse JSON", extra={'error': str(e)})
        return 2  # Return non-zero status code for JSON parsing exception
    except Exception as e:
        logger.error("An unexpected error occurred", extra={'error': str(e)})
        return 3  # Return non-zero status code for unexpected exceptions

    return 0  # Return 0 for successful execution

def main():
    parser = argparse.ArgumentParser(description="Process JSON input and output result")
    parser.add_argument('--log-file', default='proc4.log', help='The file path for the log file')
    args = parser.parse_args()

    logger = configure_logger(args.log_file)
    exit_code = process_input(logger)
    sys.exit(exit_code)

if __name__ == "__main__":
    main()