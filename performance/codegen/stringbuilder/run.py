#!/usr/bin/env python3
"""Verify frozen generated code, then compare allocated bytes without timings."""
import argparse,hashlib,json,os,shutil,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args();base=a.scratch.resolve();own=Path(__file__).resolve().parent
meta=json.loads((base/'metadata.json').read_text());expected={tuple(map(int,line.split(':')[:2])):int(line.split(':')[2]) for line in (base/'java.stdout').read_text().splitlines()}
env=dict(os.environ,GOCACHE='/tmp/java2go-campaign-go-cache',GOMAXPROCS='2',GOMEMLIMIT='512MiB',GOGC='100');env.pop('GODEBUG',None)
subprocess.run(['python3',str(own/'candidates.py'),str(base)],check=True)
jdk='/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/'
subprocess.run([jdk+'javac','-cp',str(base/'classes'),'-d',str(base/'classes'),str(own/'AllocationDriver.java')],check=True)
commands={'java':[jdk+'java','-XX:ActiveProcessorCount=2','-Xms64m','-Xmx512m','-cp',str(base/'classes'),'AllocationDriver']}
for variant,runtime in [('baseline','snapshot'),('edits','edits'),('edits_output','edits_output')]:
 out=base/('generated-'+variant);shutil.copytree(base/'generated',out)
 for name,h in meta['generated_sha256'].items():assert hashlib.sha256((out/name).read_bytes()).hexdigest()==h
 subprocess.run(['go','mod','edit','-replace=github.com/NickyBoy89/java2go='+str(base/runtime)],cwd=out,env=env,check=True)
 probe=out/'cmd/allocations';probe.mkdir();shutil.copyfile(own/'allocation_driver.go.txt',probe/'main.go')
 with (base/(variant+'-build.log')).open('w') as log:
  subprocess.run(['go','build','-mod=mod','-o',str(base/(variant+'-app')),'./cmd/app'],cwd=out,env=env,stdout=log,stderr=log,check=True)
  subprocess.run(['go','build','-mod=mod','-o',str(base/(variant+'-allocations')),'./cmd/allocations'],cwd=out,env=env,stdout=log,stderr=log,check=True)
 got=subprocess.check_output([str(base/(variant+'-app'))],env=env);assert got==(base/'java.stdout').read_bytes(),(variant,'semantic parity')
 (base/(variant+'.stdout')).write_bytes(got)
 with (base/(variant+'-runtime-tests.log')).open('w') as log:
  subprocess.run(['go','test','./stdjava','-count=1'],cwd=base/runtime,env=env,stdout=log,stderr=log,check=True)
 commands[variant]=[str(base/(variant+'-allocations'))]
subprocess.run(['python3',str(own/'verify_utf16.py'),str(base)],check=True)
print('Exact semantic parity and all runtime tests passed before measurements.',flush=True)
results={name:[] for name in commands}
# Interleave fresh forks; no wall-clock data is collected.
for fork in range(3):
 for name,command in commands.items():
  raw=subprocess.check_output(command,env=env,timeout=180);(base/f'{name}-trial-{fork}.csv').write_bytes(raw)
  rows=[]
  for line in raw.decode().splitlines():
   values=list(map(int,line.split(',')));seed,mode,trial,result,allocated=values[:5]
   assert result==expected[(seed,mode)],(name,fork,seed,mode,trial,'parity')
   rows.append({'seed':seed,'mode':mode,'repetition':trial,'result':result,'bytes':allocated,**({'mallocs':values[5]} if len(values)>5 else {})})
  assert len(rows)==48;results[name].append(rows)
 print('Allocation fork',fork,'verified for Java and all three Go runtime variants.',flush=True)
profiles={}
for name in ('baseline','edits','edits_output'):
 raw=subprocess.check_output(commands[name]+['profile'],env=env,timeout=180);(base/(name+'-profiles.jsonl')).write_bytes(raw)
 profiles[name]=[json.loads(line) for line in raw.decode().splitlines()]
 for row in profiles[name]:assert row['result']==expected[(row['seed'],row['mode'])]
meta['candidate_stringbuilder_sha256']={name:hashlib.sha256((base/name/'stdjava/stringbuilder.go').read_bytes()).hexdigest() for name in ('edits','edits_output')}
(base/'allocation-evidence.json').write_text(json.dumps({'metadata':meta,'trials':results,'profiles':profiles},indent=2)+'\n')
print('All allocation and profile repetitions match the independent Java oracle. No timing measurements.')
