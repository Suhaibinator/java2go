#!/usr/bin/env python3
"""Collect allocation diagnostics, not timings; gate every invocation on parity."""
import argparse, hashlib, json, os, subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);p.add_argument('--trials',type=int,default=3);a=p.parse_args()
root=Path(__file__).resolve().parents[3];base=a.scratch.resolve();metadata=json.loads((base/'metadata.json').read_text());accepted=Path(metadata['accepted'])
expected={r['seed']:r for r in json.loads((root/'testfiles/campaign/round01/data/oracle.json').read_text())['runs']}
env=dict(os.environ,GOMAXPROCS='2',GOMEMLIMIT='512MiB')
for key in ('GODEBUG','GOGC'):env.pop(key,None)
observations=[]
for seed in (17,41,97):
    javaout=base/f'java-{seed}'
    java=subprocess.run(['/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/java','-Dfile.encoding=UTF-8','-Duser.language=en','-Duser.country=US','-Duser.timezone=UTC','-cp',str(accepted/'java-oracle')+':'+str(accepted/'frozen/dependencies/commons-codec-1.22.1.jar'),'campaign.data.app.Main',str(seed),str(javaout)],cwd=base,capture_output=True,check=True)
    assert java.stdout==expected[seed]['stdout'].encode() and java.stderr==b'',seed
    javafiles={name.removeprefix('out/'):hashlib.sha256((javaout/name.removeprefix('out/')).read_bytes()).hexdigest() for name in expected[seed]['files']}
    assert javafiles=={name.removeprefix('out/'):digest for name,digest in expected[seed]['files'].items()},seed
    for trial in range(a.trials):
        prefix=f'seed-{seed}-trial-{trial}';stats=base/(prefix+'.json')
        go=subprocess.run([str(base/'allocationprobe'),str(seed),str(base/(prefix+'-out')),str(stats)],cwd=base,env=env,capture_output=True,check=True)
        (base/(prefix+'.stdout')).write_bytes(go.stdout)
        assert go.stdout==java.stdout*2 and go.stderr==b'',prefix
        measured=json.loads(stats.read_text());assert measured['warm_files']==javafiles and measured['measured_files']==javafiles,prefix
        observations.append({'seed':seed,'trial':trial,'allocated_bytes':measured['total_allocated_bytes'],'mallocs':measured['mallocs'],'profile':str(stats)})
(base/'allocation-summary.json').write_text(json.dumps({'metadata':metadata,'parity':'Every warm and observed Go invocation matches live Java stdout and all declared files.','observations':observations},indent=2)+'\n')
print(json.dumps(observations,indent=2))
