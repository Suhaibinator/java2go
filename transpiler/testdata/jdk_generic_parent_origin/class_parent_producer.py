#!/usr/bin/env python3
"""Pinned java.base classfile recipes, with a closed, representable Go projection.
No Java classes execute. Metadata traversal never grants source/API visibility.
"""
import argparse
import copy
import hashlib
import importlib.util
import json
import pathlib
import sys

sys.dont_write_bytecode = True
HERE = pathlib.Path(__file__).resolve().parent

def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module

signature = load_module('jdk21_class_signature', HERE / 'class_signature.py')

def digest(data):
    return hashlib.sha256(data).hexdigest()

def encoded(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=True).encode()

def serialized(value):
    return (json.dumps(value, sort_keys=True, indent=2, ensure_ascii=True) + '\n').encode()

def read_pinned(path, expected):
    data = path.read_bytes()
    if digest(data) != expected:
        raise ValueError('metadata pin mismatch: ' + str(path))
    return data

def extract(jdk_home, here=HERE):
    """Byte-only extraction through the existing checked nominal class reader."""
    provenance = json.loads((here / 'PROVENANCE.json').read_text())
    reader_path = here.parent / 'jdk21nominal/generate.py'
    read_pinned(reader_path, provenance['nominal_reader_sha256'])
    reader = load_module('jdk21_nominal_reader', reader_path)
    for relative, expected in provenance['jdk_pins'].items():
        read_pinned(jdk_home / relative, expected)
    archive = jdk_home / 'jmods/java.base.jmod'
    records = {}
    byte_count = 0
    with reader.checked_zip(archive) as source:
        module = reader.parse_class(source.read('classes/module-info.class'))['module']['name']
        if module != 'java.base':
            raise ValueError('outside declared java.base profile')
        for entry in sorted(source.namelist()):
            if not entry.startswith('classes/') or not entry.endswith('.class') or entry == 'classes/module-info.class':
                continue
            data = source.read(entry)
            byte_count += len(data)
            if len(data) > 4 * 1024 * 1024 or byte_count > 128 * 1024 * 1024 or len(records) >= 16000:
                raise ValueError('bounded class metadata corpus exceeded')
            row = reader.parse_class(data)
            if row['binary'] != entry[8:-6] or row['binary'] in records:
                raise ValueError('class entry/binary identity mismatch')
            records[row['binary']] = {key: row[key] for key in (
                'binary', 'super', 'interfaces', 'signature', 'access', 'preview',
                'synthetic', 'ownInnerRows', 'enclosingMethod')}
            records[row['binary']].update(module=module, classEntry=entry, class_sha256=digest(data))
    return {'schema': 'java2go.jdk21.generic-classfiles.v2',
            'profile': reader.PROFILE, 'archive_sha256': digest(archive.read_bytes()),
            'records': records}

