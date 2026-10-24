#!/usr/bin/env python3
"""Bound one POSIX process group and retain raw campaign evidence."""
import argparse
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import time

DEFAULTS = {"native": 65, "build": 300, "exec": 60}
ENV_KEYS = ("PATH", "JAVA_HOME", "GOMAXPROCS", "GOFLAGS", "GOTOOLCHAIN", "GOPROXY", "GOSUMDB",
            "GOCACHE", "GOMODCACHE", "GOPATH", "TMPDIR", "MAVEN_OPTS",
            "JAVA2GO_CAMPAIGN_CACHE", "JAVA2GO_CAMPAIGN_MAVEN_REPO")
DISK_FLOOR = 10 * 1024 ** 3


def write_record(path, record):
    pending = path.with_suffix(".pending")
    pending.write_text(json.dumps(record, indent=2, sort_keys=True) + "\n")
    pending.replace(path)


def output_digest(path):
    with path.open("rb") as stream:
        return {"bytes": path.stat().st_size, "sha256": hashlib.file_digest(stream, "sha256").hexdigest()}


def group_exists(pgid):
    try:
        os.killpg(pgid, 0)
        return True
    except ProcessLookupError:
        return False


def cleanup(proc):
    actions = []
    for sig, grace in ((signal.SIGTERM, 0.5), (signal.SIGKILL, 2)):
        if not group_exists(proc.pid):
            break
        try:
            os.killpg(proc.pid, sig)
        except ProcessLookupError:
            break
        actions.append(sig.name)
        deadline = time.monotonic() + grace
        while time.monotonic() < deadline:
            proc.poll()  # Reap the group leader before probing the group.
            if not group_exists(proc.pid):
                break
            time.sleep(0.02)
    proc.poll()
    return {"signals": actions, "group_absent": not group_exists(proc.pid),
            "leader_reaped": proc.returncode is not None}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--evidence", required=True, help="new durable directory")
    parser.add_argument("--cwd", required=True)
    parser.add_argument("--kind", choices=DEFAULTS, default="exec")
    parser.add_argument("--timeout", type=float, help="supervisor seconds; fixture contract overrides default")
    parser.add_argument("--native-timeout", default="55s", help="Go test deadline if argv has none")
    parser.add_argument("--env", action="append", default=[], metavar="KEY=VALUE")
    parser.add_argument("--record-env", action="append", default=[], metavar="KEY")
    parser.add_argument("argv", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    argv = args.argv[1:] if args.argv[:1] == ["--"] else args.argv
    if not argv:
        parser.error("command argv is required after --")
    timeout = args.timeout if args.timeout is not None else DEFAULTS[args.kind]
    if not math.isfinite(timeout) or timeout <= 0:
        parser.error("timeout must be finite and positive")
    env = os.environ.copy()
    env["GOMAXPROCS"] = "2"
    env["JAVA_HOME"] = "/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home"
    env["GOCACHE"] = str(Path(__file__).resolve().parents[2] / "cache/go-build")
    env["GOFLAGS"] = env.get("GOFLAGS", "") + " -mod=readonly"
    env["GOPROXY"] = "off"
    env["GOSUMDB"] = "off"
    env["GOTOOLCHAIN"] = "local"
    selected = set(ENV_KEYS) | set(args.record_env)
    for assignment in args.env:
        key, sep, value = assignment.partition("=")
        if not sep or not key or key in ("HOME", "CODEX_HOME"):
            parser.error("--env needs KEY=VALUE and cannot override HOME/CODEX_HOME")
        env[key] = value
        selected.add(key)
    requested_argv = argv.copy()
    if Path(argv[0]).name == "go" and len(argv) > 1 and argv[1] in ("test", "build", "run", "install"):
        additions = []
        if not any(v == "-p" or v.startswith("-p=") for v in argv[2:]):
            additions.append("-p=1")
        if args.kind == "native" and argv[1] == "test" and not any(v == "-timeout" or v.startswith("-timeout=") for v in argv[2:]):
            additions.append("-timeout=" + args.native_timeout)
        argv[2:2] = additions
    cwd = Path(args.cwd).resolve(strict=True)
    evidence = Path(args.evidence).resolve()
    if evidence == Path("/private/tmp") or Path("/private/tmp") in evidence.parents or evidence == Path("/tmp") or Path("/tmp") in evidence.parents:
        parser.error("evidence must be durable, outside /private/tmp")
    evidence.mkdir(parents=True, exist_ok=False)
    record_path = evidence / "result.json"
    record = {"requested_argv": requested_argv, "argv": argv, "cwd": str(cwd),
              "env": {k: env.get(k) for k in sorted(selected)}, "kind": args.kind,
              "timeout_seconds": timeout, "native_timeout": args.native_timeout,
              "disk_floor_bytes": DISK_FLOOR, "started_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
              "timed_out": False, "exit_code": None, "runner_exit": 125,
              "cleanup": {"signals": [], "group_absent": True, "leader_reaped": True}}
    stdout_path, stderr_path = evidence / "stdout", evidence / "stderr"
    stdout_path.touch()
    stderr_path.touch()
    start = time.monotonic()
    proc = None
    previous_handlers = {}

    def interrupted(sig, _frame):
        record["interrupted_signal"] = sig
        raise KeyboardInterrupt

    try:
        record["free_disk_bytes"] = {str(p): shutil.disk_usage(p).free for p in (cwd, evidence)}
        if min(record["free_disk_bytes"].values()) < DISK_FLOOR:
            raise RuntimeError("available disk below 10 GiB floor")
        Path(env["GOCACHE"]).mkdir(parents=True, exist_ok=True)
        write_record(record_path, record)
        for sig in (signal.SIGTERM, signal.SIGINT):
            previous_handlers[sig] = signal.signal(sig, interrupted)
        with stdout_path.open("wb") as out, stderr_path.open("wb") as err:
            proc = subprocess.Popen(argv, cwd=cwd, env=env, stdin=subprocess.DEVNULL,
                                    stdout=out, stderr=err, start_new_session=True)
            record["pid"] = record["process_group"] = proc.pid
            write_record(record_path, record)
            try:
                proc.wait(timeout=timeout)
            except subprocess.TimeoutExpired:
                record["timed_out"] = True
            finally:
                record["cleanup"] = cleanup(proc)
                record["exit_code"] = proc.returncode
        record["runner_exit"] = 124 if record["timed_out"] else (proc.returncode if proc.returncode >= 0 else 128 - proc.returncode)
        if not record["cleanup"]["group_absent"] or not record["cleanup"]["leader_reaped"]:
            record["runner_exit"] = 125
    except (Exception, KeyboardInterrupt) as exc:
        record["error"] = repr(exc)
        if proc is not None and record["exit_code"] is None:
            record["cleanup"] = cleanup(proc)
            record["exit_code"] = proc.returncode
    finally:
        for sig, handler in previous_handlers.items():
            signal.signal(sig, handler)
        record["duration_seconds"] = round(time.monotonic() - start, 6)
        record["finished_utc"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        record["output"] = {p.name: output_digest(p) for p in (stdout_path, stderr_path)}
        write_record(record_path, record)
    print(str(record_path))
    return record["runner_exit"]


if __name__ == "__main__":
    sys.exit(main())
