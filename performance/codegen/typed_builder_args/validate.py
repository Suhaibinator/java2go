#!/usr/bin/env python3
"""Build allocation drivers and validate runtime/behavior before measurement."""
import argparse,hashlib,json,os,shutil,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args();base=a.scratch.resolve();own=Path(__file__).resolve().parent
metadata=json.loads((base/'metadata.json').read_text());env=dict(os.environ,GOCACHE='/tmp/java2go-campaign-go-cache',GOMAXPROCS='2',JAVA_HOME='/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home')
tests='^(TestRuntime_StringBuilder|TestRuntime_StringBuilderOverloadText|TestCampaignRuntimeStringBuilderUTF16|TestCampaignRuntimeStringBuilderChain|TestNullableStringReferencesPreserveJavaSemantics)$'
for variant in ('baseline','typed'):
 output=base/(variant+'-BuilderWorkload')
 for name,h in metadata['generated_sha256'][variant+'-BuilderWorkload'].items():assert hashlib.sha256((output/name).read_bytes()).hexdigest()==h
 probe=output/'cmd/allocations';probe.mkdir();shutil.copyfile(own.parent/'stringbuilder/allocation_driver.go.txt',probe/'main.go')
 with (base/(variant+'-compiler.log')).open('w') as log:
  subprocess.run(['go','build','-mod=mod','-gcflags=all=-m=2','-o',str(base/(variant+'-allocations')),'./cmd/allocations'],cwd=output,env=env,stdout=log,stderr=log,check=True,timeout=180)
 with (base/(variant+'-runtime-tests.log')).open('w') as log:
  subprocess.run(['go','test','./stdjava','-count=1'],cwd=base/variant,env=env,stdout=log,stderr=log,check=True,timeout=180)
 with (base/(variant+'-semantic-tests.log')).open('w') as log:
  subprocess.run(['go','test','./transpiler','-run',tests,'-count=1','-v'],cwd=base/variant,env=env,stdout=log,stderr=log,check=True,timeout=180)
 print(variant,'runtime suite and five generated-behavior tests pass',flush=True)
(base/'validation.json').write_text(json.dumps({'runtime_suites':'both pass','semantic_test_pattern':tests,'semantic_tests':'both pass','generated_application_files':'hashes unchanged after compiler generation'},indent=2)+'\n')
