#!/usr/bin/env python3
"""Offline, byte-only JDK21/java.base nominal inventory. No classes execute."""
import argparse
import hashlib
import json
import pathlib
import struct
import zipfile

SCHEMA = 'java2go.jdk21.nominal.v1'
PROFILE = 'jdk21-java.base-unnamed-no-preview'
PINS = {
 'jmods/java.base.jmod':'0da0a7f5720bcdae901d45b8904ab1e7d3eb6fdd9729dcc88be19f6e94e24684',
 'lib/src.zip':'d887da897fc0065a2532500ee92320ce3bc633ccd6c4e637304045ce5d90de43',
 'lib/modules':'3925b48b275376ecc135eea32d50638d162d130d2149aa712875bcdcbad2d335',
 'lib/ct.sym':'92f865ec1bd9fc03cb96700c13cace8140e7b96905611d65199d80903fd84892',
}
def digest(data): return hashlib.sha256(data).hexdigest()
def encoded(value): return json.dumps(value,sort_keys=True,separators=(',',':'),ensure_ascii=True).encode()
class Reader:
 def __init__(self,data): self.data=data;self.pos=0
 def take(self,n):
  if n<0 or self.pos+n>len(self.data): raise ValueError('truncated class metadata')
  value=self.data[self.pos:self.pos+n];self.pos+=n;return value
 def u1(self): return self.take(1)[0]
 def u2(self): return struct.unpack('>H',self.take(2))[0]
 def u4(self): return struct.unpack('>I',self.take(4))[0]
 def end(self):
  if self.pos!=len(self.data): raise ValueError('unconsumed attribute/class metadata')
def mutf8(data):
 units=[];r=Reader(data)
 while r.pos<len(data):
  a=r.u1()
  if 1<=a<=127: units.append(a);continue
  if a&0xe0==0xc0:
   b=r.u1()
   if b&0xc0!=0x80: raise ValueError('invalid modified UTF8 continuation')
   v=((a&31)<<6)|(b&63)
   if v<128 and not(a==0xc0 and b==0x80): raise ValueError('overlong modified UTF8')
  elif a&0xf0==0xe0:
   b,c=r.u1(),r.u1()
   if b&0xc0!=0x80 or c&0xc0!=0x80: raise ValueError('invalid modified UTF8 continuation')
   v=((a&15)<<12)|((b&63)<<6)|(c&63)
   if v<2048: raise ValueError('overlong modified UTF8')
  else: raise ValueError('invalid modified UTF8 lead')
  units.append(v)
 return b''.join(struct.pack('>H',v) for v in units).decode('utf-16-be','surrogatepass')
