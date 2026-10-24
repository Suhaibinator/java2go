"""Recover pinned Java source only into a new external directory. No network or payload execution."""
from pathlib import Path,PurePosixPath
import argparse,hashlib,json,zipfile
B=Path(__file__).resolve().parents[1]
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
def relative(value):
 p=PurePosixPath(value);assert not p.is_absolute()and'..'not in p.parts,value;return Path(*p.parts)
def main():
 ap=argparse.ArgumentParser(description=__doc__);ap.add_argument('--artifact-dir',type=Path,required=True);ap.add_argument('--base-snapshot',type=Path,required=True);ap.add_argument('--output',type=Path,required=True);a=ap.parse_args()
 output=a.output.resolve();assert not output.exists()and output!=B and B not in output.parents,'Fresh external output required'
 manifest=json.loads((B/'MANIFEST.json').read_text())['files']
 for name,row in manifest.items():assert sha(B/relative(name))==row['sha256'],name
 base=json.loads((B/'BASE.json').read_text());inventory={}
 for p in sorted(a.base_snapshot.rglob('*')):
  assert not p.is_symlink(),str(p)
  if p.is_file():inventory[str(p.relative_to(a.base_snapshot))]=sha(p)
 assert len(inventory)==base['source_members']and hashlib.sha256((json.dumps(inventory,sort_keys=True,indent=2)+'\n').encode()).hexdigest()==base['source_manifest_sha256'],'Exact public217 source snapshot required'
 artifacts=json.loads((B/'dependencies/LOCKED-ARTIFACTS.json').read_text())['artifacts']
 for row in artifacts:assert sha(a.artifact_dir/relative(row['file']))==row['sha256'],row['file']
 groups=json.loads((B/'dependencies/SOURCE-GROUPS.json').read_text())['groups'];buffers={};result={}
 for app,group in groups.items():
  rows=[]
  for row in group['sources_in_actual_compile_order']:
   mode=row['retrieval']['kind']
   if mode=='bundle':data=(B/relative(row['retrieval']['path'])).read_bytes()
   elif mode=='base217':data=(a.base_snapshot/relative(row['retrieval']['path'])).read_bytes()
   elif mode=='source-archive':
    with zipfile.ZipFile(a.artifact_dir/relative(row['retrieval']['archive']))as archive:data=archive.read(row['retrieval']['entry'])
   else:raise ValueError(mode)
   assert hashlib.sha256(data).hexdigest()==row['sha256'],row['output']
   destination=relative(row['output']);assert destination not in buffers;buffers[destination]=data;rows.append({'path':row['output'],'sha256':row['sha256']})
  result[app]={'sources':rows,'resources':group['resources'],'source_count':len(rows)}
 output.mkdir(parents=True)
 for name,data in buffers.items():
  target=output/name;target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes(data)
 (output/'PREPARED-SOURCES.json').write_text(json.dumps({'status':'SOURCE_RECOVERY_ONLY_NO_JVM_GO_OR_ACTIVATION','groups':result},sort_keys=True,indent=2)+'\n')
 print(json.dumps({'source_only':True,'source_counts':{k:v['source_count']for k,v in result.items()},'prepared_root':str(output)}))
if __name__=='__main__':main()
