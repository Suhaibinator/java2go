#!/usr/bin/env python3
"""Fetch only committed, hash-pinned artifacts; retain upstream license metadata."""
import argparse
import hashlib
import json
from pathlib import Path
import tarfile
import urllib.request
import zipfile

ROOT = Path(__file__).resolve().parents[2]

def bootstrap(root=ROOT):
    lock = json.loads((root / 'campaign/dependencies.lock.json').read_text())
    cache = root / '.campaign/cache'
    cache.mkdir(parents=True, exist_ok=True)
    for artifact in lock['artifacts']:
        dest = cache / artifact['file']
        if not dest.exists():
            data = urllib.request.urlopen(artifact['url'], timeout=60).read()
            if hashlib.sha256(data).hexdigest() != artifact['sha256']:
                raise RuntimeError('download checksum mismatch: ' + artifact['file'])
            dest.write_bytes(data)
        if hashlib.sha256(dest.read_bytes()).hexdigest() != artifact['sha256']:
            raise RuntimeError('cache checksum mismatch: ' + artifact['file'])
        if artifact['kind'] == 'sources':
            out = root / '.campaign/sources' / (artifact['id'] + '-' + artifact['version'])
            out.mkdir(parents=True, exist_ok=True)
            with zipfile.ZipFile(dest) as archive:
                for member in archive.infolist():
                    target = (out / member.filename).resolve()
                    if not target.is_relative_to(out.resolve()):
                        raise RuntimeError('unsafe archive member: ' + member.filename)
                archive.extractall(out)
        elif artifact['kind'] == 'tool':
            out = root / '.campaign/tools'
            out.mkdir(parents=True, exist_ok=True)
            with tarfile.open(dest) as archive:
                archive.extractall(out, filter='data')
    maven_lock = json.loads((root / 'campaign/maven.lock.json').read_text())
    for artifact in maven_lock['artifacts']:
        dest = root / '.campaign/m2' / artifact['file']
        if not dest.exists():
            dest.parent.mkdir(parents=True, exist_ok=True)
            data = urllib.request.urlopen(artifact['url'], timeout=60).read()
            if hashlib.sha256(data).hexdigest() != artifact['sha256']:
                raise RuntimeError('Maven download checksum mismatch: ' + artifact['file'])
            dest.write_bytes(data)
        if hashlib.sha256(dest.read_bytes()).hexdigest() != artifact['sha256']:
            raise RuntimeError('Maven cache checksum mismatch: ' + artifact['file'])
    print('Pinned dependencies verified; Maven: .campaign/tools/apache-maven-3.9.16/bin/mvn')

if __name__ == '__main__':
    bootstrap()