def parse_class(data):
 r=Reader(data)
 if r.u4()!=0xcafebabe: raise ValueError('class magic')
 minor,major=r.u2(),r.u2()
 if major!=65 or minor not in (0,65535): raise ValueError('outside pinned Java21 class version')
 count=r.u2();cp=[None]*count;i=1
 while i<count:
  tag=r.u1()
  if tag==1: value=mutf8(r.take(r.u2()))
  elif tag in (3,4): value=r.take(4)
  elif tag in (5,6): value=r.take(8)
  elif tag in (7,8,16,19,20): value=r.u2()
  elif tag in (9,10,11,12,17,18): value=(r.u2(),r.u2())
  elif tag==15: value=(r.u1(),r.u2())
  else: raise ValueError('unknown constant pool tag')
  cp[i]=(tag,value);i+=2 if tag in (5,6) else 1
  if i>count: raise ValueError('invalid double constant pool slot')
 def at(index,tag):
  if not 0<index<count or cp[index] is None or cp[index][0]!=tag: raise ValueError('constant pool reference kind')
  return cp[index][1]
 def utf(index): return at(index,1)
 def named(index,tag): return utf(at(index,tag))
 def attrs(rd):
  result={}
  for _ in range(rd.u2()):
   name=utf(rd.u2());body=rd.take(rd.u4())
   if name in result: raise ValueError('duplicate attribute')
   result[name]=body
  return result
 access=r.u2();name=named(r.u2(),7);super_index=r.u2()
 super_name=named(super_index,7) if super_index else None
 interfaces=[named(r.u2(),7) for _ in range(r.u2())]
 for _ in range(2):
  for _ in range(r.u2()):
   r.u2();utf(r.u2());utf(r.u2());attrs(r)
 attributes=attrs(r);r.end()
 out={'binary':name,'super':super_name,'interfaces':interfaces,'signature':None,'access':access,'minor':minor,'major':major,'synthetic':bool(access&0x1000) or 'Synthetic' in attributes,'ownInnerRows':[],'enclosingMethod':False,'preview':minor==65535,'sourceFile':None}
 if 'Signature' in attributes:
  a=Reader(attributes['Signature']);out['signature']=utf(a.u2());a.end()
 if 'Synthetic' in attributes and attributes['Synthetic']: raise ValueError('Synthetic length')
 if 'SourceFile' in attributes:
  a=Reader(attributes['SourceFile']);out['sourceFile']=utf(a.u2());a.end()
 if 'InnerClasses' in attributes:
  a=Reader(attributes['InnerClasses'])
  for _ in range(a.u2()):
   inner,outer,simple,flags=a.u2(),a.u2(),a.u2(),a.u2();inner_name=named(inner,7) if inner else None
   row={'inner':inner_name,'outer':named(outer,7) if outer else None,'simple':utf(simple) if simple else None,'access':flags}
   if inner_name==name: out['ownInnerRows'].append(row)
  a.end()
  if len(out['ownInnerRows'])>1: raise ValueError('duplicate self InnerClasses relation')
 if 'EnclosingMethod' in attributes:
  a=Reader(attributes['EnclosingMethod']);named(a.u2(),7);method=a.u2()
  if method: at(method,12)
  a.end();out['enclosingMethod']=True
 def annotation(a):
  descriptor=utf(a.u2())
  for _ in range(a.u2()): utf(a.u2());element(a)
  return descriptor
 def element(a):
  tag=chr(a.u1())
  if tag in 'BCDFIJSZs':
   index=a.u2();at(index,1 if tag=='s' else 6 if tag=='D' else 4 if tag=='F' else 5 if tag=='J' else 3)
  elif tag=='e': utf(a.u2());utf(a.u2())
  elif tag=='c': utf(a.u2())
  elif tag=='@': annotation(a)
  elif tag=='[':
   for _ in range(a.u2()): element(a)
  else: raise ValueError('unknown annotation element')
 for label in ('RuntimeVisibleAnnotations','RuntimeInvisibleAnnotations'):
  if label in attributes:
   a=Reader(attributes[label]);annotations=[annotation(a) for _ in range(a.u2())];a.end()
   if 'Ljdk/internal/javac/PreviewFeature;' in annotations: out['preview']=True
 if 'Module' in attributes:
  a=Reader(attributes['Module']);module=named(a.u2(),19);a.u2();version=a.u2()
  if version: utf(version)
  for _ in range(a.u2()):
   named(a.u2(),19);a.u2();version=a.u2()
   if version: utf(version)
  exports={}
  for _ in range(a.u2()):
   pkg=named(a.u2(),20);flags=a.u2();targets=[named(a.u2(),19) for _ in range(a.u2())]
   if pkg in exports: raise ValueError('duplicate export')
   exports[pkg]={'flags':flags,'targets':targets}
  for _ in range(a.u2()):
   named(a.u2(),20);a.u2()
   for _ in range(a.u2()): named(a.u2(),19)
  for _ in range(a.u2()): named(a.u2(),7)
  for _ in range(a.u2()):
   named(a.u2(),7)
   for _ in range(a.u2()): named(a.u2(),7)
  a.end();out['module']={'name':module,'exports':exports}
 return out

def checked_zip(path):
 z=zipfile.ZipFile(path);seen=set()
 for name in z.namelist():
  if name in seen or name.startswith('/') or '\\' in name or '..' in name.split('/'): raise ValueError('duplicate/unsafe archive entry')
  seen.add(name)
 return z

