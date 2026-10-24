#!/usr/bin/env python3
"""Validate a proposed List slot-clear patch in isolated runtime snapshots only."""
import difflib
import hashlib
import json
import os
from pathlib import Path
import shutil
import statistics
import subprocess
import tempfile
import time

HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]
JDK=Path('/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')

def digest(path): return hashlib.sha256(path.read_bytes()).hexdigest()

def main():
    out=ROOT/'.campaign/performance'/('list-candidate-'+time.strftime('%Y%m%dT%H%M%S'))
    out.mkdir(parents=True)
    sandbox=Path(tempfile.mkdtemp(prefix='java2go-list-candidate-'))
    env=dict(os.environ,GOWORK='off',GOMAXPROCS='2',GOMEMLIMIT='256MiB',GOGC='100',JAVA_HOME=str(JDK),TZ='UTC',LC_ALL='C')
    commands=[]
    def execute(name, command, cwd=ROOT, expected=0, timeout=120):
        result=subprocess.run([str(x) for x in command],cwd=cwd,env=env,capture_output=True,timeout=timeout)
        (out/(name+'.stdout')).write_bytes(result.stdout)
        (out/(name+'.stderr')).write_bytes(result.stderr)
        commands.append(dict(name=name,command=[str(x) for x in command],cwd=str(cwd),exit_code=result.returncode,timeout_seconds=timeout))
        (out/'commands.json').write_text(json.dumps(commands,indent=2)+'\n')
        if result.returncode!=expected: raise RuntimeError(f'{name}: expected {expected}, got {result.returncode}, see {out}')
        return result
    snapshots={}
    source_hash=digest(ROOT/'stdjava/list.go')
    for variant in ['baseline','candidate']:
        target=sandbox/variant
        target.mkdir()
        shutil.copytree(ROOT/'stdjava',target/'stdjava')
        for name in ['go.mod','go.sum']: shutil.copyfile(ROOT/name,target/name)
        snapshots[variant]=target
    original=(snapshots['baseline']/'stdjava/list.go').read_text()
    needle='\tl.elements = append(l.elements[:index], l.elements[index+1:]...)\n'
    replacement='\tlast := len(l.elements) - 1\n\tcopy(l.elements[index:], l.elements[index+1:])\n\tvar zero T\n\tl.elements[last] = zero\n\tl.elements = l.elements[:last]\n'
    if original.count(needle)!=1: raise RuntimeError('List.RemoveAt changed: manually review candidate')
    candidate=original.replace(needle,replacement)
    (snapshots['candidate']/'stdjava/list.go').write_text(candidate)
    patch=''.join(difflib.unified_diff(original.splitlines(True),candidate.splitlines(True),fromfile='a/stdjava/list.go',tofile='b/stdjava/list.go',n=0))
    (HERE/'list-remove-clear.patch').write_text(patch)
    (out/'list-remove-clear.patch').write_text(patch)
    for variant,path in snapshots.items():
        execute(variant+'-stdjava', ['go','test','./stdjava'],cwd=path)
        shutil.copyfile(HERE/'semantics/slot_test.go.txt',path/'stdjava/performance_candidate_slot_test.go')
        execute(variant+'-slots',['go','test','./stdjava','-run','^TestCandidateVacatedSlots$','-count=1'],cwd=path,expected=1 if variant=='baseline' else 0)
    execute('compiler-build',['go','build','-o',out/'java2go','./cmd/java2go'])
    programs={}
    for kind,source,driver in [('memory',HERE/'source/ListRetention.java',HERE/'harness/driver.go.txt'),('semantics',HERE/'semantics/ListSemantics.java',HERE/'semantics/driver.go.txt')]:
        source_dir=sandbox/(kind+'-source'); source_dir.mkdir(); shutil.copyfile(source,source_dir/source.name)
        generated=out/(kind+'-generated')
        execute(kind+'-transpile',[out/'java2go','-strict','-w','-output',generated,source_dir])
        shutil.copyfile(driver,generated/'driver.go')
        for variant,path in snapshots.items():
            (generated/'go.mod').write_text('module perf.listcandidate\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => '+str(path)+'\n')
            binary=out/(kind+'-'+variant)
            execute(kind+'-'+variant+'-build',['go','build','-mod=mod','-o',binary,'.'],cwd=generated)
            programs[(kind,variant)]=binary
    classes=out/'classes'; classes.mkdir()
    execute('javac',[JDK/'bin/javac','--release','21','-d',classes,HERE/'source/ListRetention.java',HERE/'harness/RetentionDriver.java',HERE/'semantics/ListSemantics.java',HERE/'semantics/SemanticsDriver.java'])
    java=[JDK/'bin/java','-XX:+UseSerialGC','-XX:ActiveProcessorCount=2','-Xms32m','-Xmx256m','-cp',classes]
    oracle=execute('semantics-java',java+['SemanticsDriver'],timeout=30).stdout
    for variant in snapshots:
        got=execute('semantics-'+variant,[programs[('semantics',variant)]],timeout=30).stdout
        if got!=oracle: raise RuntimeError(f'{variant} semantic parity failed; see {out}')
    results=[]
    for size in [131072,262144]:
        for trial in range(3):
            outputs={}
            variants=['java','baseline','candidate'] if trial%2==0 else ['candidate','baseline','java']
            for variant in variants:
                cmd=(java+['RetentionDriver'] if variant=='java' else [programs[('memory',variant)]])+['64',str(size),'4']
                result=execute(f'memory-{variant}-{size}-{trial}',cmd,timeout=30)
                outputs[variant]=result.stdout
                metrics=[json.loads(line) for line in result.stderr.decode().splitlines()]
                by={m['phase']:m['delta'] for m in metrics}
                results.append(dict(variant=variant,payload_bytes=size,trial=trial,metrics=metrics,removed_minus_cleared=by['removed-3']-by['cleared']))
            if len(set(outputs.values()))!=1: raise RuntimeError('Memory application semantic stdout mismatch')
    if source_hash!=digest(ROOT/'stdjava/list.go'): raise RuntimeError('Live List changed during audit; repeat on stable snapshot')
    summary=[]
    for size in [131072,262144]:
        for variant in ['java','baseline','candidate']:
            samples=[r['removed_minus_cleared'] for r in results if r['variant']==variant and r['payload_bytes']==size]
            summary.append(dict(variant=variant,payload_bytes=size,batch_payload_bytes=size*64,removed_minus_cleared=samples,median=statistics.median(samples)))
    metadata=dict(scope='isolated List slot-clear candidate; retained heap only; no timing claim or campaign gate',sandbox=str(sandbox),
                  live_list_sha256=source_hash,candidate_list_sha256=digest(snapshots['candidate']/'stdjava/list.go'),
                  shared_memory_java_sha256=digest(HERE/'source/ListRetention.java'),semantic_stdout=oracle.decode(),
                  semantic_parity=True,memory_semantic_parity=True,summary=summary,results=results)
    (out/'results.json').write_text(json.dumps(metadata,indent=2)+'\n')
    print(json.dumps(dict(output=str(out),sandbox=str(sandbox),semantic_parity=True,summary=summary),indent=2))

if __name__=='__main__': main()
