#!/usr/bin/env python3
"""Parity-gated alternating process/steady measurements; no shell commands.
Config: mode (startup|steady), java/go argv lists, oracle file, forks,
warmups, samples, java_env/go_env (overrides), resources (description).
In steady mode argv must run the repeat drivers with matching counts.
Writes partial raw evidence after every validated fork; fails closed on mismatch.
"""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import random
import statistics
import subprocess
import time

p = argparse.ArgumentParser()
p.add_argument('config', type=Path)
p.add_argument('output', type=Path)
a = p.parse_args()
c = json.loads(a.config.read_text())
mode = c['mode']
if mode not in ('startup', 'steady'):
    raise SystemExit('mode must be startup or steady')
forks = int(c.get('forks', 8))
warmups, samples = int(c.get('warmups', 3)), int(c.get('samples', 5))
if forks < 2 or warmups < 1 or samples < 1:
    raise SystemExit('require forks >= 2, warmups >= 1, samples >= 1')
oracle = Path(c['oracle']).read_bytes()
if not oracle:
    raise SystemExit('requires a nonempty exact stdout oracle and empty application stderr')
result = {'config': c, 'oracle_sha256': hashlib.sha256(oracle).hexdigest(), 'runs': [],
          'note': 'steady timings include main output; startup is complete process latency'}
def save():
    a.output.write_text(json.dumps(result, indent=2) + '\n')
for pair in range(forks):
    for runtime in (('java', 'go') if pair % 2 == 0 else ('go', 'java')):
        env = os.environ.copy()
        for key in ('JAVA_TOOL_OPTIONS', '_JAVA_OPTIONS', 'JDK_JAVA_OPTIONS', 'GOGC', 'GOMEMLIMIT', 'GOMAXPROCS', 'GODEBUG'):
            env.pop(key, None)
        env.update({'TZ': 'UTC', 'LANG': 'C', 'LC_ALL': 'C'})
        env.update(c.get(runtime + '_env', {}))
        start = time.perf_counter_ns()
        proc = subprocess.Popen(c[runtime], cwd=c.get(runtime + '_cwd'), env=env,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        peak_sampled_rss = 0
        deadline = time.monotonic() + c.get('timeout_seconds', 900)
        while True:
            try:
                stdout, stderr = proc.communicate(timeout=0.25)
                break
            except subprocess.TimeoutExpired:
                rss = subprocess.run(['ps', '-o', 'rss=', '-p', str(proc.pid)], capture_output=True, text=True)
                if rss.returncode == 0 and rss.stdout.strip():
                    peak_sampled_rss = max(peak_sampled_rss, int(rss.stdout.strip()) * 1024)
                if time.monotonic() > deadline or peak_sampled_rss > c.get('rss_abort_bytes', float('inf')):
                    proc.kill()
                    proc.communicate()
                    raise SystemExit(f'{runtime}: timeout or sampled RSS ceiling exceeded; no claim valid')
        elapsed = time.perf_counter_ns() - start
        count = 1 if mode == 'startup' else warmups + samples
        if proc.returncode or stdout != oracle * count:
            raise SystemExit(f'{runtime} pair {pair}: exit/output parity failed; no performance claim valid')
        if mode == 'startup':
            if stderr:
                raise SystemExit(f'{runtime}: unexpected stderr')
            times, warm = [elapsed], []
        else:
            lines = stderr.decode('ascii').splitlines()
            expected = list(range(-warmups, samples))
            frames = [line.split('\t') for line in lines]
            if len(frames) != len(expected) or any(len(f) != 3 or f[0] != 'PERF' or int(f[1]) != index or int(f[2]) <= 0 for f, index in zip(frames, expected)):
                raise SystemExit(f'{runtime}: malformed timing frames/application stderr')
            warm = [int(f[2]) for f in frames[:warmups]]
            times = [int(f[2]) for f in frames[warmups:]]
        if c.get('artifact_directory'):
            artifact = Path(c['artifact_directory'])
            artifact.mkdir(parents=True, exist_ok=True)
            (artifact / f'{pair}-{runtime}.stdout').write_bytes(stdout)
            (artifact / f'{pair}-{runtime}.stderr').write_bytes(stderr)
        result['runs'].append({'peak_sampled_rss_bytes': peak_sampled_rss, 'pair': pair, 'runtime': runtime, 'process_ns': elapsed,
                               'warmup_ns': warm, 'sample_ns': times,
                               'median_ns': statistics.median(times)})
        save()
ratios = []
for pair in range(forks):
    times = {r['runtime']: r['median_ns'] for r in result['runs'] if r['pair'] == pair}
    ratios.append(times['go'] / times['java'])
logs = [math.log(r) for r in ratios]
rng = random.Random(72821)
boot = sorted(math.exp(statistics.mean(rng.choices(logs, k=len(logs)))) for _ in range(10000))
result['summary'] = {'go_over_java_geomean': math.exp(statistics.mean(logs)),
                     'paired_fork_ratios': ratios,
                     'paired_fork_bootstrap_95pct': [boot[249], boot[9749]],
                     'caution': 'Exploratory uncertainty; inspect warmup/trends and host contention before claiming superiority.'}
save()
print(json.dumps(result['summary'], indent=2))
