#!/usr/bin/env python3
"""Bounded nine-process JDK oracle; writes outputs outside its frozen source tree."""

import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile


ROOT = Path(__file__).resolve().parent
JDK = Path('/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')
SOURCES = [
    'src/probe/forward/state/Trace.java',
    'src/probe/forward/source/Token.java',
    'src/probe/forward/source/RawCursor.java',
    'src/probe/forward/source/Forwarder.java',
    'src/probe/forward/app/Main.java',
]


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def check_inputs() -> dict[str, str]:
    expected = {}
    for line in (ROOT / 'SOURCE_HASHES.sha256').read_text().splitlines():
        digest, name = line.split('  ', 1)
        actual = sha(ROOT / name)
        if actual != digest:
            raise RuntimeError(f'frozen input hash mismatch: {name}: {actual}')
        expected[name] = digest
    return expected


def run_bounded(command: list[str], seconds: int, stdout: Path, stderr: Path) -> dict:
    with stdout.open('wb') as out, stderr.open('wb') as err:
        child = subprocess.Popen(command, cwd=ROOT, stdout=out, stderr=err, start_new_session=True)
        try:
            exit_code = child.wait(timeout=seconds)
            timed_out = False
        except subprocess.TimeoutExpired:
            timed_out = True
            err.write(f'oracle subprocess timed out after {seconds}s\n'.encode())
            err.flush()
            try:
                os.killpg(child.pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
            try:
                child.wait(timeout=2)
            except subprocess.TimeoutExpired:
                pass
            try:
                os.killpg(child.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            child.wait()
            exit_code = 124
    return {
        'command': command,
        'timeout_seconds': seconds,
        'timed_out': timed_out,
        'exit': exit_code,
        'stdout_path': stdout.name,
        'stderr_path': stderr.name,
        'stdout_sha256': sha(stdout),
        'stderr_sha256': sha(stderr),
        'stdout_bytes': stdout.stat().st_size,
        'stderr_bytes': stderr.stat().st_size,
    }


def main() -> int:
    frozen = check_inputs()
    out = Path(tempfile.mkdtemp(prefix='generic-forwarder-cast-oracle.', dir='/private/tmp'))
    (out / 'classes').mkdir()
    status = {'output': str(out), 'source_hashes': frozen, 'source_manifest_sha256': sha(ROOT / 'SOURCE_HASHES.sha256')}
    try:
        compile_command = [str(JDK / 'bin/javac'), '--release', '21', '-encoding', 'UTF-8',
                           '-d', str(out / 'classes'), *SOURCES]
        status['compile'] = run_bounded(compile_command, 300, out / 'javac.stdout', out / 'javac.stderr')
        (out / 'status.json').write_text(json.dumps(status, indent=2, sort_keys=True) + '\n')
        if status['compile']['exit'] != 0:
            raise RuntimeError(f'JDK compile failed: exit {status["compile"]["exit"]}')

        observations = []
        canonical = {}
        for seed in (17, 41, 97):
            seed_runs = []
            for repeat in (1, 2, 3):
                prefix = f'seed-{seed}.repeat-{repeat}'
                command = [str(JDK / 'bin/java'), '-cp', str(out / 'classes'),
                           'probe.forward.app.Main', str(seed)]
                result = run_bounded(command, 60, out / f'{prefix}.stdout', out / f'{prefix}.stderr')
                result.update({'seed': seed, 'repeat': repeat})
                observations.append(result)
                seed_runs.append(result)
                status['observations'] = observations
                (out / 'status.json').write_text(json.dumps(status, indent=2, sort_keys=True) + '\n')
                if result['exit'] != 0:
                    raise RuntimeError(f'JVM seed {seed} repeat {repeat} failed: exit {result["exit"]}')
            first_stdout = (out / seed_runs[0]['stdout_path']).read_bytes()
            first_stderr = (out / seed_runs[0]['stderr_path']).read_bytes()
            for item in seed_runs[1:]:
                if (out / item['stdout_path']).read_bytes() != first_stdout or (out / item['stderr_path']).read_bytes() != first_stderr:
                    raise RuntimeError(f'JVM seed {seed} is nondeterministic')
            stdout_name = f'expected.seed-{seed}.stdout'
            stderr_name = f'expected.seed-{seed}.stderr'
            shutil.copyfile(out / seed_runs[0]['stdout_path'], out / stdout_name)
            shutil.copyfile(out / seed_runs[0]['stderr_path'], out / stderr_name)
            canonical[str(seed)] = {
                'stdout': {'path': stdout_name, 'sha256': sha(out / stdout_name), 'bytes': len(first_stdout)},
                'stderr': {'path': stderr_name, 'sha256': sha(out / stderr_name), 'bytes': len(first_stderr)},
            }
        if check_inputs() != frozen:
            raise RuntimeError('frozen input hashes changed during oracle run')
        status['input_hashes_match_before_after'] = True
        status['canonical_streams'] = canonical
        (out / 'status.json').write_text(json.dumps(status, indent=2, sort_keys=True) + '\n')
        oracle = {
            'kind': 'supplemental-generic-forwarder-cast-jdk21-oracle',
            'jdk_home': str(JDK),
            'compile': status['compile'],
            'source_manifest_sha256': status['source_manifest_sha256'],
            'seeds': canonical,
            'runs': [{'seed': r['seed'], 'repeat': r['repeat'], 'exit': r['exit'],
                      'stdout_sha256': r['stdout_sha256'], 'stderr_sha256': r['stderr_sha256']}
                     for r in observations],
        }
        (out / 'oracle.json').write_text(json.dumps(oracle, indent=2, sort_keys=True) + '\n')
        (out / 'provenance.json').write_text(json.dumps(status, indent=2, sort_keys=True) + '\n')
        print(f'oracle_root={out}')
        return 0
    except Exception as error:
        status['failure'] = str(error)
        (out / 'status.json').write_text(json.dumps(status, indent=2, sort_keys=True) + '\n')
        print(f'oracle_failed={error} artifacts={out}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
