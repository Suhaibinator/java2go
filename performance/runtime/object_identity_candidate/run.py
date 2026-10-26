#!/usr/bin/env python3
"""Isolated BFS queue candidate: TDD, source parity, allocation, and reachability."""
import difflib, hashlib, json, os, shutil, subprocess, tempfile, time
from pathlib import Path
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]
PRIOR=HERE.parent/'object_identity'
REV='8cb6cd529c124802d9d004ef43d41dffe16f1085'
JDK=Path('/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
 out=ROOT/'.campaign/performance'/('object-identity-candidate-'+time.strftime('%Y%m%dT%H%M%S'));out.mkdir(parents=True)
 sandbox=Path(tempfile.mkdtemp(prefix='java2go-identity-queue-'));baseline=sandbox/'baseline';baseline.mkdir()
 env=dict(os.environ,GOWORK='off',GOMAXPROCS='2',GOMEMLIMIT='256MiB',GOGC='100',JAVA_HOME=str(JDK),TZ='UTC',LC_ALL='C')
 commands=[]
 def run(name,cmd,cwd=ROOT,expected=0,timeout=120):
  cmd=[str(x) for x in cmd];r=subprocess.run(cmd,cwd=cwd,env=env,capture_output=True,timeout=timeout)
  (out/(name+'.stdout')).write_bytes(r.stdout);(out/(name+'.stderr')).write_bytes(r.stderr)
  commands.append(dict(name=name,command=cmd,cwd=str(cwd),exit_code=r.returncode,timeout_seconds=timeout))
  (out/'commands.json').write_text(json.dumps(commands,indent=2)+'\n')
  if r.returncode!=expected:raise RuntimeError(f'{name}: exit {r.returncode}, expected {expected}: {out}')
  return r
 run('go-version',['go','version']);run('java-version',[JDK/'bin/java','-version']);run('host',['uname','-sm'])
 (out/'snapshot.tar').write_bytes(run('archive',['git','archive','--format=tar',REV]).stdout)
 run('extract',['tar','-xf',out/'snapshot.tar','-C',baseline])
 candidate=sandbox/'candidate';candidate.mkdir();shutil.copytree(baseline/'stdjava',candidate/'stdjava')
 for name in ['go.mod','go.sum']:shutil.copyfile(baseline/name,candidate/name)
 variants={'baseline':baseline,'candidate':candidate}
 for variant,runtime in variants.items():shutil.copyfile(HERE/'harness/regression_test.go.txt',runtime/'stdjava/performance_queue_test.go')
 run('baseline-semantic-regressions',['go','test','./stdjava','-run','^TestCandidate','-skip','^TestCandidateSmallTraversalAllocation$','-count=1'],cwd=baseline)
 run('baseline-allocation-red',['go','test','./stdjava','-run','^TestCandidateSmallTraversalAllocation$','-count=1'],cwd=baseline,expected=1)
 file=candidate/'stdjava/reference_arrays.go';original=file.read_text()
 first='queue := []TypeID{actual}\n\tfor len(queue) > 0 {\n\t\tcurrent := queue[0]\n\t\tqueue = queue[1:]'
 second='queue := []TypeID{actual}\n\tresult := make([]TypeID, 0, 4)\n\tfor len(queue) > 0 {\n\t\tcurrent := queue[0]\n\t\tqueue = queue[1:]'
 queue='var initialQueue [8]TypeID\n\tqueue := initialQueue[:1]\n\tqueue[0] = actual\n'
 loop='\tfor head := 0; head < len(queue); head++ {\n\t\tcurrent := queue[head]'
 assert original.count(first)==1 and original.count(second)==1
 changed=original.replace(first,queue+loop).replace(second,queue+'\tresult := make([]TypeID, 0, 4)\n'+loop)
 file.write_text(changed)
 patch=''.join(difflib.unified_diff(original.splitlines(True),changed.splitlines(True),fromfile='a/stdjava/reference_arrays.go',tofile='b/stdjava/reference_arrays.go',n=0))
 (HERE/'queue.patch').write_text(patch);(out/'queue.patch').write_text(patch)
 check=sandbox/'patch-check';(check/'stdjava').mkdir(parents=True)
 shutil.copyfile(baseline/'stdjava/reference_arrays.go',check/'stdjava/reference_arrays.go')
 run('patch-check',['/usr/bin/patch','-p1','-i',out/'queue.patch'],cwd=check)
 assert (check/'stdjava/reference_arrays.go').read_bytes()==file.read_bytes()
 run('candidate-allocation-green',['go','test','./stdjava','-run','^TestCandidateSmallTraversalAllocation$','-count=1'],cwd=candidate)
 for variant,runtime in variants.items():
  skip=['-skip','^TestCandidateSmallTraversalAllocation$'] if variant=='baseline' else []
  run(variant+'-runtime-suite',['go','test','./stdjava','-count=1']+skip,cwd=runtime)
  run(variant+'-runtime-race',['go','test','-race','./stdjava','-count=1','-skip','^TestCandidateSmallTraversalAllocation$'],cwd=runtime,timeout=240)
  shutil.copyfile(PRIOR/'harness/diagnostics.go.txt',runtime/'stdjava/performance_identity_diagnostics.go')
 run('compiler-build',['go','build','-o',out/'java2go','./cmd/java2go'],cwd=baseline)
 generated=out/'generated';run('transpile',[out/'java2go','-strict','-w','-output',generated,PRIOR/'source'])
 semantics=out/'semantics';run('transpile-semantics',[out/'java2go','-strict','-w','-output',semantics,HERE/'source'])
 for variant,runtime in variants.items():
  for label,source,driver in [('app',generated,PRIOR/'harness/counted_driver.go.txt'),('semantics',semantics,HERE/'harness/semantics_driver.go.txt')]:
   dest=sandbox/(label+'-'+variant);shutil.copytree(source,dest)
   (dest/'go.mod').write_text('module perf.objectidentity\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => '+str(runtime)+'\n')
   shutil.copyfile(driver,dest/'driver.go')
   run(variant+'-'+label+'-build',['go','build','-mod=mod','-o',out/(variant+'-'+label),'.'],cwd=dest)
   for p in source.glob('*.go'):assert sha(p)==sha(dest/p.name)
 classes=out/'classes';classes.mkdir()
 run('javac',[JDK/'bin/javac','--release','21','-d',classes,PRIOR/'source/ObjectIdentityProbe.java',PRIOR/'harness/IdentityDriver.java',HERE/'source/QueueSemantics.java',HERE/'harness/SemanticsDriver.java'])
 java=[JDK/'bin/java','-XX:+UseSerialGC','-XX:ActiveProcessorCount=2','-Xms32m','-Xmx256m','-cp',classes]
 checks={v:run(v+'-semantics',java+['SemanticsDriver'] if v=='java' else [out/(v+'-semantics')],timeout=30).stdout for v in ['java','baseline','candidate']}
 assert len(set(checks.values()))==1,checks
 results=[]
 for seed in [17,97]:
  for mode in [0,1,2]:
   outputs={}
   for variant in (['java','baseline','candidate'] if seed==17 else ['candidate','baseline','java']):
    cmd=java+['IdentityDriver'] if variant=='java' else [out/(variant+'-app')]
    r=run(f'{variant}-seed{seed}-mode{mode}',cmd+[seed,mode],timeout=30);outputs[variant]=r.stdout
    results.append(dict(variant=variant,seed=seed,mode=mode,stdout_sha256=hashlib.sha256(r.stdout).hexdigest(),metrics=[json.loads(x) for x in r.stderr.decode().splitlines()]))
   assert len(set(outputs.values()))==1,(seed,mode)
 for variant in variants:
  dest=sandbox/('app-'+variant);shutil.copyfile(PRIOR/'harness/allocation_test.go.txt',dest/'identity_allocation_test.go')
  for trial in range(3):run(variant+'-allocation-'+str(trial),['go','test','-mod=mod','-run','^TestObjectIdentityAllocations$','-v','-count=1','.'],cwd=dest,timeout=30)
 fingerprints={v:{str(p.relative_to(runtime/'stdjava')):sha(p) for p in sorted((runtime/'stdjava').glob('*.go'))} for v,runtime in variants.items()}
 metadata=dict(scope='Isolated BFS queue candidate; allocation/reachability only; no timing claims',revision=REV,sandbox=str(sandbox),archive_sha256=sha(out/'snapshot.tar'),source_sha256=sha(PRIOR/'source/ObjectIdentityProbe.java'),generated_sha256=sha(generated/'ObjectIdentityProbe.go'),semantic_source_sha256=sha(HERE/'source/QueueSemantics.java'),semantic_generated_sha256=sha(semantics/'QueueSemantics.go'),patch_sha256=sha(HERE/'queue.patch'),runtime_fingerprints=fingerprints,parity=True,semantics_stdout=checks['java'].decode(),results=results)
 (out/'results.json').write_text(json.dumps(metadata,indent=2)+'\n')
 print(json.dumps(dict(output=str(out),sandbox=str(sandbox),parity=True),indent=2))
if __name__=='__main__':main()
