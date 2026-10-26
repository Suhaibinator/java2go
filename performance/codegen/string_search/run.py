#!/usr/bin/env python3
"""Run semantic matrix and allocation diagnostics only; no timing comparison."""
import argparse,hashlib,json,os,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args();base=a.scratch.resolve()
env=dict(os.environ,GOCACHE='/tmp/java2go-campaign-go-cache',GOMAXPROCS='2')
for key in ('GOGC','GODEBUG'):env.pop(key,None)
jdk='/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/'
subprocess.run([jdk+'javac','-d',str(base),str(base/'Oracle.java')],check=True)
java=subprocess.check_output([jdk+'java','-cp',str(base),'Oracle']);(base/'java.stdout').write_bytes(java)
lines=java.splitlines();expected=b'\n'.join(lines[:-4])+b'\n';java_null=dict(line.decode().split(':',1) for line in lines[-4:])
result={'java_search_cases':len(lines)-4,'java_null_cases':java_null,'oracle_sha256':hashlib.sha256(java).hexdigest(),'go_version':subprocess.check_output(['go','version'],text=True).strip(),'variants':{}}
for name in ('baseline','candidate'):
 subprocess.run(['go','build','-o',str(base/(name+'-oracle')),'./oracle'],cwd=base/name,env=env,check=True)
 got=subprocess.check_output([str(base/(name+'-oracle'))]);(base/(name+'.stdout')).write_bytes(got)
 if got!=expected:raise SystemExit(name+': Java search matrix mismatch')
 with (base/(name+'-compiler.log')).open('w') as log:
  subprocess.run(['go','build','-o',str(base/(name+'-allocations')),'-gcflags=github.com/NickyBoy89/java2go/stdjava=-m=2','./probe'],cwd=base/name,env=env,stdout=log,stderr=log,check=True)
 trials=[]
 for trial in range(3):
  data=subprocess.check_output([str(base/(name+'-allocations'))],env=env);(base/f'{name}-trial-{trial}.json').write_bytes(data);parsed=json.loads(data)
  assert parsed['null_checks']==java_null,(name,'Java null oracle');trials.append(parsed)
 result['variants'][name]=trials
 test=subprocess.run(['go','test','./stdjava','-count=1'],cwd=base/name,env=env,capture_output=True,text=True)
 (base/(name+'-runtime-tests.log')).write_text(test.stdout+test.stderr);test.check_returncode()
for b,c in zip(result['variants']['baseline'],result['variants']['candidate']):
 assert b['invalid_go_utf8_regression']==c['invalid_go_utf8_regression']
 for original,candidate in zip(b['observations'],c['observations']):assert (original['name'],original['result'])==(candidate['name'],candidate['result'])
(base/'evidence.json').write_text(json.dumps(result,indent=2)+'\n')
print('Both runtime copies pass Java matrix, Java null oracle, invalid-Go-UTF8 regression and full runtime tests.')
print('Allocation evidence:',base/'evidence.json')