def bind_records(facts):
    """Resolve variables using only actual own/lexical declaration ownership."""
    classes = facts['records']
    parsed = {}
    for name, row in sorted(classes.items()):
        if row['module'] != 'java.base' or row['binary'] != name or row['classEntry'] != 'classes/' + name + '.class':
            raise ValueError('foreign or inconsistent class metadata identity')
        if len(row['ownInnerRows']) > 1:
            raise ValueError('ambiguous actual InnerClasses owner')
        if row['signature']:
            parameters, parents = signature.Signature(row['signature']).class_signature()
            if [parent['binary'] for parent in parents] != [row['super']] + row['interfaces']:
                raise ValueError('Signature/header erasure disagreement: ' + name)
        else:
            parameters = []
            parents = [{'kind': 'nominal', 'binary': parent,
                        'segments': [{'binary': parent, 'arguments': []}]} for parent in
                       ([row['super']] if row['super'] else []) + row['interfaces']]
        parsed[name] = {'parameters': parameters, 'parents': parents}

    def environment(name):
        values, visited = {}, set()
        while name:
            if name in visited:
                raise ValueError('cyclic lexical class owner')
            visited.add(name)
            row, parsed_row = classes.get(name), parsed.get(name)
            if row is None or parsed_row is None:
                return values, 'missing lexical declaration'
            for slot, parameter in enumerate(parsed_row['parameters']):
                values.setdefault(parameter['name'], {'module': row['module'],
                    'owner': name.replace('/', '.'), 'class_sha256': row['class_sha256'],
                    'slot': slot, 'name': parameter['name']})
            if row['enclosingMethod']:
                return values, 'executable lexical binder unavailable'
            owners = row['ownInnerRows']
            if not owners or owners[0]['access'] & 8:
                return values, None
            name = owners[0]['outer']
        return values, None

    result = {}
    for name, row in sorted(classes.items()):
        record = copy.deepcopy(parsed[name])
        env, unavailable = environment(name)
        issues = set()
        for node in [n for parameter in record['parameters'] for bound in parameter['bounds'] for n in signature.walk(bound)] + [n for parent in record['parents'] for n in signature.walk(parent)]:
            if node['kind'] == 'variable':
                if node['name'] not in env:
                    issues.add('unresolved variable: ' + node['name'] + ': ' + str(unavailable))
                else:
                    node.update(kind='parameter', declaration=env[node['name']])
            elif node['kind'] == 'nominal':
                previous = None
                for segment in node['segments']:
                    target = parsed.get(segment['binary'])
                    if target is None:
                        issues.add('missing referenced declaration: ' + segment['binary'])
                        continue
                    arity = len(target['parameters'])
                    segment['declarationArity'] = arity
                    segment['raw'] = arity > 0 and not segment['arguments']
                    if segment['arguments'] and len(segment['arguments']) != arity:
                        issues.add('argument arity mismatch: ' + segment['binary'])
                    if previous is not None:
                        owners = classes[segment['binary']]['ownInnerRows']
                        if len(owners) != 1 or owners[0]['outer'] != previous:
                            issues.add('unproved owner segment relation: ' + segment['binary'])
                    previous = segment['binary']
        record['parameters'] = [{'module': row['module'], 'owner': name.replace('/', '.'),
            'class_sha256': row['class_sha256'], 'slot': index, 'name': parameter['name'],
            'bounds': parameter['bounds']} for index, parameter in enumerate(record['parameters'])]
        record['javaParents'] = record['parents'][1:] if row['access'] & 0x200 else record['parents']
        record['issues'] = sorted(issues)
        record['classfile'] = row
        result[name] = record
    return result

def representable(record):
    reasons, references = set(record['issues']), set()
    own = {(p['owner'], p['class_sha256'], p['slot']) for p in record['parameters']}
    for parent in record['javaParents']:
        for node in signature.walk(parent):
            kind = node['kind']
            if kind == 'parameter':
                parameter = node['declaration']
                if (parameter['owner'], parameter['class_sha256'], parameter['slot']) not in own:
                    reasons.add('captured outer declaration parameter')
            elif kind == 'nominal':
                if len(node['segments']) != 1:
                    reasons.add('parameterized owner chain')
                references.add(node['binary'])
            elif kind == 'wildcard':
                reasons.add('wildcard parent recipe')
            elif kind == 'primitive':
                reasons.add('primitive array recipe component')
            elif kind not in ('array',):
                reasons.add('unbound/unknown recipe kind: ' + kind)
    return sorted(reasons), references

