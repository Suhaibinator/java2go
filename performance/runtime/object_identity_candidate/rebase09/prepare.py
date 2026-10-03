#!/usr/bin/env python3
"""Read/copy/hash only: prepare a current-source queue candidate; never build or test."""
import datetime,difflib,hashlib,json,shutil,tempfile
from pathlib import Path
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[3]
CORE=['astutil','cmd','dot','e2e','fuzz','nodeutil','parsing','project','stdjava','symbol','testfiles','transpiler']
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
 sandbox=Path(tempfile.mkdtemp(prefix='java2go-queue-round09-'))
 baseline=sandbox/'baseline';baseline.mkdir()
 manifest={}
 for name in ['api.go','go.mod','go.sum']:
  manifest[name]=sha(ROOT/name);shutil.copyfile(ROOT/name,baseline/name)
 for name in CORE:
  for source in sorted((ROOT/name).rglob('*')):
   if source.is_file():
    rel=source.relative_to(ROOT);manifest[str(rel)]=sha(source)
    dest=baseline/rel;dest.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(source,dest)
 for rel,want in manifest.items():
  if sha(ROOT/rel)!=want or sha(baseline/rel)!=want:raise RuntimeError('Source changed during snapshot: '+rel)
 candidate=sandbox/'candidate';shutil.copytree(baseline,candidate)
 p=candidate/'stdjava/reference_arrays.go';original=p.read_text()
 first='queue := []TypeID{actual}\n\tfor len(queue) > 0 {\n\t\tcurrent := queue[0]\n\t\tqueue = queue[1:]'
 second='queue := []TypeID{actual}\n\tresult := make([]TypeID, 0, 4)\n\tfor len(queue) > 0 {\n\t\tcurrent := queue[0]\n\t\tqueue = queue[1:]'
 queue='var initialQueue [8]TypeID\n\tqueue := initialQueue[:1]\n\tqueue[0] = actual\n'
 loop='\tfor head := 0; head < len(queue); head++ {\n\t\tcurrent := queue[head]'
 assert original.count(first)==1 and original.count(second)==1
 changed=original.replace(first,queue+loop).replace(second,queue+'\tresult := make([]TypeID, 0, 4)\n'+loop)
 p.write_text(changed)
 patch=''.join(difflib.unified_diff(original.splitlines(True),changed.splitlines(True),fromfile='a/stdjava/reference_arrays.go',tofile='b/stdjava/reference_arrays.go',n=0))
 (HERE/'queue.patch').write_text(patch)
 assert patch==(HERE.parent/'queue.patch').read_text(),'Queue context changed: review before preparation'
 fingerprints={}
 for variant,path in [('baseline',baseline),('candidate',candidate)]:
  fingerprints[variant]={rel:sha(path/rel) for rel in manifest}
 changed_paths=[rel for rel in manifest if fingerprints['baseline'][rel]!=fingerprints['candidate'][rel]]
 assert changed_paths==['stdjava/reference_arrays.go'],changed_paths
 inputs=sandbox/'inputs';shutil.copytree(HERE.parent/'harness',inputs/'candidate-harness');shutil.copytree(HERE.parent/'source',inputs/'candidate-source')
 prior=HERE.parents[1]/'object_identity'
 shutil.copytree(prior/'harness',inputs/'identity-harness');shutil.copytree(prior/'source',inputs/'identity-source')
 metadata=dict(status='PREPARED_NOT_EXECUTED',captured_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),provenance='Content-addressed current worktree snapshot; parent labels completed round09. No Git command or commit assertion.',source_root=str(ROOT),sandbox=str(sandbox),source_manifest_sha256=hashlib.sha256(json.dumps(manifest,sort_keys=True).encode()).hexdigest(),source_manifest=manifest,queue_patch_sha256=sha(HERE/'queue.patch'),candidate_file_sha256=sha(p),baseline_file_sha256=sha(baseline/'stdjava/reference_arrays.go'),changed_paths=changed_paths,manifest_sha256={k:hashlib.sha256(json.dumps(v,sort_keys=True).encode()).hexdigest() for k,v in fingerprints.items()},input_hashes={str(p.relative_to(inputs)):sha(p) for p in sorted(inputs.rglob('*')) if p.is_file()})
 (HERE/'prepared.json').write_text(json.dumps(metadata,indent=2)+'\n')
 (sandbox/'provenance.json').write_text(json.dumps(metadata,indent=2)+'\n')
 print(json.dumps({k:metadata[k] for k in ['status','sandbox','source_manifest_sha256','queue_patch_sha256','baseline_file_sha256','candidate_file_sha256']},indent=2))
if __name__=='__main__':main()
