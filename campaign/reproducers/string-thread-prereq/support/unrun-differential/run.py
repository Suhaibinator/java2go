#!/usr/bin/env python3
"""Strict frozen16v3 differential; run only in an assigned heavy slot."""

from __future__ import annotations

import hashlib
import json
import os
import signal
import shutil
import subprocess
import time
from pathlib import Path


HERE = Path(__file__).resolve().parent
SOURCE = Path("/private/tmp/java2go-string-thread-prereq")
ORACLE = Path("/private/tmp/java2go-string-thread-prereq-validation")
SNAPSHOT = Path("/private/tmp/java2go-checkpoint16-v3-ui52b9_f")
SNAPSHOT_MANIFEST = Path("/private/tmp/java2go-checkpoint16-v3-manifest.json")
EXPECTED_SOURCE_SHA = "206b89997fca211711a90e504d29dc1f314a674cfa6a63cb8f7a53037fc684cf"
SOURCE_ROOT = SOURCE / "src/main/java"
OUT = HERE / "artifacts"
STAGE = OUT / "maven-input"
GENERATED = OUT / "generated"
BIN = OUT / "bin"
REPORT = OUT / "report.json"
PIN = HERE / "pin.json"
JDK = Path("/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home")
GO_CACHE = Path("/private/tmp/java2go-campaign-go-cache")


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def source_hash() -> str:
    digest = hashlib.sha256()
    for path in sorted(p for p in SOURCE.rglob("*") if p.is_file()):
        digest.update(path.relative_to(SOURCE).as_posix().encode() + b"\0")
        digest.update(path.read_bytes() + b"\0")
    return digest.hexdigest()


def snapshot_mismatches() -> list[str]:
    manifest = json.loads(SNAPSHOT_MANIFEST.read_text())
    return [relative for relative, expected in manifest.items()
            if not (SNAPSHOT / relative).is_file()
            or sha((SNAPSHOT / relative).read_bytes()) != expected]


def observe(name: str, command: list[str], cwd: Path, timeout: int,
            env: dict[str, str]) -> dict:
    started = time.monotonic()
    stdout, stderr = b"", b""
    exit_code, expired = None, False
    infrastructure_error = None
    try:
        process = subprocess.Popen(command, cwd=cwd, env=env,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   start_new_session=True)
        try:
            stdout, stderr = process.communicate(timeout=timeout)
            exit_code = process.returncode
        except subprocess.TimeoutExpired:
            expired = True
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            except OSError as error:
                infrastructure_error = f"failed to kill process group: {error}"
            try:
                stdout, stderr = process.communicate(timeout=10)
            except subprocess.TimeoutExpired:
                infrastructure_error = "process group survived SIGKILL cleanup"
    except OSError as error:
        infrastructure_error = f"failed to start process: {error}"
    stdout_path = OUT / f"{name}.stdout"
    stderr_path = OUT / f"{name}.stderr"
    stdout_path.write_bytes(stdout)
    stderr_path.write_bytes(stderr)
    return {"name": name, "command": command, "cwd": str(cwd),
            "timeout_seconds": timeout, "duration_seconds": time.monotonic() - started,
            "exit": exit_code, "timeout": expired,
            "infrastructure_error": infrastructure_error,
            "stdout": str(stdout_path), "stderr": str(stderr_path),
            "stdout_sha256": sha(stdout), "stderr_sha256": sha(stderr),
            "stdout_bytes": len(stdout), "stderr_bytes": len(stderr)}


def save(report: dict) -> None:
    report["input_sha256_after"] = source_hash()
    report["snapshot_mismatches_after"] = snapshot_mismatches()
    report["runner_sha256_after"] = sha(Path(__file__).read_bytes())
    report["pom_sha256_after"] = sha((HERE / "pom.xml").read_bytes())
    errors = []
    if report["input_sha256_after"] != EXPECTED_SOURCE_SHA:
        errors.append("frozen Java input changed")
    if report["snapshot_mismatches_after"]:
        errors.append("frozen16v3 compiler snapshot changed")
    if report["runner_sha256_after"] != report["pins"]["runner_sha256"]:
        errors.append("differential runner changed")
    if report["pom_sha256_after"] != report["pins"]["pom_sha256"]:
        errors.append("pinned Maven POM changed")
    report["integrity_errors"] = errors
    report["invalidated"] = bool(errors)
    REPORT.write_text(json.dumps(report, indent=2) + "\n")


def stop(report: dict, failure: str) -> None:
    report["first_discrepancy"] = failure
    save(report)
    print(json.dumps({"first_discrepancy": failure,
                      "invalidated": report["invalidated"],
                      "integrity_errors": report["integrity_errors"],
                      "report": str(REPORT)}))
    raise SystemExit(3 if report["invalidated"] else 2)


