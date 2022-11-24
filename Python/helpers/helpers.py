import traceback
import sys
from . import gapid_env

def on_error(gapid_env, message):
    print(f"Error: {message}")
    for line in traceback.extract_stack(limit=10):
        print(f"    {line.filename}:{line.lineno}")
    sys.exit(-1)

def status_message(gapid_env, message):
    print(f"Status: {message}")