def generate(jdk,json_path,go_path):
 sources={}
 for rel,want in PINS.items():
  data=(jdk/rel).read_bytes();got=digest(data)
  if got!=want: raise ValueError('pinned JDK metadata mismatch: '+rel)
  sources[rel]={'sha256':got,'bytes':len(data)}
 release=(jdk/'release').read_bytes()
 if b'JAVA_VERSION="21.0.6"' not in release: raise ValueError('JDK release mismatch')
 sources['release']={'sha256':digest(release),'bytes':len(release)}
 jmod=checked_zip(jdk/'jmods/java.base.jmod');src=checked_zip(jdk/'lib/src.zip');module_data=jmod.read('classes/module-info.class');module=parse_class(module_data)['module']
 if module['name']!='java.base': raise ValueError('wrong module')
 packages={p.replace('/','.'): {'module':'java.base','export':'unqualified','completeClassMetadataScan':True,'sourceCrossCheck':'archive-path-only','classEntries':[],'declarations':{}} for p,e in module['exports'].items() if not e['targets']}
 classes={}
 for entry in sorted(jmod.namelist()):
  if not entry.startswith('classes/') or not entry.endswith('.class') or entry=='classes/module-info.class': continue
  binary=entry[len('classes/'):-len('.class')];pkg,sep,simple=binary.rpartition('/');package=pkg.replace('/','.')
  if not sep or package not in packages: continue
  data=jmod.read(entry);c=parse_class(data)
  if c['binary']!=binary: raise ValueError('entry/this_class mismatch')
  classes[binary]=dict(c,classEntry=entry,classEntrySHA=digest(data),package=package)
  package_record=packages[package];package_record['classEntries'].append({'entry':entry,'sha256':digest(data)})
  if simple=='package-info' or c['ownInnerRows'] or c['enclosingMethod']: continue
  source=c['sourceFile']
  if not source or '/' in source or '\\' in source: raise ValueError('unknown top-level compilation unit')
  src_entry='java.base/'+pkg+'/'+source
  source_present=src_entry in src.namelist()
  record={'module':'java.base','package':package,'simpleBinaryName':simple,'binaryName':binary.replace('/','.'),'access':c['access'],'interface':bool(c['access']&0x200),'public':bool(c['access']&1),'preview':c['preview'],'synthetic':c['synthetic'],'nesting':'topLevel','classEntry':entry,'classEntrySHA':digest(data),'sourceEntry':src_entry,'sourceEntryStatus':'present' if source_present else 'absent-from-pinned-source-archive','sourceEntrySHA':digest(src.read(src_entry)) if source_present else None}
  if simple in package_record['declarations']: raise ValueError('duplicate top-level declaration')
  package_record['declarations'][simple]=record
 for package,record in packages.items():
  record['classEntrySetSHA']=digest(encoded(record['classEntries']));record['topLevelSetSHA']=digest(encoded(record['declarations']));record['certificateID']=digest(encoded({'profile':PROFILE,'package':package,'archive':PINS['jmods/java.base.jmod'],'classEntries':record['classEntrySetSHA'],'topLevels':record['topLevelSetSHA']}))
 # Member completeness is independent of package top-level completeness. The
 # relation comes from the class's own InnerClasses row, never its binary spelling.
 owners={};members={}
 for binary,c in sorted(classes.items()):
  if binary.endswith('/package-info'): continue
  owner=binary.replace('/','.')
  owners[owner]={'super':c['super'].replace('/','.') if c['super'] else '', 'interfaces':[v.replace('/','.') for v in c['interfaces']], 'members':{}, 'classEntrySHA':c['classEntrySHA']}
 for binary,c in sorted(classes.items()):
  rows=c['ownInnerRows']
  if not rows or c['enclosingMethod']: continue
  row=rows[0]
  if not row['outer'] or not row['simple']: continue
  if row['outer'] not in classes: raise ValueError('ordinary member missing actual owner')
  outer=classes[row['outer']]
  if outer['package']!=c['package']: raise ValueError('member owner package mismatch')
  name=binary.replace('/','.');owner=row['outer'].replace('/','.')
  source=c['sourceFile']
  if not source or '/' in source or '\\' in source: raise ValueError('unknown member compilation unit')
  source_entry='java.base/'+binary.rpartition('/')[0]+'/'+source
  source_present=source_entry in src.namelist()
  record={'binaryName':name,'declaringOwner':owner,'simpleName':row['simple'],'module':'java.base','package':c['package'],'export':'unqualified','access':row['access'],'public':bool(row['access']&1),'static':bool(row['access']&8),'interface':bool(row['access']&0x200),'preview':c['preview'],'synthetic':c['synthetic'],'signature':c['signature'],'classEntry':c['classEntry'],'classEntrySHA':c['classEntrySHA'],'sourceEntry':source_entry,'sourceEntrySHA':digest(src.read(source_entry)) if source_present else None}
  if row['simple'] in owners[owner]['members']: raise ValueError('duplicate direct member name')
  owners[owner]['members'][row['simple']]=name;members[name]=record
 def chain(name,path):
  if name in path: raise ValueError('cyclic declaring owner chain')
  if name in members:
   row=members[name];parents=chain(row['declaringOwner'],path|{name})
   return parents+[row['simpleName']] if parents else []
  c=classes[name.replace('.','/')]
  if c['ownInnerRows'] or c['enclosingMethod']: return []
  return [name]
 for name,row in members.items():
  row['sourceChain']=chain(name,set())
 for name,record in owners.items():
  record['certificateID']=digest(encoded({'profile':PROFILE,'owner':name,'archive':PINS['jmods/java.base.jmod'],'record':record}))
 result={'schema':SCHEMA,'profile':PROFILE,'module':'java.base','javaRelease':21,'previewEnabled':False,'exportPolicy':'unqualified','nestingPolicy':'direct-top-level-only','generatorSHA':digest(pathlib.Path(__file__).read_bytes()),'inputs':sources,'moduleInfoSHA':digest(module_data),'packages':packages,'memberDeclarations':members,'memberOwners':owners}
 json_bytes=json.dumps(result,indent=2,sort_keys=True,ensure_ascii=True).encode()+b'\n';json_path.parent.mkdir(parents=True,exist_ok=True);json_path.write_bytes(json_bytes)
 lines=['// Code generated by testdata/jdk21nominal/generate.py; DO NOT EDIT.','// Schema '+SCHEMA+'; profile '+PROFILE+'.','// Inventory JSON SHA256: '+digest(json_bytes),'package transpiler','','var jdk21NominalPackages = map[string]jdk21NominalPackage{']
 for package,record in sorted(packages.items()):
  lines.append(json.dumps(package)+': {certificate: '+json.dumps(record['certificateID'])+', declarations: map[string]jdk21NominalDeclaration{')
  for simple,decl in sorted(record['declarations'].items()):
   lines.append(json.dumps(simple)+': {binaryName: '+json.dumps(decl['binaryName'])+', public: '+str(decl['public']).lower()+', isInterface: '+str(decl['interface']).lower()+', preview: '+str(decl['preview']).lower()+', synthetic: '+str(decl['synthetic']).lower()+'},')
  lines.append('}},')
 lines.append('}');go_path.write_text('\n'.join(lines)+'\n');print(json.dumps({'packages':len(packages),'declarations':sum(len(p['declarations']) for p in packages.values()),'jsonSHA':digest(json_bytes),'goSHA':digest(go_path.read_bytes())},sort_keys=True))
 member_lines=['// Code generated by testdata/jdk21nominal/generate.py; DO NOT EDIT.','// Inventory JSON SHA256: '+digest(json_bytes),'package transpiler','','var jdk21MemberDeclarations = map[string]jdk21MemberDeclaration{']
 for name,row in sorted(members.items()):
  member_lines.append(json.dumps(name)+': {binaryName:'+json.dumps(name)+', declaringOwner:'+json.dumps(row['declaringOwner'])+', simpleName:'+json.dumps(row['simpleName'])+', public:'+str(row['public']).lower()+', static:'+str(row['static']).lower()+', isInterface:'+str(row['interface']).lower()+', preview:'+str(row['preview']).lower()+', synthetic:'+str(row['synthetic']).lower()+', sourceChain: []string{'+','.join(json.dumps(v) for v in row['sourceChain'])+'}, signature:'+json.dumps(row['signature'] or '')+', classSHA:'+json.dumps(row['classEntrySHA'])+', sourceSHA:'+json.dumps(row['sourceEntrySHA'] or '')+'},')
 member_lines+=['}','','var jdk21MemberOwners = map[string]jdk21MemberOwner{']
 for name,row in sorted(owners.items()):
  member_lines.append(json.dumps(name)+': {certificate:'+json.dumps(row['certificateID'])+', superclass:'+json.dumps(row['super'])+', interfaces: []string{'+','.join(json.dumps(v) for v in row['interfaces'])+'}, members: map[string]string{'+','.join(json.dumps(k)+':'+json.dumps(v) for k,v in sorted(row['members'].items()))+'}},')
 member_lines.append('}')
 go_path.with_name('jdk21_nominal_member_inventory_data.go').write_text('\n'.join(member_lines)+'\n')
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--jdk',type=pathlib.Path,required=True);p.add_argument('--json',type=pathlib.Path,required=True);p.add_argument('--go',type=pathlib.Path,required=True);a=p.parse_args();generate(a.jdk,a.json,a.go)
