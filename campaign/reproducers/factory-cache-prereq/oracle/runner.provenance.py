#!/usr/bin/env python3
"""Deferred JDK21 oracle for the queued generic factory/cache prerequisite."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time


ROOT = Path('/private/tmp/java2go-factory-cache-prereq')
JDK = Path('/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')
JAR = Path('/Users/suhaib/.codex/worktrees/adversarial-java/java2go/.campaign/cache/commons-lang3-3.20.0.jar')
SOURCE_SHA = '354d17005f9ef9b42f82ed6e26ab1b6ff6e87699877940d4b5bc894f86a9e73f'
JAR_SHA = '69e5c9fa35da7a51a5fd2099dfe56a2d8d32cf233e2f6d770e796146440263f4'
SEEDS = (17, 41, 97)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def source_sha():
    digest = hashlib.sha256()
    for path in sorted(p for p in ROOT.rglob('*') if p.is_file()):
        digest.update(str(path.relative_to(ROOT)).encode())
        digest.update(b'\0')
        digest.update(path.read_bytes())
        digest.update(b'\0')
    return digest.hexdigest()


def execute(command, cwd, timeout, prefix, env):
    started = time.monotonic()
    try:
        proc = subprocess.run(command, cwd=cwd, env=env, capture_output=True,
                              timeout=timeout)
        status, stdout, stderr, timed_out = proc.returncode, proc.stdout, proc.stderr, False
    except subprocess.TimeoutExpired as error:
        status, stdout, stderr, timed_out = None, error.stdout or b'', error.stderr or b'', True
    stdout_path = Path(str(prefix) + '.stdout')
    stderr_path = Path(str(prefix) + '.stderr')
    stdout_path.write_bytes(stdout)
    stderr_path.write_bytes(stderr)
    return {'command': [str(part) for part in command], 'cwd': str(cwd),
            'timeout_seconds': timeout, 'seconds': time.monotonic() - started,
            'exit': status, 'timed_out': timed_out,
            'stdout_path': str(stdout_path), 'stdout_sha256': sha(stdout),
            'stdout_bytes': len(stdout), 'stderr_path': str(stderr_path),
            'stderr_sha256': sha(stderr), 'stderr_bytes': len(stderr)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--jar', type=Path, default=JAR)
    args = parser.parse_args()
    out = args.out.resolve()
    if out.exists() or out == ROOT or out.is_relative_to(ROOT):
        parser.error('--out must be a new directory outside the source fixture')
    before = source_sha()
    if before != SOURCE_SHA:
        parser.error(f'source fingerprint changed: {before}')
    jar = args.jar.resolve()
    if sha(jar.read_bytes()) != JAR_SHA:
        parser.error('pinned Commons Lang jar SHA-256 mismatch')
    out.mkdir(parents=True)
    classes = out / 'classes'
    classes.mkdir()
    env = os.environ.copy()
    env['JAVA_HOME'] = str(JDK)
    env['PATH'] = str(JDK / 'bin') + os.pathsep + env.get('PATH', '')
    report = {'fixture': str(ROOT), 'source_sha256_before': before,
              'jar': str(jar), 'jar_sha256': JAR_SHA, 'seeds': list(SEEDS),
              'repeats': 3, 'runs': []}
    sources = sorted(str(p) for p in ROOT.rglob('*.java'))
    command = [str(JDK / 'bin/javac'), '--release', '21', '-encoding', 'UTF-8',
               '-cp', str(jar), '-d', str(classes), *sources]
    report['javac'] = execute(command, ROOT, 300, out / 'javac', env)
    (out / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
    if report['javac']['exit'] != 0 or report['javac']['timed_out']:
        report['status'] = 'invalid-java-or-javac-failed'
        report['source_sha256_after'] = source_sha()
        (out / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
        return 1
    for seed in SEEDS:
        arg = (ROOT / f'inputs/seed-{seed}.txt').read_text().strip()
        for repeat in (1, 2, 3):
            command = [str(JDK / 'bin/java'), '-cp', str(classes) + os.pathsep + str(jar),
                       'prereq.factory.app.Main', arg]
            prefix = out / f'jdk.seed-{seed}.repeat-{repeat}'
            report['runs'].append({'seed': seed, 'repeat': repeat,
                                   **execute(command, ROOT, 60, prefix, env)})
            (out / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
    stable = True
    for seed in SEEDS:
        runs = [run for run in report['runs'] if run['seed'] == seed]
        signature = {(r['exit'], r['timed_out'], r['stdout_sha256'], r['stderr_sha256'])
                     for r in runs}
        stable = stable and len(signature) == 1 and all(r['exit'] == 0 for r in runs)
        if len(signature) == 1 and all(r['exit'] == 0 for r in runs):
            for stream in ('stdout', 'stderr'):
                (out / f'expected.seed-{seed}.{stream}').write_bytes(
                    Path(runs[0][f'{stream}_path']).read_bytes())
    report['source_sha256_after'] = source_sha()
    report['status'] = ('passed' if stable and before == report['source_sha256_after']
                        else 'unstable-or-failed')
    (out / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
    return 0 if report['status'] == 'passed' else 1


if __name__ == '__main__':
    sys.exit(main())
