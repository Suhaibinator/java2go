#!/usr/bin/env python3
"""Focused reproduction; creates only private copies, records no CPU benchmark."""
import argparse,hashlib,json,os,shutil,signal,subprocess,tempfile,time
from pathlib import Path
HERE=Path(__file__).resolve().parent
EXPECTED=json.loads((HERE/'evidence.json').read_text())
JDK=Path('/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')
CORE=['astutil','cmd','dot','e2e','fuzz','nodeutil','parsing','project','stdjava','symbol','testfiles','transpiler']
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--baseline',type=Path,required=True);parser.add_argument('--released',action='store_true');args=parser.parse_args()
 if not args.released:parser.error('Coordinate a slot before providing --released.')
 assert sha(args.baseline/'stdjava/bigdecimal.go')==EXPECTED['baseline_bigdecimal_sha256'],'Supply the pre-optimization baseline'
 for rel,want in EXPECTED['measured_source_hashes'].items():assert sha(HERE/rel)==want,rel
 assert sha(HERE/'equal-scale.patch')==EXPECTED['patch_sha256']
 out=Path(tempfile.mkdtemp(prefix='decimal-compare-repro-',dir='/private/tmp'));base=out/'baseline';base.mkdir();source_hashes={}
 for name in ['api.go','go.mod','go.sum']+CORE:
  source=args.baseline/name
  for p in ([source] if source.is_file() else sorted(p for p in source.rglob('*') if p.is_file())):
   rel=p.relative_to(args.baseline);source_hashes[str(rel)]=sha(p);dest=base/rel;dest.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(p,dest);assert sha(dest)==sha(p)
 candidate=out/'candidate';shutil.copytree(base,candidate)
 file=candidate/'stdjava/bigdecimal.go';text=file.read_text();needle='\tReferenceRequireNonNull(other)\n\tleftSign, rightSign := value.coefficient.Sign(), other.coefficient.Sign()'
 assert text.count(needle)==1
 file.write_text(text.replace(needle,'\tReferenceRequireNonNull(other)\n\tif value.scale == other.scale {\n\t\treturn int32(value.coefficient.Cmp(&other.coefficient))\n\t}\n\tleftSign, rightSign := value.coefficient.Sign(), other.coefficient.Sign()'))
 assert sha(file)==EXPECTED['candidate_bigdecimal_sha256']
 env=dict(os.environ,JAVA_HOME=str(JDK),PATH=str(JDK/'bin')+os.pathsep+os.environ['PATH'],LANG='en_US.UTF-8',LC_ALL='en_US.UTF-8',TZ='UTC',GOWORK='off',GOMAXPROCS='2',GOMEMLIMIT='256MiB',GOGC='100',GOCACHE='/private/tmp/java2go-campaign-go-cache',TMPDIR='/private/tmp')
 records=[];deadline=time.monotonic()+900;success=False;failure=None;frozen_apps={}
 def run(name,cmd,cwd=out,build=False,expected=0):
  limit=min(300 if build else 60,deadline-time.monotonic());assert limit>0,'Aggregate 900-second limit'
  cmd=[str(x) for x in cmd];timed_out=False
  with (out/(name+'.stdout')).open('wb') as a,(out/(name+'.stderr')).open('wb') as b:
   process=subprocess.Popen(cmd,cwd=cwd,env=env,stdout=a,stderr=b,start_new_session=True)
   try:code=process.wait(timeout=limit)
   except subprocess.TimeoutExpired:
    timed_out=True
    try:os.killpg(process.pid,signal.SIGKILL)
    except ProcessLookupError:pass
    code=process.wait()
  stdout=(out/(name+'.stdout')).read_bytes();stderr=(out/(name+'.stderr')).read_bytes()
  records.append(dict(name=name,command=cmd,cwd=str(cwd),deadline_seconds=limit,timed_out=timed_out,exit_code=code,stdout_sha256=hashlib.sha256(stdout).hexdigest(),stderr_sha256=hashlib.sha256(stderr).hexdigest()))
  (out/'commands.json').write_text(json.dumps(records,indent=2)+'\n')
  assert not timed_out and code==expected,(name,code,timed_out)
  return stdout,stderr
 def check_sources(label):
  for variant in ['baseline','candidate']:
   for rel,want in source_hashes.items():
    if variant=='candidate' and rel=='stdjava/bigdecimal.go':want=EXPECTED['candidate_bigdecimal_sha256']
    assert sha(out/variant/rel)==want,(label,variant,rel)
   fixture=out/variant/'stdjava/performance_equal_scale_test.go'
   if fixture.exists():assert sha(fixture)==EXPECTED['measured_source_hashes']['harness/regression_test.go.txt']
  for path,want in frozen_apps.items():assert sha(Path(path))==want,(label,path)
  (out/(label+'-source-hashes.json')).write_text(json.dumps(dict(baseline=source_hashes,candidate_bigdecimal=sha(file),application=frozen_apps),indent=2)+'\n')
 try:
  check_sources('before');run('go-version',['go','version']);run('jdk-version',[JDK/'bin/java','-version'])
  compiler=out/'java2go';run('compiler',['go','build','-o',compiler,'./cmd/java2go'],base,True)
  generated=out/'generated';run('transpile',[compiler,'-strict','-w','-output',generated,HERE/'source'])
  generated_hashes={p.name:sha(p) for p in generated.glob('*.go')}
  (out/'generated-hashes.json').write_text(json.dumps(dict(actual=generated_hashes,historical=EXPECTED['generated_hashes'],matches_historical=generated_hashes==EXPECTED['generated_hashes']),indent=2)+'\n')
  for variant in ['baseline','candidate']:
   runtime=out/variant;shutil.copyfile(HERE/'harness/regression_test.go.txt',runtime/'stdjava/performance_equal_scale_test.go')
   binary=out/(variant+'-runtime');run(variant+'-runtime-build',['go','test','-c','-o',binary,'./stdjava'],runtime,True)
   run(variant+'-runtime-semantic',[binary,'-test.run','^TestEqualScaleCandidate','-test.skip','^TestEqualScaleCandidateAllocation$','-test.timeout=60s'],runtime/'stdjava')
   stdout,_=run(variant+'-allocation-guard',[binary,'-test.run','^TestEqualScaleCandidateAllocation$','-test.timeout=60s'],runtime/'stdjava',expected=1 if variant=='baseline' else 0)
   if variant=='baseline':assert b'equal-scale compare allocated ' in stdout,'Not the expected allocation failure'
   app=out/(variant+'-app');shutil.copytree(generated,app);shutil.copyfile(HERE/'harness/driver.go.txt',app/'driver.go')
   (app/'go.mod').write_text('module perf.decimalcompare\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => '+str(runtime)+'\n')
   run(variant+'-build',['go','build','-mod=mod','-o',out/(variant+'-bin'),'.'],app,True)
   assert {name:sha(app/name) for name in generated_hashes}==generated_hashes
   frozen_apps.update({str(p):sha(p) for p in app.iterdir() if p.is_file() and (p.suffix=='.go' or p.name in ['go.mod','go.sum'])})
  classes=out/'classes';classes.mkdir();run('javac',[JDK/'bin/javac','--release','21','-d',classes,HERE/'source/DecimalCompareProbe.java',HERE/'harness/DecimalCompareDriver.java'],build=True)
  for seed in [17,97,65537]:
   for mode in [0,1,2]:
    observed=[]
    for variant in ['java','baseline','candidate']:
     cmd=[JDK/'bin/java','-XX:+UseSerialGC','-XX:ActiveProcessorCount=2','-Xms32m','-Xmx256m','-cp',classes,'DecimalCompareDriver'] if variant=='java' else [out/(variant+'-bin')]
     observed.append(run(f'{variant}-{seed}-{mode}',cmd+[seed,128,mode,8]))
    assert observed[0]==observed[1]==observed[2],('stdout/stderr parity',seed,mode)
  for variant in ['baseline','candidate']:
   app=out/(variant+'-app');shutil.copyfile(HERE/'harness/allocation_test.go.txt',app/'allocation_test.go')
   binary=out/(variant+'-allocation');run(variant+'-allocation-build',['go','test','-c','-mod=mod','-o',binary,'.'],app,True)
   for trial in range(3):run(variant+'-trial-'+str(trial),[binary,'-test.run','^TestDecimalCompareWorkloadAllocations$','-test.count=1','-test.timeout=60s','-test.v'],app)
  success=True
 except Exception as exc:failure=repr(exc)
 finally:
  try:check_sources('after')
  except Exception as exc:success=False;failure=(failure or '')+' source verification: '+repr(exc)
  summary=dict(success=success,error=failure,output=str(out),scope='Focused semantic/allocation reproduction; not full integration or CPU benchmark')
  (out/'summary.json').write_text(json.dumps(summary,indent=2)+'\n');print(json.dumps(summary,indent=2))
 if not success:raise SystemExit(1)
if __name__=='__main__':main()
