#!/usr/bin/env python3
"""Matched retained-heap diagnostic; no throughput result or campaign gate.
Run only in a performance window coordinated with the campaign root.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import statistics
import subprocess
import time

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
JDK = Path('/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--trials', type=int, default=3)
    parser.add_argument('--rounds', type=int, default=4)
    parser.add_argument('--count', type=int, default=64)
    parser.add_argument('--sizes', type=int, nargs='+', default=[131072, 262144])
    args = parser.parse_args()
    if min(args.trials, args.rounds, args.count, *args.sizes) < 1:
        parser.error('all workload dimensions must be positive')
    out = ROOT / '.campaign/performance' / ('list-retention-' + time.strftime('%Y%m%dT%H%M%S'))
    out.mkdir(parents=True)
    records = []
    env = dict(os.environ, GOWORK='off', GOMAXPROCS='2', GOMEMLIMIT='256MiB', GOGC='100', JAVA_HOME=str(JDK), TZ='UTC', LC_ALL='C')

    def execute(name, command, cwd=ROOT, timeout=120):
        started = time.monotonic()
        result = subprocess.run([str(x) for x in command], cwd=cwd, env=env, capture_output=True, timeout=timeout)
        record = dict(name=name, command=[str(x) for x in command], cwd=str(cwd), exit_code=result.returncode,
                      elapsed_seconds=time.monotonic()-started, timeout_seconds=timeout)
        (out / (name+'.stdout')).write_bytes(result.stdout)
        (out / (name+'.stderr')).write_bytes(result.stderr)
        records.append(record)
        (out/'commands.json').write_text(json.dumps(records, indent=2)+'\n')
        if result.returncode:
            raise RuntimeError(f'{name} failed; see {out}')
        return result

    before = sha(ROOT/'stdjava/list.go')
    execute('go-version', ['go', 'version'])
    execute('java-version', [JDK/'bin/java', '-version'])
    execute('compiler-build', ['go', 'build', '-o', out/'java2go', './cmd/java2go'])
    generated = out/'generated'
    execute('transpile', [out/'java2go', '-strict', '-w', '-output', generated, HERE/'source'])
    shutil.copyfile(HERE/'harness/driver.go.txt', generated/'driver.go')
    (generated/'go.mod').write_text('module perf.listretention\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => '+str(ROOT)+'\n')
    execute('go-build', ['go', 'build', '-mod=mod', '-o', out/'list-go', '.'], cwd=generated)
    classes=out/'classes'
    classes.mkdir()
    execute('javac', [JDK/'bin/javac', '--release', '21', '-d', classes, HERE/'source/ListRetention.java', HERE/'harness/RetentionDriver.java'])
    results=[]
    for size in args.sizes:
        for trial in range(args.trials):
            outputs={}
            order=['java','go'] if trial%2==0 else ['go','java']
            for runtime in order:
                prefix=f'{runtime}-bytes{size}-trial{trial}'
                dimensions=[str(args.count), str(size), str(args.rounds)]
                command=([JDK/'bin/java', '-XX:+UseSerialGC', '-XX:ActiveProcessorCount=2', '-Xms32m', '-Xmx256m', '-cp', classes, 'RetentionDriver'] if runtime=='java' else [out/'list-go'])+dimensions
                result=execute(prefix, command, timeout=30)
                outputs[runtime]=result.stdout
                metrics=[json.loads(line) for line in result.stderr.decode().splitlines()]
                results.append(dict(runtime=runtime, trial=trial, count=args.count, payload_bytes=size, metrics=metrics))
            if outputs['java']!=outputs['go']:
                raise RuntimeError(f'Semantic stdout mismatch for {size}, trial {trial}: {out}')
    if before!=sha(ROOT/'stdjava/list.go'):
        raise RuntimeError('List implementation changed while building/running; rerun on a stable snapshot')
    summary=[]
    for size in args.sizes:
        for runtime in ['java','go']:
            selected=[r for r in results if r['runtime']==runtime and r['payload_bytes']==size]
            final=[next(m['delta'] for m in r['metrics'] if m['phase']==f'removed-{args.rounds-1}') for r in selected]
            clear=[next(m['delta'] for m in r['metrics'] if m['phase']=='cleared') for r in selected]
            summary.append(dict(runtime=runtime, payload_bytes=size, batch_payload_bytes=size*args.count,
                                final_removed_delta_bytes=final, median_final_removed_delta_bytes=statistics.median(final),
                                clear_delta_bytes=clear, median_clear_delta_bytes=statistics.median(clear)))
    metadata=dict(scope='Retained heap, not speed; identical transpiled application logic with runtime-specific GC instrumentation',
                  semantic_stdout_parity=True, workload=vars(args), env={k:env[k] for k in ['GOWORK','GOMAXPROCS','GOMEMLIMIT','GOGC','JAVA_HOME','TZ','LC_ALL']},
                  source_sha256=sha(HERE/'source/ListRetention.java'), generated_sha256=sha(generated/'ListRetention.go'),
                  stdjava_list_sha256=before, summary=summary, results=results)
    (out/'results.json').write_text(json.dumps(metadata,indent=2)+'\n')
    print(json.dumps(dict(output=str(out), parity=True, summary=summary),indent=2))

if __name__=='__main__':
    main()
