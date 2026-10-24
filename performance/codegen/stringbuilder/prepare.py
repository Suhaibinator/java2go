#!/usr/bin/env python3
"""Freeze compiler/runtime and compile unchanged Java source into isolated Go."""
import argparse, hashlib, io, json, os, shutil, subprocess, tarfile
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args()
root=Path(__file__).resolve().parents[3];own=Path(__file__).resolve().parent;out=a.scratch.resolve()
if out.exists() and any(out.iterdir()):raise SystemExit('scratch must be empty')
out.mkdir(parents=True,exist_ok=True);snapshot=out/'snapshot';snapshot.mkdir()
revision='8cb6cd5'
archive=subprocess.check_output(['git','archive',revision,'api.go','go.mod','go.sum','astutil','cmd/java2go','dot','nodeutil','parsing','project','symbol','transpiler','stdjava'],cwd=root)
with tarfile.open(fileobj=io.BytesIO(archive)) as tar:tar.extractall(snapshot,filter='data')
manifest={str(f.relative_to(snapshot)):hashlib.sha256(f.read_bytes()).hexdigest() for f in sorted(snapshot.rglob('*')) if f.is_file()}
project=out/'project';source=project/'src/main/java/builderprobe';source.mkdir(parents=True)
shutil.copyfile(own/'BuilderWorkload.java',source/'BuilderWorkload.java')
(project/'pom.xml').write_text('<project><modelVersion>4.0.0</modelVersion><groupId>perf</groupId><artifactId>builder</artifactId><version>1</version></project>\n')
env=dict(os.environ,GOCACHE='/tmp/java2go-campaign-go-cache',GOMAXPROCS='2')
with (out/'build-transpiler.log').open('w') as log:
 subprocess.run(['go','build','-o',str(out/'java2go'),'./cmd/java2go'],cwd=snapshot,env=env,stdout=log,stderr=log,check=True)
with (out/'transpile.log').open('w') as log:
 subprocess.run([str(out/'java2go'),'-strict','-maven',str(project),'-main-class','builderprobe.BuilderWorkload','-runtime',str(snapshot),'-module','perf.generated/builder','-output',str(out/'generated')],cwd=out,env=env,stdout=log,stderr=log,check=True)
subprocess.run(['go','build','-mod=mod','-o',str(out/'baseline-app'),'./cmd/app'],cwd=out/'generated',env=env,check=True)
jdk='/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/'
subprocess.run([jdk+'javac','-d',str(out/'classes'),str(source/'BuilderWorkload.java')],check=True)
java=subprocess.check_output([jdk+'java','-cp',str(out/'classes'),'builderprobe.BuilderWorkload'])
go=subprocess.check_output([str(out/'baseline-app')],env=env)
(out/'java.stdout').write_bytes(java);(out/'baseline.stdout').write_bytes(go)
if java!=go:raise SystemExit('Java/generated Go parity mismatch')
generated={str(f.relative_to(out/'generated')):hashlib.sha256(f.read_bytes()).hexdigest() for f in sorted((out/'generated').rglob('*.go'))}
(out/'metadata.json').write_text(json.dumps({'revision':revision,'compiler_and_runtime_sha256':manifest,'source_sha256':hashlib.sha256((source/'BuilderWorkload.java').read_bytes()).hexdigest(),'generated_sha256':generated,'oracle_sha256':hashlib.sha256(java).hexdigest(),'go_version':subprocess.check_output(['go','version'],text=True).strip()},indent=2)+'\n')
print('Exact Java/generated Go parity for all 12 seed/mode runs. Scratch:',out)
