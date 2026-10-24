#!/usr/bin/env python3
"""Run the existing JVM-backed UTF16 builder regression on isolated candidates."""
import argparse,os,shutil,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args();base=a.scratch.resolve()
env=dict(os.environ,GOCACHE='/tmp/java2go-campaign-go-cache',GOMAXPROCS='2',JAVA_HOME='/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')
for name in ('snapshot','edits','edits_output'):
 root=base/name
 if name!='snapshot':
  for f in (base/'snapshot').iterdir():
   if f.name=='stdjava' or (root/f.name).exists():continue
   if f.is_dir():shutil.copytree(f,root/f.name)
   else:shutil.copyfile(f,root/f.name)
 with (base/(name+'-utf16-regression.log')).open('w') as log:
  subprocess.run(['go','test','./transpiler','-run','^TestCampaignRuntimeStringBuilderUTF16$','-count=1','-v'],cwd=root,env=env,stdout=log,stderr=log,check=True,timeout=180)
 print(name,'passes existing Java-oracle UTF16 builder regression',flush=True)
