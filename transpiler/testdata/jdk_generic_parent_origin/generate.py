#!/usr/bin/env python3
"""Generate the bounded native generic-parent records from checked-in JDK facts.
No Java application or JDK class executes. Facts were extracted independently;
PROVENANCE.json pins their classfile/source verification and tool inputs.
"""
import argparse
import hashlib
import json
import pathlib
import shutil
import subprocess

HERE = pathlib.Path(__file__).resolve().parent

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def generate():
    # Offline classfile-derived facts replace the earlier bounded reflection table.
    # Main/options and all source/runtime consumers retain their existing APIs.
    import importlib.util
    import sys
    sys.dont_write_bytecode = True
    provenance = json.loads((HERE / 'PROVENANCE.json').read_text())
    producer = HERE / 'class_parent_producer.py'
    if digest(producer) != provenance['class_parent_producer_sha256']:
        raise ValueError('class parent producer pin mismatch')
    spec = importlib.util.spec_from_file_location('jdk21_class_parent_producer', producer)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    output, coverage = module.generate(HERE)
    if module.serialized(coverage) != (HERE / 'SYSTEMATIC-COVERAGE.json').read_bytes():
        raise ValueError('generated closed-coverage/held-shape ledger differs')
    return output

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    parser.add_argument('--gofmt', default=shutil.which('gofmt'))
    parser.add_argument('--output', type=pathlib.Path, default=HERE.parents[1] / 'jdk21_generic_parent_origin_data.go')
    args = parser.parse_args()
    if not args.gofmt:
        parser.error('provide --gofmt /path/to/gofmt')
    output = subprocess.run([args.gofmt], input=generate(), capture_output=True, check=True).stdout
    if args.check:
        if args.output.read_bytes() != output:
            raise SystemExit('generated parent data differs')
        print('generated parent data matches checked-in facts')
    else:
        args.output.write_bytes(output)

if __name__ == '__main__':
    main()
