#!/usr/bin/env python3
"""Run one oracle subprocess with an explicit wall-clock bound and group cleanup."""

import os
from pathlib import Path
import signal
import subprocess
import sys


def main() -> int:
    if len(sys.argv) < 5:
        print("usage: run-bounded.py SECONDS STDOUT STDERR COMMAND [ARG...]", file=sys.stderr)
        return 2
    seconds = float(sys.argv[1])
    stdout_path = Path(sys.argv[2])
    stderr_path = Path(sys.argv[3])
    command = sys.argv[4:]
    with stdout_path.open("wb") as stdout, stderr_path.open("wb") as stderr:
        child = subprocess.Popen(command, stdout=stdout, stderr=stderr, start_new_session=True)
        try:
            return child.wait(timeout=seconds)
        except subprocess.TimeoutExpired:
            stderr.write(f"oracle subprocess timed out after {seconds:g}s\n".encode())
            stderr.flush()
            try:
                os.killpg(child.pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
            try:
                child.wait(timeout=2)
            except subprocess.TimeoutExpired:
                pass
            # A child may exit while its descendants remain in its process group.
            try:
                os.killpg(child.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            child.wait()
            return 124


if __name__ == "__main__":
    raise SystemExit(main())
