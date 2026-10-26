#!/usr/bin/env python3
"""Bounded fresh-process allocation comparison; all results checked against JVM."""
import argparse,hashlib,json,os,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args();base=a.scratch.resolve()
assert (base/'validation.json').exists(),'run validation first'
meta=json.loads((base/'metadata.json').read_text());expected={tuple(map(int,line.split(':')[:2])):int(line.split(':')[2]) for line in meta['java_observations']['BuilderWorkload'].splitlines()}
env=dict(os.environ,GOCACHE='/tmp/java2go-campaign-go-cache',GOMAXPROCS='2',GOMEMLIMIT='512MiB',GOGC='100');env.pop('GODEBUG',None)
results={name:[] for name in ('baseline','typed')}
for fork in range(3):
 for name in results:
  raw=subprocess.check_output([str(base/(name+'-allocations'))],env=env,timeout=180);(base/f'{name}-trial-{fork}.csv').write_bytes(raw)
  rows=[]
  for line in raw.decode().splitlines():
   seed,mode,repetition,result,allocated,mallocs=map(int,line.split(','));assert result==expected[(seed,mode)]
   rows.append({'seed':seed,'mode':mode,'repetition':repetition,'result':result,'bytes':allocated,'mallocs':mallocs})
  assert len(rows)==48;results[name].append(rows)
 print('Fresh allocation fork',fork,'matches JVM for both variants',flush=True)
profiles={}
for name in results:
 raw=subprocess.check_output([str(base/(name+'-allocations')),'profile'],env=env,timeout=180);(base/(name+'-profiles.jsonl')).write_bytes(raw)
 profiles[name]=[json.loads(line) for line in raw.decode().splitlines()]
 for p in profiles[name]:assert p['result']==expected[(p['seed'],p['mode'])]
# Confirm every generated application source remains compiler-produced.
for name,manifest in meta['generated_sha256'].items():
 for file,h in manifest.items():assert hashlib.sha256((base/name/file).read_bytes()).hexdigest()==h
(base/'evidence.json').write_text(json.dumps({'metadata':meta,'validation':json.loads((base/'validation.json').read_text()),'trials':results,'profiles':profiles},indent=2)+'\n')
print('All 288 measured runs and six profiles match JVM. No timings collected.')
