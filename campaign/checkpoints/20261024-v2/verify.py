#!/usr/bin/env python3
"""Checkpoint hashes and full source hashes only; no semantic acceptance."""
from pathlib import Path
import argparse
import hashlib
import json


def hashfile(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def source_manifest(root):
    result = {}
    for path in root.rglob('*'):
        relative = path.relative_to(root)
        if relative.parts[0] == '.git':
            continue  # Git administrative metadata is outside the source manifest.
        if path.is_symlink():
            raise SystemExit('Symlink refused: ' + str(relative))
        if path.is_file():
            result[relative.as_posix()] = hashfile(path)
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--source', type=Path)
    parser.add_argument('--candidate', default='base')
    args = parser.parse_args()
    root = Path(__file__).resolve().parent
    if args.source:
        if args.candidate != 'base' and args.candidate not in (
            'union-metadata-bounds-kind', 'raf-file-overload-v10',
            'sam-anonymous-functional-v1'
        ):
            raise SystemExit('Unknown candidate ID')
        relative = 'base' if args.candidate == 'base' else 'candidates/' + args.candidate
        expected = json.loads((root / relative / 'SOURCE-MANIFEST.json').read_text())
        actual = source_manifest(args.source.resolve())
        wrong = sorted(key for key in set(expected) | set(actual)
                       if expected.get(key) != actual.get(key))
        if wrong:
            raise SystemExit('SOURCE MISMATCH: ' + ', '.join(wrong))
        print('Exact full source verified:', args.candidate, len(actual),
              'files. No semantic verdict.')
        return
    expected = json.loads((root / 'MANIFEST.json').read_text())['files']
    actual = {}
    for path in root.rglob('*'):
        relative = path.relative_to(root).as_posix()
        if path.is_symlink():
            raise SystemExit('Symlink refused: ' + relative)
        if path.is_file() and relative != 'MANIFEST.json':
            actual[relative] = {'sha256': hashfile(path), 'bytes': path.stat().st_size}
    if actual != expected:
        raise SystemExit('BUNDLE HASH OR COMPLETE INVENTORY MISMATCH')
    print('Bundle complete inventory and hashes verified. Check MANIFEST.json against the external coordinator digest.')


if __name__ == '__main__':
    main()
