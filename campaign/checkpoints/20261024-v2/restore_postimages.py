#!/usr/bin/env python3
"""Review-only mapped blob restoration; refuses any target except exact public214."""
from pathlib import Path
import argparse
import hashlib
import json
import os
import shutil
import tempfile


def manifest(root):
    result = {}
    for path in root.rglob('*'):
        relative = path.relative_to(root)
        if relative.parts[0] == '.git':
            continue
        if path.is_symlink():
            raise SystemExit('Symlink refused: ' + str(relative))
        if path.is_file():
            result[relative.as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--candidate', required=True, choices=(
        'union-metadata-bounds-kind', 'raf-file-overload-v10',
        'sam-anonymous-functional-v1'))
    parser.add_argument('--target', type=Path, required=True)
    args = parser.parse_args()
    bundle = Path(__file__).resolve().parent
    target = args.target.resolve()
    base = json.loads((bundle / 'base/SOURCE-MANIFEST.json').read_text())
    if manifest(target) != base:
        raise SystemExit('Target is not exact complete public214; refusing writes')
    candidate = bundle / 'candidates' / args.candidate
    entries = json.loads((candidate / 'DELTA.json').read_text())['paths']
    # Validate every logical path and all blob bytes before modifying the disposable target.
    for entry in entries:
        logical = Path(entry['path'])
        if logical.is_absolute() or '..' in logical.parts:
            raise SystemExit('Unsafe logical path')
        if entry['operation'] != 'deleted':
            stored = Path(entry['stored_postimage'])
            if stored.is_absolute() or '..' in stored.parts:
                raise SystemExit('Unsafe blob path')
            data = (bundle / stored).read_bytes()
            if hashlib.sha256(data).hexdigest() != entry['postimage_sha256']:
                raise SystemExit('Blob hash mismatch')
    for entry in entries:
        destination = target / entry['path']
        if entry['operation'] == 'deleted':
            destination.unlink()
            continue
        destination.parent.mkdir(parents=True, exist_ok=True)
        # Replacing an inode preserves any hardlinked base donor unchanged.
        descriptor, temporary_name = tempfile.mkstemp(
            prefix='.checkpoint-restore-', dir=destination.parent)
        temporary = Path(temporary_name)
        try:
            with os.fdopen(descriptor, 'wb') as output:
                with (bundle / entry['stored_postimage']).open('rb') as source:
                    shutil.copyfileobj(source, output)
                os.fchmod(output.fileno(),
                          0o755 if entry['postimage_mode'] == '100755' else 0o644)
            os.replace(temporary, destination)
        finally:
            temporary.unlink(missing_ok=True)
    expected = json.loads((candidate / 'SOURCE-MANIFEST.json').read_text())
    if manifest(target) != expected:
        raise SystemExit('Restored full source does not match candidate')
    print('Exact mapped source restored:', args.candidate, 'UNACCEPTED; review only.')


if __name__ == '__main__':
    main()
