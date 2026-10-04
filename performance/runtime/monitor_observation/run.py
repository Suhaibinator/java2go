#!/usr/bin/env python3
"""Isolated monitor observation audit. Coordinate short probes; no timing claims."""
import difflib, hashlib, json, os, shutil, subprocess, tempfile, time
from pathlib import Path
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]
JDK=Path('/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def main():
 out=ROOT/'.campaign/performance'/('monitor-observation-'+time.strftime('%Y%m%dT%H%M%S'));out.mkdir(parents=True)
 sandbox=Path(tempfile.mkdtemp(prefix='java2go-monitor-observation-'))
 env=dict(os.environ,GOWORK='off',GOMAXPROCS='2',GOMEMLIMIT='256MiB',GOGC='100',JAVA_HOME=str(JDK),TZ='UTC',LC_ALL='C')
 records=[]
 def run(name,cmd,cwd=ROOT,expected=0,timeout=120):
  r=subprocess.run([str(x) for x in cmd],cwd=cwd,env=env,capture_output=True,timeout=timeout)
  (out/(name+'.stdout')).write_bytes(r.stdout);(out/(name+'.stderr')).write_bytes(r.stderr)
  records.append(dict(name=name,command=[str(x) for x in cmd],cwd=str(cwd),exit_code=r.returncode,timeout_seconds=timeout))
  (out/'commands.json').write_text(json.dumps(records,indent=2)+'\n')
  if r.returncode!=expected:raise RuntimeError(f'{name} unexpected exit {r.returncode}: {out}')
  return r
 snapshots={}
 fingerprints={p.name:sha(p) for p in [ROOT/'stdjava/monitor_observation.go',ROOT/'stdjava/concurrent.go',ROOT/'stdjava/character_io.go']}
 for variant in ['baseline','candidate']:
  dest=sandbox/variant;dest.mkdir();shutil.copytree(ROOT/'stdjava',dest/'stdjava')
  for name in ['go.mod','go.sum']:shutil.copyfile(ROOT/name,dest/name)
  snapshots[variant]=dest
 original=(snapshots['baseline']/'stdjava/monitor_observation.go').read_text()
 needle='\tmonitor := monitorRecord(value)\n'
 if original.count(needle)!=1:raise RuntimeError('Observation implementation changed; manual review required')
 proposed=original.replace(needle,'\tidentity := monitorIdentityFor(value)\n\tmonitorsMu.Lock()\n\tmonitor := monitors[identity]\n\tmonitorsMu.Unlock()\n\tif monitor == nil {\n\t\treturn false\n\t}\n')
 (snapshots['candidate']/'stdjava/monitor_observation.go').write_text(proposed)
 patch=''.join(difflib.unified_diff(original.splitlines(True),proposed.splitlines(True),fromfile='a/stdjava/monitor_observation.go',tofile='b/stdjava/monitor_observation.go',n=0))
 (HERE/'lookup-only.patch').write_text(patch);(out/'lookup-only.patch').write_text(patch)
 for variant,path in snapshots.items():
  run(variant+'-runtime-suite',['go','test','./stdjava'],cwd=path)
  shutil.copyfile(HERE/'harness/registry_test.go.txt',path/'stdjava/performance_observation_test.go')
  run(variant+'-diagnostics',['go','test','./stdjava','-run','^TestObservation(Registry|Identity)Diagnostic$','-v','-count=1'],cwd=path)
  run(variant+'-no-insert',['go','test','./stdjava','-run','^TestObservationNoInsertCandidate$','-count=1'],cwd=path,expected=1 if variant=='baseline' else 0)
 run('compiler-build',['go','build','-o',out/'java2go','./cmd/java2go'])
 generated=out/'generated';run('transpile',[out/'java2go','-strict','-w','-output',generated,HERE/'source'])
 shutil.copyfile(HERE/'harness/driver.go.txt',generated/'driver.go')
 for variant,path in snapshots.items():
  (generated/'go.mod').write_text('module perf.monitorobservation\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => '+str(path)+'\n')
  run(variant+'-build',['go','build','-mod=mod','-o',out/variant,'.'],cwd=generated)
 classes=out/'classes';classes.mkdir();run('javac',[JDK/'bin/javac','--release','21','-d',classes,HERE/'source/MonitorObservation.java',HERE/'harness/ObservationDriver.java'])
 java=[JDK/'bin/java','-XX:+UseSerialGC','-XX:ActiveProcessorCount=2','-Xms32m','-Xmx256m','-cp',classes,'ObservationDriver']
 results=[]
 for mode in [0,1]:
  for trial in range(3):
   outputs={}
   for variant in (['java','baseline','candidate'] if trial%2==0 else ['candidate','baseline','java']):
    cmd=(java if variant=='java' else [out/variant])+[str(mode)]
    r=run(f'{variant}-mode{mode}-trial{trial}',cmd,timeout=30);outputs[variant]=r.stdout
    results.append(dict(variant=variant,mode=mode,trial=trial,metrics=[json.loads(x) for x in r.stderr.decode().splitlines()]))
   if len(set(outputs.values()))!=1:raise RuntimeError(f'Generated parity mismatch mode {mode}: {out}')
 metadata=dict(scope='Post-GC retention/registry allocation only; no timing claims',sandbox=str(sandbox),fingerprints=fingerprints,source_sha256=sha(HERE/'source/MonitorObservation.java'),parity=True,results=results)
 (out/'results.json').write_text(json.dumps(metadata,indent=2)+'\n')
 summary=[dict(variant=x['variant'],mode=x['mode'],deltas=[m['delta'] for m in x['metrics']]) for x in results if x['trial']==0]
 print(json.dumps(dict(output=str(out),sandbox=str(sandbox),parity=True,first_trial=summary),indent=2))
if __name__=='__main__':main()
