#!/usr/bin/env python3
"""Compile/check isolated real-runtime copies; --measure enables brief timings."""
import argparse, hashlib, json, os, subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);p.add_argument('--measure',action='store_true');a=p.parse_args()
base=a.scratch.resolve();probe=Path(__file__).resolve().parent
variants=('baseline','unsigned','cold')
env=dict(os.environ,GOCACHE='/tmp/java2go-campaign-go-cache',GOMAXPROCS='2',GOMEMLIMIT='512MiB')
for key in ('GOGC','GODEBUG'):env.pop(key,None)
jdk='/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/'
subprocess.run([jdk+'javac','-d',str(base),str(probe/'AssignOracle.java')],check=True)
expected=subprocess.check_output([jdk+'java','-cp',str(base),'AssignOracle']);(base/'java.expected').write_bytes(expected)
for name in variants:
    for label,flags in [('compiler','-m=2 -d=ssa/check_bce/debug=1'),('assembly','-S')]:
        with (base/(name+'-'+label+'.log')).open('w') as log:
            subprocess.run(['go','test','-c','-o',str(base/(name+'.test')),'-gcflags=github.com/NickyBoy89/java2go/probe='+flags,'./probe'],cwd=base/name,env=env,stdout=log,stderr=log,check=True)
    subprocess.run(['go','build','-o',str(base/(name+'-oracle')),'./oracle'],cwd=base/name,env=env,check=True)
    got=subprocess.check_output([str(base/(name+'-oracle'))]);(base/(name+'-oracle.stdout')).write_bytes(got)
    if got!=expected:raise SystemExit(name+': Java parity failed; no measurement allowed')
    subprocess.run([str(base/(name+'.test')),'-test.run=TestWideIndices','-test.count=1'],env=env,check=True)
    if name in ('baseline','unsigned'):
        result=subprocess.run(['go','test','./stdjava','-count=1'],cwd=base/name,env=env,capture_output=True,text=True)
        (base/(name+'-runtime-tests.log')).write_text(result.stdout+result.stderr)
        result.check_returncode()
print('All variants passed Java behavior and index checks; baseline/unsigned passed runtime tests.')
if not a.measure:raise SystemExit(0)
rows=[];orders=[('baseline','unsigned','cold'),('unsigned','cold','baseline'),('cold','baseline','unsigned')]*2
for block,order in enumerate(orders):
    for name in order:
        result=subprocess.run([str(base/(name+'.test')),'-test.run=TestWideIndices','-test.bench=Benchmark','-test.benchtime=200ms','-test.count=1'],env=env,capture_output=True,text=True,check=True)
        (base/f'{block}-{name}.bench').write_text(result.stdout)
        for line in result.stdout.splitlines():
            if line.startswith('Benchmark'):
                f=line.split();rows.append({'block':block,'variant':name,'benchmark':f[0].rsplit('-',1)[0],'iterations':int(f[1]),'ns_per_op':float(f[2]),'bytes_per_op':int(f[4]),'allocs_per_op':int(f[6])})
    (base/'measurements.json').write_text(json.dumps({'records':rows,'orders':orders,'go':subprocess.check_output(['go','version'],text=True).strip(),'GOMAXPROCS':2,'GOMEMLIMIT':'512MiB','benchtime':'200ms'},indent=2)+'\n')
print(base/'measurements.json')