def main() -> None:
    if OUT.exists():
        raise SystemExit(f"Refusing to replace existing artifacts: {OUT}")
    OUT.mkdir()
    BIN.mkdir()
    pins = json.loads(PIN.read_text())
    report = {"source": str(SOURCE), "pom": str(HERE / "pom.xml"),
              "pins": pins, "pin_manifest_sha256": sha(PIN.read_bytes()),
              "runner_sha256_before": sha(Path(__file__).read_bytes()),
              "pom_sha256_before": sha((HERE / "pom.xml").read_bytes()),
              "oracle": str(ORACLE / "oracle.json"),
              "snapshot": str(SNAPSHOT), "snapshot_manifest": str(SNAPSHOT_MANIFEST),
              "stages": [], "go_runs": [], "first_discrepancy": None}
    if (report["runner_sha256_before"] != pins["runner_sha256"]
            or report["pom_sha256_before"] != pins["pom_sha256"]):
        stop(report, "runner or POM pin mismatch before execution")
    report["input_sha256_before"] = source_hash()
    if report["input_sha256_before"] != EXPECTED_SOURCE_SHA:
        stop(report, "frozen input hash changed before execution")
    mismatches = snapshot_mismatches()
    report["snapshot_mismatches_before"] = mismatches
    if mismatches:
        stop(report, "frozen16v3 snapshot manifest mismatch before execution")

    oracle = json.loads((ORACLE / "oracle.json").read_text())
    validation = json.loads((ORACLE / "validation.json").read_text())
    if (oracle["input_sha256_path_nul_bytes_nul"] != EXPECTED_SOURCE_SHA
            or validation["source_sha256_before"] != EXPECTED_SOURCE_SHA
            or validation["source_sha256_after"] != EXPECTED_SOURCE_SHA
            or not validation["stable"]):
        stop(report, "frozen JVM source or validation metadata mismatch")
    jvm_runs = {(item["seed"], item["repetition"]): item
                for item in validation["runs"]}
    if set(jvm_runs) != {(seed, repetition) for seed in (17, 41, 97)
                         for repetition in (1, 2, 3)}:
        stop(report, "JVM nine-run matrix incomplete")
    for item in jvm_runs.values():
        if (item["exit"] != 0 or item["timeout"]
                or sha(Path(item["stdout_path"]).read_bytes()) != item["stdout_sha256"]
                or sha(Path(item["stderr_path"]).read_bytes()) != item["stderr_sha256"]):
            stop(report, "JVM observation artifact mismatch")

    shutil.copytree(SOURCE_ROOT, STAGE / "src/main/java")
    shutil.copyfile(HERE / "pom.xml", STAGE / "pom.xml")
    original = sorted(SOURCE_ROOT.rglob("*.java"))
    copied = sorted((STAGE / "src/main/java").rglob("*.java"))
    if len(original) != 7 or len(copied) != 7:
        stop(report, "staged source closure is not exactly seven classes")
    for path in original:
        if path.read_bytes() != (STAGE / "src/main/java" /
                                 path.relative_to(SOURCE_ROOT)).read_bytes():
            stop(report, f"staged source bytes differ: {path}")

    env = dict(os.environ)
    env["GOWORK"] = "off"
    env["GORACE"] = "halt_on_error=1"
    env["JAVA_HOME"] = str(JDK)
    env["PATH"] = str(JDK / "bin") + os.pathsep + env.get("PATH", "")
    env["GOMAXPROCS"] = "2"
    env["GOCACHE"] = str(GO_CACHE)
    env["TMPDIR"] = "/private/tmp"
    env["JAVA2GO_ASSERTIONS"] = "false"
    report["command_env"] = {key: env[key] for key in
                             ("JAVA_HOME", "GOMAXPROCS", "GOCACHE", "TMPDIR",
                              "GOWORK", "GORACE", "JAVA2GO_ASSERTIONS", "PATH")}
    stages = [
        ("compiler-build", ["go", "build", "-mod=readonly", "-o",
                            str(BIN / "java2go"), "./cmd/java2go"], SNAPSHOT, 300),
        ("strict-transpile", [str(BIN / "java2go"), "-strict", "-maven",
                              str(STAGE), "-main-class", "prereq.thread.app.Main",
                              "-runtime", str(SNAPSHOT), "-module",
                              "prereq.test/stringthread", "-output",
                              str(GENERATED)], SNAPSHOT, 300),
        ("all-packages-race-build", ["go", "build", "-race", "-mod=mod", "./..."],
         GENERATED, 300),
        ("app-race-build", ["go", "build", "-race", "-mod=mod", "-o",
                            str(BIN / "app"), "./cmd/app"], GENERATED, 300),
    ]
    for name, command, cwd, timeout in stages:
        result = observe(name, command, cwd, timeout, env)
        report["stages"].append(result)
        save(report)
        if result["infrastructure_error"]:
            stop(report, f"infrastructure:{name}")
        if result["timeout"] or result["exit"] != 0:
            stop(report, name)

    for seed in (17, 41, 97):
        for repetition in (1, 2, 3):
            name = f"go-seed-{seed}-run-{repetition}"
            execution_dir = OUT / "executions" / name
            execution_dir.mkdir(parents=True)
            result = observe(name, [str(BIN / "app"), str(seed)], execution_dir, 60, env)
            result["unexpected_outputs"] = sorted(
                path.relative_to(execution_dir).as_posix()
                for path in execution_dir.rglob("*"))
            expected = jvm_runs[(seed, repetition)]
            result["matches_jvm"] = (not result["timeout"]
                                     and result["infrastructure_error"] is None
                                     and not result["unexpected_outputs"]
                                     and result["exit"] == expected["exit"]
                                     and result["stdout_sha256"] == expected["stdout_sha256"]
                                     and result["stderr_sha256"] == expected["stderr_sha256"])
            report["go_runs"].append(result)
            if not result["matches_jvm"] and report["first_discrepancy"] is None:
                prefix = "infrastructure:" if result["infrastructure_error"] else ""
                report["first_discrepancy"] = prefix + name
            save(report)

    if report["invalidated"]:
        stop(report, "input, compiler, runner, or POM integrity invalidated")
    print(json.dumps({"first_discrepancy": report["first_discrepancy"],
                      "matching_runs": sum(item["matches_jvm"] for item in report["go_runs"]),
                      "total_runs": len(report["go_runs"]), "report": str(REPORT)}))
    if report["first_discrepancy"] is not None:
        raise SystemExit(2)


if __name__ == "__main__":
    main()
