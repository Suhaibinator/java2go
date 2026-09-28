#!/usr/bin/env python3
"""Focused integration probe; this does not accept or alter a frozen challenge."""
import datetime
import hashlib
import json
import os
import re
from pathlib import Path
import shutil
import subprocess
import zipfile

ROOT = Path(__file__).resolve().parents[3]
MEMBER = "com/google/gson/internal/bind/util/ISO8601Utils.java"

def sha(data):
    return hashlib.sha256(data).hexdigest()

def implementation():
    revision = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, timeout=10).decode().strip()
    files = {}
    for name in ("api.go", "go.mod", "go.sum", "astutil", "nodeutil", "parsing", "project", "symbol", "stdjava", "transpiler", "cmd/java2go", "campaign", "cmd/javacampaign"):
        path = ROOT / name
        for item in ([path] if path.is_file() else path.rglob("*")):
            if not item.is_file() or item.name.endswith("_test.go"):
                continue
            if item.suffix == ".go" or item.name in ("go.mod", "go.sum") or item.name.endswith(".lock.json"):
                files[item.relative_to(ROOT).as_posix()] = sha(item.read_bytes())
    encoded = revision + "\n" + "".join(name + "\0" + files[name] + "\n" for name in sorted(files))
    return {"revision": revision, "sha256": sha(encoded.encode()), "files": files}

def main():
    lock = json.loads((ROOT / "campaign/dependencies.lock.json").read_text())
    artifact = next(a for a in lock["artifacts"] if a["id"] == "gson" and a["kind"] == "sources")
    archive = ROOT / ".campaign/cache" / artifact["file"]
    data = archive.read_bytes()
    if sha(data) != artifact["sha256"]:
        raise RuntimeError("Gson source archive does not match dependency lock")
    with zipfile.ZipFile(archive) as sources:
        source = sources.read(MEMBER)
    extracted = ROOT / ".campaign/sources" / ("gson-" + artifact["version"]) / MEMBER
    if extracted.exists() and extracted.read_bytes() != source:
        raise RuntimeError("Extracted source differs from the locked archive member")
    stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
    output = ROOT / ".campaign/probes" / ("iso8601-" + stamp)
    project = output / "project"
    source_path = project / "src/main/java" / MEMBER
    source_path.parent.mkdir(parents=True)
    source_path.write_bytes(source)
    driver = project / "src/main/java/probe/Main.java"
    driver.parent.mkdir(parents=True)
    shutil.copyfile(Path(__file__).with_name("Main.java"), driver)
    (project / "pom.xml").write_text('<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>iso8601</artifactId><version>1</version></project>')
    environment = dict(os.environ, TZ="UTC", GOWORK="off")
    home = os.environ.get("JAVA_HOME")
    def java_tool(name):
        result = str(Path(home) / "bin" / name) if home else shutil.which(name)
        if not result:
            raise RuntimeError("JDK21 required: missing " + name)
        return result
    report = {"kind": "focused-integration-probe", "archive": str(archive),
              "archive_sha256": sha(data), "member": MEMBER,
              "source_sha256": sha(source), "driver_sha256": sha(driver.read_bytes()), "commands": [],
              "implementation_before": implementation(), "exact_parity": False, "stable": False}
    def save():
        (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")
    def run(label, command, cwd, timeout=300):
        try:
            result = subprocess.run(command, cwd=cwd, env=environment, capture_output=True, timeout=timeout)
        except subprocess.TimeoutExpired as error:
            (output / (label + ".stdout")).write_bytes(error.stdout or b"")
            (output / (label + ".stderr")).write_bytes(error.stderr or b"")
            report["commands"].append({"label": label, "argv": command, "cwd": str(cwd), "timeout_seconds": timeout, "timed_out": True})
            save()
            raise
        (output / (label + ".stdout")).write_bytes(result.stdout)
        (output / (label + ".stderr")).write_bytes(result.stderr)
        report["commands"].append({"label": label, "argv": command, "cwd": str(cwd), "timeout_seconds": timeout, "returncode": result.returncode})
        save()
        if result.returncode:
            raise RuntimeError(label + " failed: " + str(output / (label + ".stderr")))
        return result
    try:
        report["jdk"] = {}
        for tool in ("java", "javac"):
            version = run(tool + "-version", [java_tool(tool), "-version"], ROOT, 60)
            text = (version.stdout + version.stderr).decode()
            report["jdk"][tool] = {"path": java_tool(tool), "version": text}
            if not re.search(r'(?:version "|javac )21(?:[.\s"+-]|$)', text):
                raise RuntimeError("JDK21 required: " + text)
        classes = output / "classes"
        run("javac", [java_tool("javac"), "--release", "21", "-d", str(classes), str(source_path), str(driver)], ROOT)
        expected = run("java", [java_tool("java"), "-Duser.timezone=UTC", "-cp", str(classes), "probe.Main"], ROOT, 60)
        generated = output / "generated"
        run("transpile", ["go", "run", "./cmd/java2go", "-strict", "-maven", str(project), "-main-class", "probe.Main", "-runtime", str(ROOT), "-module", "example.test/iso8601", "-output", str(generated)], ROOT)
        run("build-all", ["go", "build", "-mod=mod", "./..."], generated)
        binary = output / "app"
        run("build-race", ["go", "build", "-mod=mod", "-race", "-o", str(binary), "./cmd/app"], generated)
        actual = run("go", [str(binary)], ROOT, 60)
        if (actual.stdout, actual.stderr, actual.returncode) != (expected.stdout, expected.stderr, expected.returncode):
            raise RuntimeError("Exact JVM/Go stdout, stderr, or exit mismatch: " + str(output))
        if source_path.read_bytes() != source:
            raise RuntimeError("Upstream source changed during probe")
        report["exact_parity"] = True
    finally:
        report["implementation_after"] = implementation()
        report["stable"] = report["implementation_before"] == report["implementation_after"]
        save()
    if not report["stable"]:
        raise RuntimeError("Implementation changed during probe: " + str(output))

    print(output)

if __name__ == "__main__":
    main()
