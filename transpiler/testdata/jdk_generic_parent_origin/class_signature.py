"""Pure bounded JVMS class Signature grammar. No class loading or source lookup.
The caller binds variables to actual declaration identities after parsing.
"""
class SignatureError(ValueError):pass
class Signature:
 def __init__(self,text): self.text=text;self.i=0;self.depth=0
 def peek(self): return self.text[self.i:self.i+1]
 def take(self):
  if self.i==len(self.text):raise SignatureError('truncated signature')
  c=self.text[self.i];self.i+=1;return c
 def expect(self,c):
  if self.take()!=c:raise SignatureError('expected '+c+' at '+str(self.i-1))
 def identifier(self,stops,binary=False):
  a=self.i
  while self.peek()and self.peek()not in stops:self.i+=1
  if self.i==a:raise SignatureError('empty identifier')
  value=self.text[a:self.i]
  forbidden='. ;[<>:'.replace(' ','') + ('' if binary else '/')
  if any(c in forbidden or ord(c)==0 for c in value) or binary and any(not p for p in value.split('/')):raise SignatureError('invalid signature identifier')
  return value
 def reference(self):
  self.depth+=1
  if self.depth>64:raise SignatureError('depth cap64')
  try:
   c=self.take()
   if c=='T':
    name=self.identifier(';');self.expect(';');return {'kind':'variable','name':name}
   if c=='[':
    rank=1
    while self.peek()=='[':self.i+=1;rank+=1
    if rank>255:raise SignatureError('array rank cap255')
    if self.peek()in 'BCDFIJSZ' and self.peek():item={'kind':'primitive','descriptor':self.take()}
    else:item=self.reference()
    return {'kind':'array','rank':rank,'component':item}
   if c!='L':raise SignatureError('expected reference, got '+repr(c))
   first=self.identifier('<.;',binary=True);parts=[{'binary':first,'arguments':self.arguments()}];binary=first
   while self.peek()=='.':
    self.i+=1;simple=self.identifier('<.;');binary+='$'+simple;parts.append({'binary':binary,'arguments':self.arguments()})
   self.expect(';');return {'kind':'nominal','binary':binary,'segments':parts}
  finally:self.depth-=1
 def arguments(self):
  if self.peek()!='<':return []
  self.i+=1;args=[]
  while self.peek()!='>':
   if not self.peek():raise SignatureError('unterminated type arguments')
   if self.peek()=='*':self.i+=1;args.append({'kind':'wildcard','variance':'unbounded'})
   elif self.peek()in '+-':variance=self.take();args.append({'kind':'wildcard','variance':'extends'if variance=='+'else'super','bound':self.reference()})
   else:args.append(self.reference())
   if len(args)>255:raise SignatureError('type argument cap255')
  self.i+=1
  if not args:raise SignatureError('empty generic arguments')
  return args
 def class_signature(self):
  params=[]
  if self.peek()=='<':
   self.i+=1
   while self.peek()!='>':
    name=self.identifier(':');self.expect(':');bounds=[]
    if self.peek()!=':':bounds.append(self.reference())
    while self.peek()==':':self.i+=1;bounds.append(self.reference())
    params.append({'name':name,'bounds':bounds})
    if len(params)>255:raise SignatureError('formal parameter cap255')
   self.i+=1
   if not params:raise SignatureError('empty type parameters')
  parents=[]
  while self.peek():
   p=self.reference()
   if p['kind']!='nominal':raise SignatureError('parent is not class type')
   parents.append(p)
  if not parents:raise SignatureError('missing superclass signature')
  names=[p['name']for p in params]
  if len(names)!=len(set(names)):raise SignatureError('duplicate class type parameter')
  return params,parents

def walk(n):
 yield n
 if n['kind']=='nominal':
  for s in n['segments']:
   for a in s['arguments']:yield from walk(a)
 elif n['kind']=='array':yield from walk(n['component'])
 elif n['kind']=='wildcard'and'bound'in n:yield from walk(n['bound'])