def close_records(records, profile_owners):
    """Greatest complete reference closure; inheritance cycles separately refused."""
    reasons, references = {}, {}
    for name, row in records.items():
        reasons[name], references[name] = representable(row)
    state = {}
    def hierarchy(name, path):
        if name in path:
            return False
        if name in state:
            return state[name]
        row = records.get(name)
        value = row is not None and all(hierarchy(p['binary'], path | {name}) for p in row['javaParents'])
        state[name] = value
        return value
    for name in records:
        if not hierarchy(name, set()):
            reasons[name].append('missing/cyclic parent hierarchy')
    supported = {name for name in records if not reasons[name]}
    while True:
        missing = {name: sorted(references[name] - supported) for name in supported if references[name] - supported}
        if not missing:
            break
        for name, dependencies in missing.items():
            reasons[name].append('unsupported reference closure: ' + ','.join(dependencies))
        supported.difference_update(missing)
    emitted = set()
    def visit(name):
        if name in emitted:
            return
        if name not in supported:
            raise ValueError('incomplete emitted record closure')
        emitted.add(name)
        for dependency in sorted(references[name]):
            visit(dependency)
    for name in sorted(profile_owners & supported):
        visit(name)
    return {name: records[name] for name in sorted(emitted)}, {
        name: reasons[name] for name in sorted(records) if name not in supported}

def old_recipe(node):
    if node['kind'] == 'parameter':
        declaration = node['declaration']
        return {'kind': 'parameter', 'owner': declaration['owner'], 'index': declaration['slot'], 'name': declaration['name']}
    if node['kind'] == 'nominal' and len(node['segments']) == 1:
        return {'kind': 'nominal', 'owner': node['binary'].replace('/', '.'),
                'arguments': [old_recipe(a) for a in node['segments'][0]['arguments']]}
    raise ValueError('outside old calibration shape')

def verify_calibration(here, records):
    for old in json.loads((here / 'JDK-GENERIC-PARENT-FACTS.json').read_text())['owners']:
        row = records[old['owner'].replace('.', '/')]
        current = {'owner': old['owner'], 'class_sha256': row['classfile']['class_sha256'],
                   'class_signature': row['classfile']['signature'],
                   'parameters': [{'kind': 'parameter', 'owner': p['owner'], 'index': p['slot'], 'name': p['name']} for p in row['parameters']],
                   'parents': [old_recipe(p) for p in row['javaParents']]}
        if current != {key: old[key] for key in current}:
            raise ValueError('old eight calibration changed: ' + old['owner'])
    for old in json.loads((here / 'JDK-RAW-TERMINALS.json').read_text()):
        row = records[old['owner'].replace('.', '/')]
        if row['classfile']['class_sha256'] != old['class_sha256'] or [old_recipe(p) for p in row['parents']] != old['parents']:
            raise ValueError('raw terminal classfile proof changed')

def prepare(here=HERE):
    provenance = json.loads((here / 'PROVENANCE.json').read_text())
    read_pinned(here / 'class_signature.py', provenance['signature_parser_sha256'])
    read_pinned(here / 'JDK-GENERIC-PARENT-FACTS.json', provenance['facts_sha256'])
    read_pinned(here / 'JDK-RAW-TERMINALS.json', provenance['terminal_sha256'])
    raw = read_pinned(here / 'SYSTEMATIC-CLASSFILES.json', provenance['systematic_facts_sha256'])
    facts = json.loads(raw)
    if facts['schema'] != 'java2go.jdk21.generic-classfiles.v2' or facts['profile'] != 'jdk21-java.base-unnamed-no-preview' or facts['archive_sha256'] != provenance['jdk_pins']['jmods/java.base.jmod']:
        raise ValueError('systematic profile/archive/schema mismatch')
    inventory = json.loads(read_pinned(here.parent / 'jdk21nominal/inventory.json', provenance['nominal_inventory_sha256']))
    records = bind_records(facts)
    verify_calibration(here, records)
    profile = {name.replace('.', '/') for name in inventory['memberOwners']}
    if not profile <= records.keys():
        raise ValueError('profile declaration missing from classfile facts')
    emitted, held = close_records(records, profile)
    coverage = {'schema': 'java2go.jdk21.generic-parent-coverage.v2',
                'profile': facts['profile'], 'classfile_records': len(records),
                'class_signatures': sum(bool(r['classfile']['signature']) for r in records.values()),
                'profile_records': len(profile), 'emitted_closed_records': len(emitted),
                'profile_records_emitted': len(profile & emitted.keys()),
                'old_eight_exact': 8, 'held': held,
                'policy': 'No source visibility/API admission; only whole representable reference/parent closures. Held edges are never erased or replaced.'}
    return facts, emitted, coverage

