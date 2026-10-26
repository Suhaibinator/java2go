#!/usr/bin/env python3
"""Regenerate original Java through frozen baseline and isolated typed compiler."""
import argparse,hashlib,json,os,shutil,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);p.add_argument('--frozen',type=Path,default=Path('/tmp/java2go-stringbuilder-verified'));a=p.parse_args()
base=a.scratch.resolve();frozen=a.frozen.resolve();own=Path(__file__).resolve().parent;previous=own.parent/'stringbuilder'
if base.exists() and any(base.iterdir()):raise SystemExit('scratch must be empty')
old=json.loads((frozen/'metadata.json').read_text())
for name,h in old['compiler_and_runtime_sha256'].items():assert hashlib.sha256((frozen/'snapshot'/name).read_bytes()).hexdigest()==h,('frozen source changed',name)
assert hashlib.sha256((previous/'BuilderWorkload.java').read_bytes()).hexdigest()==old['source_sha256']
base.mkdir(parents=True,exist_ok=True)
for variant in ('baseline','typed'):
 shutil.copytree(frozen/'snapshot',base/variant)
source=base/'typed/transpiler/intrinsics_stdlib.go';s=source.read_text()
needle='''		converted := javaStringConversionExpr(valueNode, args[valueIndex], ctx, source)'''
assert s.count(needle)==1
s=s.replace(needle,'''		baseType, _ := parseJavaTypeString(javaType)
		if stripJavaQualifier(baseType) == "String" && resolveClassScopeByQualifiedName(ctx, baseType) == nil {
			args[valueIndex] = coerceArgumentToExpectedType(args[valueIndex], valueNode, "String", ctx, source)
			return methodCall(recv, goMethod+"String", args...)
		}
'''+needle);source.write_text(s)
shutil.copyfile(own/'typed_helpers.go.txt',base/'typed/stdjava/stringbuilder_typed.go')
subprocess.run(['gofmt','-w',str(source),str(base/'typed/stdjava/stringbuilder_typed.go')],check=True)
project=base/'project';java=project/'src/main/java/builderprobe';java.mkdir(parents=True)
for source in (previous/'BuilderWorkload.java',own/'ArgumentOracle.java'):shutil.copyfile(source,java/source.name)
(project/'pom.xml').write_text('<project><modelVersion>4.0.0</modelVersion><groupId>perf</groupId><artifactId>builder</artifactId><version>1</version></project>\n')
env=dict(os.environ,GOCACHE='/tmp/java2go-campaign-go-cache',GOMAXPROCS='2')
jdk='/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/'
subprocess.run([jdk+'javac','-d',str(base/'classes'),*[str(f) for f in java.glob('*.java')]],check=True)
observations={}
for main in ('BuilderWorkload','ArgumentOracle'):
 output=subprocess.check_output([jdk+'java','-cp',str(base/'classes'),'builderprobe.'+main]);(base/(main+'-java.stdout')).write_bytes(output);observations[main]=output
manifests={}
for variant in ('baseline','typed'):
 with (base/(variant+'-build.log')).open('w') as log:
  subprocess.run(['go','build','-o',str(base/(variant+'-java2go')),'./cmd/java2go'],cwd=base/variant,env=env,stdout=log,stderr=log,check=True)
  for main in ('BuilderWorkload','ArgumentOracle'):
   output=base/(variant+'-'+main)
   subprocess.run([str(base/(variant+'-java2go')),'-strict','-maven',str(project),'-main-class','builderprobe.'+main,'-runtime',str(base/variant),'-module','perf.generated/builder','-output',str(output)],cwd=base,env=env,stdout=log,stderr=log,check=True)
   subprocess.run(['go','build','-mod=mod','-o',str(base/(variant+'-'+main+'-app')),'./cmd/app'],cwd=output,env=env,stdout=log,stderr=log,check=True)
   got=subprocess.check_output([str(base/(variant+'-'+main+'-app'))],env=env);(base/(variant+'-'+main+'.stdout')).write_bytes(got)
   assert got==observations[main],(variant,main,'JVM parity')
   manifests[variant+'-'+main]={str(f.relative_to(output)):hashlib.sha256(f.read_bytes()).hexdigest() for f in sorted(output.rglob('*.go'))}
metadata={'revision':old['revision'],'source_sha256':old['source_sha256'],'oracle_source_sha256':hashlib.sha256((java/'ArgumentOracle.java').read_bytes()).hexdigest(),'snapshot_manifest_sha256':hashlib.sha256(json.dumps(old['compiler_and_runtime_sha256'],sort_keys=True).encode()).hexdigest(),'generated_sha256':manifests,'java_observations':{k:v.decode() for k,v in observations.items()},'go_version':subprocess.check_output(['go','version'],text=True).strip()}
(base/'metadata.json').write_text(json.dumps(metadata,indent=2)+'\n')
print('Unchanged stateful workload and supplemental argument oracle match JVM for both regenerated compilers.')
