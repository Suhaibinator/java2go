#!/usr/bin/env python3
"""Copy accepted generated code unchanged; add a separate allocation-only driver."""
import argparse, hashlib, io, json, os, shutil, subprocess, tarfile
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args()
root=Path(__file__).resolve().parents[3];scratch=a.scratch.resolve(); accepted=root/'.campaign/runs/20260928T050238Z-2636650079'
if scratch.exists() and any(scratch.iterdir()):raise SystemExit('scratch must be empty')
scratch.mkdir(parents=True,exist_ok=True);runtime=scratch/'runtime';runtime.mkdir()
archive=subprocess.check_output(['git','archive','563fddb','stdjava','go.mod','go.sum'],cwd=root)
with tarfile.open(fileobj=io.BytesIO(archive)) as tar:tar.extractall(runtime,filter='data')
shutil.copytree(accepted/'generated',scratch/'generated')
probe=scratch/'generated/cmd/allocationprobe';probe.mkdir();shutil.copyfile(Path(__file__).with_name('allocation_driver.go.txt'),probe/'main.go')
env=dict(os.environ,GOCACHE='/tmp/java2go-campaign-go-cache')
subprocess.run(['go','mod','edit','-replace=github.com/NickyBoy89/java2go='+str(runtime)],cwd=scratch/'generated',env=env,check=True)
subprocess.run(['go','build','-o',str(scratch/'allocationprobe'),'./cmd/allocationprobe'],cwd=scratch/'generated',env=env,check=True)
manifest={str(f.relative_to(accepted/'generated')):hashlib.sha256(f.read_bytes()).hexdigest() for f in (accepted/'generated').rglob('*.go')}
for name,h in manifest.items():assert hashlib.sha256((scratch/'generated'/name).read_bytes()).hexdigest()==h
(scratch/'metadata.json').write_text(json.dumps({'accepted':str(accepted),'runtime_commit':'563fddbe837ba558ae3d1f57c9a947a179f845af','generated_sha256':manifest,'go_version':subprocess.check_output(['go','version'],text=True).strip()},indent=2)+'\n')
print(scratch)
