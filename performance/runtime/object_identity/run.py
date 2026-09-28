#!/usr/bin/env python3
"""Frozen-source identity allocation/reachability diagnostic; no elapsed-time claims."""
import hashlib, json, os, shutil, subprocess, tempfile, time
from pathlib import Path
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]
JDK=Path('/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')
REV='8cb6cd529c124802d9d004ef43d41dffe16f1085'
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
 out=ROOT/'.campaign/performance'/('object-identity-'+time.strftime('%Y%m%dT%H%M%S'));out.mkdir(parents=True)
 sandbox=Path(tempfile.mkdtemp(prefix='java2go-object-identity-'))
 baseline=sandbox/'baseline';baseline.mkdir()
 env=dict(os.environ,GOWORK='off',GOMAXPROCS='2',GOMEMLIMIT='256MiB',GOGC='100',JAVA_HOME=str(JDK),TZ='UTC',LC_ALL='C')
 commands=[]
 def run(name,cmd,cwd=ROOT,timeout=120):
  cmd=[str(x) for x in cmd]
  r=subprocess.run(cmd,cwd=cwd,env=env,capture_output=True,timeout=timeout)
  (out/(name+'.stdout')).write_bytes(r.stdout);(out/(name+'.stderr')).write_bytes(r.stderr)
  commands.append(dict(name=name,command=cmd,cwd=str(cwd),exit_code=r.returncode,timeout_seconds=timeout))
  (out/'commands.json').write_text(json.dumps(commands,indent=2)+'\n')
  if r.returncode:raise RuntimeError(f'{name} failed: {out}')
  return r
 run('go-version',['go','version'])
 run('java-version',[JDK/'bin/java','-version'])
 run('host',['uname','-sm'])
 archive=run('archive',['git','archive','--format=tar',REV]).stdout
 (out/'snapshot.tar').write_bytes(archive)
 run('extract',['tar','-xf',out/'snapshot.tar','-C',baseline])
 counted=sandbox/'instrumented';counted.mkdir()
 shutil.copytree(baseline/'stdjava',counted/'stdjava')
 for name in ['go.mod','go.sum']:shutil.copyfile(baseline/name,counted/name)
 shutil.copyfile(HERE/'harness/diagnostics.go.txt',counted/'stdjava/performance_identity_diagnostics.go')
 run('compiler-build',['go','build','-o',out/'java2go','./cmd/java2go'],cwd=baseline)
 generated=out/'generated';run('transpile',[out/'java2go','-strict','-w','-output',generated,HERE/'source'])
 generated_hash=sha(generated/'ObjectIdentityProbe.go')
 for variant,runtime,driver in [('baseline',baseline,'driver.go.txt'),('instrumented',counted,'counted_driver.go.txt')]:
  dest=sandbox/('app-'+variant);shutil.copytree(generated,dest)
  (dest/'go.mod').write_text('module perf.objectidentity\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => '+str(runtime)+'\n')
  shutil.copyfile(HERE/'harness'/driver,dest/'driver.go')
  run(variant+'-build',['go','build','-mod=mod','-o',out/variant,'.'],cwd=dest)
  assert sha(dest/'ObjectIdentityProbe.go')==generated_hash
 classes=out/'classes';classes.mkdir()
 run('javac',[JDK/'bin/javac','--release','21','-d',classes,HERE/'source/ObjectIdentityProbe.java',HERE/'harness/IdentityDriver.java'])
 java=[JDK/'bin/java','-XX:+UseSerialGC','-XX:ActiveProcessorCount=2','-Xms32m','-Xmx256m','-cp',classes,'IdentityDriver']
 results=[]
 for seed in [17,97]:
  for mode in [0,1,2]:
   outputs={}
   for variant in (['java','baseline','instrumented'] if seed==17 else ['instrumented','baseline','java']):
    r=run(f'{variant}-seed{seed}-mode{mode}',(java if variant=='java' else [out/variant])+[seed,mode],timeout=30)
    outputs[variant]=r.stdout
    results.append(dict(variant=variant,seed=seed,mode=mode,stdout_sha256=hashlib.sha256(r.stdout).hexdigest(),metrics=[json.loads(x) for x in r.stderr.decode().splitlines()]))
   if len(set(outputs.values()))!=1:raise RuntimeError(f'parity mismatch seed={seed} mode={mode}: {out}')
 dest=sandbox/'app-instrumented';shutil.copyfile(HERE/'harness/allocation_test.go.txt',dest/'identity_allocation_test.go')
 for trial in range(3):run('allocation-'+str(trial),['go','test','-mod=mod','-run','^TestObjectIdentityAllocations$','-v','-count=1','.'],cwd=dest,timeout=30)
 run('escape-analysis',['go','build','-mod=mod','-gcflags=perf.objectidentity=-m=2','.'],cwd=dest)
 fingerprints={}
 for label,tree in [('baseline',baseline/'stdjava'),('instrumented',counted/'stdjava')]:
  files={str(p.relative_to(tree)):sha(p) for p in sorted(tree.rglob('*.go'))}
  fingerprints[label]=dict(files=files,manifest_sha256=hashlib.sha256(json.dumps(files,sort_keys=True).encode()).hexdigest())
 metadata=dict(scope='Allocation and post-GC reachability only; no speed claims',revision=REV,sandbox=str(sandbox),archive_sha256=sha(out/'snapshot.tar'),source_sha256=sha(HERE/'source/ObjectIdentityProbe.java'),generated_sha256=generated_hash,runtime_fingerprints=fingerprints,harness_fingerprints={p.name:sha(p) for p in sorted((HERE/'harness').iterdir())},parity=True,results=results)
 (out/'results.json').write_text(json.dumps(metadata,indent=2)+'\n')
 print(json.dumps(dict(output=str(out),sandbox=str(sandbox),parity=True,results=results),indent=2))
if __name__=='__main__':main()
