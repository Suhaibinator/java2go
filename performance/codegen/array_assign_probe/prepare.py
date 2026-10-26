#!/usr/bin/env python3
"""Prepare isolated real-runtime variants; never edits repository runtime files."""
from pathlib import Path
import argparse, shutil, subprocess, json
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args()
root=Path(__file__).resolve().parents[3]; probe=Path(__file__).resolve().parent
scratch=a.scratch.resolve()
if scratch.exists() and any(scratch.iterdir()): raise SystemExit('scratch directory must be empty')
scratch.mkdir(parents=True,exist_ok=True)
frozen=scratch/'snapshot'; frozen.mkdir()
for f in ['go.mod','go.sum']: shutil.copyfile(root/f,frozen/f)
shutil.copytree(root/'stdjava',frozen/'stdjava')
original=(frozen/'stdjava/reference_arrays.go').read_text()
old='''\tindex64 := int64(index)
\tif index64 < 0 || index64 >= int64(len(array.Elements)) {
\t\tpanic(NewArrayIndexOutOfBoundsException("array index out of bounds"))
\t}
\treturn int(index64)'''
new='''\tif uint64(index) >= uint64(len(array.Elements)) {
\t\tpanic(NewArrayIndexOutOfBoundsException("array index out of bounds"))
\t}
\treturn int(index)'''
# Restrict replacement to primitive helper, not similarly shaped reference helper.
start=original.index('func primitiveArrayIndex[');end=original.index('\nfunc PrimitiveArrayGet[',start)
assert old in original[start:end]
unsigned=original[:start]+original[start:end].replace(old,new)+original[end:]
# Deliberately evaluate a cold-function factoring candidate too; compiler evidence decides.
cold=original[:start]+original[start:end].replace('panic(NewNullPointerException("array access on null"))','panicPrimitiveArrayNull()').replace('panic(NewArrayIndexOutOfBoundsException("array index out of bounds"))','panicPrimitiveArrayBounds()')+original[end:]
cold+='''\n//go:noinline
func panicPrimitiveArrayNull() { panic(NewNullPointerException("array access on null")) }
//go:noinline
func panicPrimitiveArrayBounds() { panic(NewArrayIndexOutOfBoundsException("array index out of bounds")) }
'''
for name,source in [('baseline',original),('unsigned',unsigned),('cold',cold)]:
    out=scratch/name;out.mkdir(exist_ok=True)
    for f in ['go.mod','go.sum']:shutil.copyfile(frozen/f,out/f)
    shutil.copytree(frozen/'stdjava',out/'stdjava',dirs_exist_ok=True)
    (out/'stdjava/reference_arrays.go').write_text(source)
    (out/'probe').mkdir(exist_ok=True);shutil.copyfile(probe/'probe_test.go',out/'probe/probe_test.go')
    (out/'oracle').mkdir(exist_ok=True);shutil.copyfile(probe/'oracle.go.txt',out/'oracle/main.go')
(scratch/'metadata.json').write_text(json.dumps({'root':str(root),'variants':['baseline','unsigned','cold']},indent=2)+'\n')
print(scratch)
