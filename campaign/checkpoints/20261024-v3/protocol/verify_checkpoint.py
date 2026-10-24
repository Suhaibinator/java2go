"""Verify inactive checkpoint bytes and archived observation hashes. No payload execution."""
from pathlib import Path
import argparse,ast,hashlib,json
B=Path(__file__).resolve().parents[1]
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
def main():
 ap=argparse.ArgumentParser(description=__doc__);ap.parse_args();m=json.loads((B/'MANIFEST.json').read_text());expected=m['files'];actual={str(p.relative_to(B))for p in B.rglob('*')if p.is_file()};assert actual==set(expected)|{'MANIFEST.json'}
 for name,row in expected.items():
  p=B/name;assert not p.is_symlink()and p.stat().st_size==row['bytes']and sha(p)==row['sha256'],name
 assert len(list(B.glob('apps/*/src/main/java/**/*.java')))==14
 assert not list(B.rglob('*.go'))and not list(B.rglob('fixture.json'))
 assert not any(p.suffix in['.class','.jar','.pyc','.so','.dylib','.a']for p in B.rglob('*')if p.is_file())
 for app in['gson','supplier']:
  index=json.loads((B/f'author-oracles/{app}/INDEX.json').read_text());assert len(index['actual_runs'])==9
  for row in index['actual_runs']:
   assert row['exit']==0 and row['repeat']in[1,2,3]
   for stream in['stdout','stderr']:
    v=row[stream];p=B/v['included_path'];assert sha(p)==v['sha256']and p.stat().st_size==v['bytes']
   for name,v in row['files'].items():
    p=B/v['included_path'];assert sha(p)==v['sha256']and p.stat().st_size==v['bytes'],name
  assert sorted((r['seed'],r['repeat'])for r in index['actual_runs'])==[(s,r)for s in[17,41,97]for r in[1,2,3]]
 for p in(B/'protocol').glob('*.py'):ast.parse(p.read_text(),filename=str(p))
 print(json.dumps({'status':'CHECKPOINT_BYTES_AND_ARCHIVED_AUTHOR_OBSERVATIONS_VALID_NOT_ACTIVATION','files':len(expected),'bytes':sum(r['bytes']for r in expected.values()),'authored_Java_units':14,'source_only_no_payload':True}))
if __name__=='__main__':main()
