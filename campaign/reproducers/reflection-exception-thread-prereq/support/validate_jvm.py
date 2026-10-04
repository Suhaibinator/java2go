#!/usr/bin/env python3
"""JDK oracle runner; execute only in a coordinator-assigned heavy slot."""

from __future__ import annotations

import hashlib
import json
import os
import signal
import subprocess
import time
from pathlib import Path


SOURCE = Path("/private/tmp/java2go-reflection-exception-thread-prereq")
OUT = Path("/private/tmp/java2go-reflection-exception-thread-validation")
JDK = Path("/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home")
EXPECTED_INPUT_SHA = "a64df8bc84345ba2c8f4f1f8340cdd33b535e737f1203fd2ef313a07b594a42e"


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def input_sha() -> str:
    digest = hashlib.sha256()
    for path in sorted(p for p in SOURCE.rglob("*") if p.is_file()):
        digest.update(path.relative_to(SOURCE).as_posix().encode() + b"\0")
        digest.update(path.read_bytes() + b"\0")
    return digest.hexdigest()


def run(name: str, command: list[str], cwd: Path, timeout: int) -> dict:
    started = time.monotonic()
    stdout, stderr = b"", b""
    code, timed_out, infrastructure_error = None, False, None
    try:
        process = subprocess.Popen(command, cwd=cwd, stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE, start_new_session=True)
        try:
            stdout, stderr = process.communicate(timeout=timeout)
            code = process.returncode
        except subprocess.TimeoutExpired:
            timed_out = True
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            except OSError as error:
                infrastructure_error = f"killpg failed: {error}"
            try:
                stdout, stderr = process.communicate(timeout=10)
            except subprocess.TimeoutExpired:
                infrastructure_error = "process group survived timeout cleanup"
    except OSError as error:
        infrastructure_error = f"process start failed: {error}"
    stdout_path = OUT / f"{name}.stdout"
    stderr_path = OUT / f"{name}.stderr"
    stdout_path.write_bytes(stdout)
    stderr_path.write_bytes(stderr)
    return {"name": name, "command": command, "cwd": str(cwd),
            "timeout_seconds": timeout, "duration_seconds": time.monotonic() - started,
            "exit": code, "timeout": timed_out,
            "infrastructure_error": infrastructure_error,
            "stdout": str(stdout_path), "stderr": str(stderr_path),
            "stdout_sha256": sha(stdout), "stderr_sha256": sha(stderr),
            "stdout_bytes": len(stdout), "stderr_bytes": len(stderr)}


def main() -> None:
    if OUT.exists():
        raise SystemExit(f"Refusing to replace previous validation: {OUT}")
    OUT.mkdir()
    (OUT / "classes").mkdir()
    report = {"status": "unvalidated", "source": str(SOURCE),
              "input_sha256_before": input_sha(), "compile": None,
              "runs": [], "stable": False}
    if report["input_sha256_before"] != EXPECTED_INPUT_SHA:
        report["status"] = "input-hash-mismatch"
    else:
        files = sorted((SOURCE / "src/main/java").rglob("*.java"))
        command = [str(JDK / "bin/javac"), "--release", "21", "-encoding",
                   "UTF-8", "-d", str(OUT / "classes"),
                   *[str(path) for path in files]]
        report["compile"] = run("javac", command, SOURCE, 300)
        compiled = report["compile"]
        if (compiled["exit"] == 0 and not compiled["timeout"]
                and compiled["infrastructure_error"] is None):
            for seed in (17, 41, 97):
                for repetition in (1, 2, 3):
                    name = f"seed-{seed}-run-{repetition}"
                    cwd = OUT / "work" / name
                    cwd.mkdir(parents=True)
                    result = run(name, [str(JDK / "bin/java"), "-cp",
                                        str(OUT / "classes"),
                                        "prereq.reflection.app.Main", str(seed)],
                                 cwd, 60)
                    result["unexpected_outputs"] = sorted(
                        path.relative_to(cwd).as_posix() for path in cwd.rglob("*"))
                    result["seed"] = seed
                    result["repetition"] = repetition
                    report["runs"].append(result)
        report["input_sha256_after"] = input_sha()
        stable = (report["input_sha256_after"] == EXPECTED_INPUT_SHA
                  and report["compile"]["exit"] == 0
                  and not report["compile"]["timeout"]
                  and report["compile"]["infrastructure_error"] is None
                  and len(report["runs"]) == 9)
        for seed in (17, 41, 97):
            selected = [item for item in report["runs"] if item["seed"] == seed]
            stable = stable and len(selected) == 3 and all(
                item["exit"] == 0 and not item["timeout"]
                and item["infrastructure_error"] is None
                and item["stderr_bytes"] == 0 and not item["unexpected_outputs"]
                for item in selected)
            stable = stable and len({(item["exit"], item["stdout_sha256"],
                                     item["stderr_sha256"]) for item in selected}) == 1
        report["stable"] = stable
        report["status"] = "validated" if stable else "invalid-or-unstable"
        if stable:
            for seed in (17, 41, 97):
                item = next(x for x in report["runs"] if x["seed"] == seed)
                (OUT / f"expected.seed-{seed}.stdout").write_bytes(
                    Path(item["stdout"]).read_bytes())
                (OUT / f"expected.seed-{seed}.stderr").write_bytes(
                    Path(item["stderr"]).read_bytes())
    report["input_sha256_after"] = input_sha()
    (OUT / "validation.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"status": report["status"], "stable": report["stable"],
                      "input_sha256_before": report["input_sha256_before"],
                      "input_sha256_after": report["input_sha256_after"],
                      "report": str(OUT / "validation.json")}))
    raise SystemExit(0 if report["stable"] else 2)


if __name__ == "__main__":
    main()
