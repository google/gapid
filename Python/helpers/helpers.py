import traceback
import sys
from . import trace_env

def on_error(trace_env, message):
    print(f"Error: {message}")
    for line in traceback.extract_stack(limit=10):
        print(f"    {line.filename}:{line.lineno}")
    sys.exit(-1)

def status_message(trace_env, message):
    print(f"Status: {message}")

def log_message(trace_env, message):
    print(f"{message}")