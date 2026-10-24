#!/usr/bin/env python3
"""Source-level JVM/Go monitor edges; deliberately reports existing mismatches.
Supply the raw output directory from run.py to reuse its isolated runtimes.
"""
import json,os,shutil,subprocess,sys
from pathlib import Path
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]
JDK=Path('/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')
def main():
 out=Path(sys.argv[1]).resolve();metadata=json.loads((out/'results.json').read_text());sandbox=Path(metadata['sandbox'])
 env=dict(os.environ,GOWORK='off',GOMAXPROCS='2',JAVA_HOME=str(JDK))
 records=[]
 def run(name,cmd,cwd=ROOT):
  r=subprocess.run([str(x) for x in cmd],cwd=cwd,env=env,capture_output=True,timeout=60)
  (out/(name+'.stdout')).write_bytes(r.stdout);(out/(name+'.stderr')).write_bytes(r.stderr)
  records.append(dict(name=name,command=[str(x) for x in cmd],cwd=str(cwd),exit_code=r.returncode))
  (out/'edges-commands.json').write_text(json.dumps(records,indent=2)+'\n')
  if r.returncode:raise RuntimeError(f'{name} failed; inspect {out}')
  return r.stdout.decode()
 gen=out/'edge-generated';run('edge-transpile',[out/'java2go','-strict','-w','-output',gen,HERE/'edges'])
 shutil.copyfile(HERE/'harness/edge_driver.go.txt',gen/'driver.go')
 outputs={}
 for variant in ['baseline','candidate']:
  (gen/'go.mod').write_text('module perf.monitoredges\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => '+str(sandbox/variant)+'\n')
  run('edge-'+variant+'-build',['go','build','-mod=mod','-o',out/('edge-'+variant),'.'],cwd=gen)
  outputs[variant]=run('edge-'+variant,[out/('edge-'+variant)])
 classes=out/'edge-classes';classes.mkdir(exist_ok=True)
 run('edge-javac',[JDK/'bin/javac','--release','21','-d',classes,HERE/'edges/MonitorEdges.java',HERE/'harness/EdgesDriver.java'])
 outputs['java']=run('edge-java',[JDK/'bin/java','-cp',classes,'EdgesDriver'])
 result=dict(scope='Source-level semantic edges; known mismatch observation, not a passing gate',outputs=outputs,
             candidate_equals_baseline=outputs['candidate']==outputs['baseline'],candidate_equals_java=outputs['candidate']==outputs['java'])
 (out/'edges-results.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result,indent=2))
if __name__=='__main__':main()