def generate(here=HERE):
    facts, rows, coverage = prepare(here)
    q = json.dumps
    certificates = {name: digest(encoded({'profile': facts['profile'], 'archive': facts['archive_sha256'], 'record': row})) for name, row in rows.items()}
    indexes = {name: index for index, name in enumerate(rows)}
    lines = ['package transpiler\n', 'import "github.com/NickyBoy89/java2go/symbol"\n',
        '// Code generated by testdata/jdk_generic_parent_origin/generate.py; DO NOT EDIT.\n',
        '// Pinned classfile facts, held-shape ledger, and provenance are in that relative directory.\n',
        'var jdk21GenericParentCertificates = map[string]string{\n']
    for name in rows:
        lines.append('\t' + q(name.replace('/', '.')) + ': ' + q(certificates[name]) + ',\n')
    lines.append('}\n\nvar jdk21GenericParentRecords = buildJDK21GenericParentRecords()\n\nfunc buildJDK21GenericParentRecords() map[string]jdk21GenericParentRecord {\n')
    for name, row in rows.items():
        for parameter in row['parameters']:
            lines.append('\tp%d_%d := &symbol.TypeParamDeclaration{SourceName: %s, GoName: %s}\n' % (indexes[name], parameter['slot'], q(parameter['name']), q(parameter['name'])))
    def recipe(node):
        kind = node['kind']
        if kind == 'parameter':
            token = node['declaration']; owner = token['owner'].replace('.', '/')
            parameter = rows[owner]['parameters'][token['slot']]
            if any(parameter[key] != token[key] for key in ('owner', 'slot', 'class_sha256', 'module', 'name')):
                raise ValueError('unowned recipe parameter')
            return 'inferredJavaTypeOrigin{parameter: p%d_%d, javaType: %s}' % (indexes[owner], token['slot'], q(token['name']))
        if kind == 'array':
            component = recipe(node['component'])
            # These are typed origin fields, not a Go type or caller-name proof.
            return component[:-1] + ', arrayRank: %d}' % node['rank']
        if kind != 'nominal' or len(node['segments']) != 1 or node['binary'] not in rows:
            raise ValueError('unrepresentable/unclosed recipe escaped filter')
        args = node['segments'][0]['arguments']
        return 'inferredJavaTypeOrigin{externalName: ' + q(node['binary'].replace('/', '.')) + ', javaType: ' + q(node['binary'].replace('/', '.')) + (', arguments: []inferredJavaTypeOrigin{' + ','.join(recipe(arg) for arg in args) + '}' if args else '') + '}'
    lines.append('\treturn map[string]jdk21GenericParentRecord{\n')
    for name, row in rows.items():
        parameters = ','.join('p%d_%d' % (indexes[name], p['slot']) for p in row['parameters'])
        parents = ','.join(recipe(p) for p in row['javaParents'])
        lines.append('\t\t' + q(name.replace('/', '.')) + ': {certificate: ' + q(certificates[name]) + ',parameters: []*symbol.TypeParamDeclaration{' + parameters + '},parents: []inferredJavaTypeOrigin{' + parents + '}},\n')
    lines.append('\t}\n}\n')
    return ''.join(lines).encode(), coverage

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--extract', type=pathlib.Path, metavar='PINNED_JDK_HOME')
    parser.add_argument('--coverage', action='store_true')
    args = parser.parse_args()
    if args.extract:
        (HERE / 'SYSTEMATIC-CLASSFILES.json').write_bytes(serialized(extract(args.extract)))
    elif args.coverage:
        _, _, coverage = prepare()
        (HERE / 'SYSTEMATIC-COVERAGE.json').write_bytes(serialized(coverage))
    else:
        parser.error('choose --extract or --coverage; generate.py writes/checks Go data')

if __name__ == '__main__':
    main()
