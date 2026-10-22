"""Grammar and declaration/closure proof controls; no Java application executes."""
import copy
import pathlib
import sys
import unittest
sys.dont_write_bytecode = True
import class_parent_producer as producer
Signature = producer.signature.Signature
SignatureError = producer.signature.SignatureError

def row(name, sig='', superclass='java/lang/Object', interfaces=(), outer=None, static=False, method=False):
    return dict(binary=name, module='java.base', classEntry='classes/'+name+'.class',
        class_sha256=('0'*63+str(len(name)%10)), super=superclass, interfaces=list(interfaces),
        signature=sig, access=33, preview=False, synthetic=False,
        ownInnerRows=[] if outer is None else [dict(inner=name, outer=outer, simple='Inner', access=8 if static else 0)], enclosingMethod=method)

def facts(*rows):
    result={'java/lang/Object':row('java/lang/Object',superclass=None)}
    result.update((r['binary'],r)for r in rows)
    return dict(records=result)

class GrammarTests(unittest.TestCase):
    def test_reordered_repeated_nested_and_array(self):
        parameters, parents=Signature('<A:Ljava/lang/Object;B:Ljava/lang/Object;>Lpkg/Face<TB;Ljava/util/List<TA;>;[TA;>;').class_signature()
        self.assertEqual(['A','B'],[p['name']for p in parameters])
        args=parents[0]['segments'][0]['arguments']
        self.assertEqual('B',args[0]['name']);self.assertEqual('A',args[1]['segments'][0]['arguments'][0]['name']);self.assertEqual('array',args[2]['kind'])
    def test_wildcard_and_real_owner_segments_retained(self):
        _, parents=Signature('Lpkg/Outer<Ljava/lang/String;>.Inner<+Ljava/lang/Number;>;Lpkg/Face<*>;').class_signature()
        self.assertEqual(['pkg/Outer','pkg/Outer$Inner'],[x['binary']for x in parents[0]['segments']])
        self.assertEqual('extends',parents[0]['segments'][1]['arguments'][0]['variance'])
        self.assertEqual('unbounded',parents[1]['segments'][0]['arguments'][0]['variance'])
    def test_malformed_truncated_ambiguous_grammar_refused(self):
        for source in ['', '<>', 'TT;', 'Ljava/lang/Object;;','<T:Ljava/lang/Object;T:Ljava/lang/Object;>Ljava/lang/Object;', 'Lpkg/Bad<>;', 'Lpkg//Bad;', 'Lpkg/Bad<+>;', 'Lpkg/Bad<[V>;', 'Lpkg/Bad<Tmissing>;', 'Lpkg/Bad<'+('['*256)+'I>;']:
            with self.subTest(source=source),self.assertRaises(SignatureError):Signature(source).class_signature()

class DeclarationTests(unittest.TestCase):
    def test_exact_own_tokens_not_spelling_or_position(self):
        f=facts(row('pkg/Face','<A:Ljava/lang/Object;B:Ljava/lang/Object;>Ljava/lang/Object;'),row('pkg/Child','<A:Ljava/lang/Object;B:Ljava/lang/Object;>Ljava/lang/Object;Lpkg/Face<TB;TA;>;',interfaces=['pkg/Face']))
        rows=producer.bind_records(f);args=rows['pkg/Child']['javaParents'][1]['segments'][0]['arguments']
        self.assertEqual([1,0],[x['declaration']['slot']for x in args])
        self.assertTrue(all(x['declaration']['owner']=='pkg.Child'for x in args))
        self.assertNotEqual(rows['pkg/Face']['parameters'][0]['owner'],args[1]['declaration']['owner'])
    def test_signature_header_disagreement_is_hard_error(self):
        with self.assertRaises(ValueError):producer.bind_records(facts(row('pkg/Bad','Ljava/lang/Object;Lpkg/Other;')))
    def test_arity_and_missing_metadata_held(self):
        f=facts(row('pkg/Generic','<A:Ljava/lang/Object;B:Ljava/lang/Object;>Ljava/lang/Object;'),row('pkg/Bad','Ljava/lang/Object;Lpkg/Generic<Ljava/lang/Object;>;',interfaces=['pkg/Generic']))
        rows=producer.bind_records(f);emitted,held=producer.close_records(rows,{'pkg/Bad'})
        self.assertNotIn('pkg/Bad',emitted);self.assertIn('argument arity mismatch',str(held['pkg/Bad']))
        f=facts(row('pkg/Missing',interfaces=['pkg/Unknown']));rows=producer.bind_records(f);emitted,held=producer.close_records(rows,{'pkg/Missing'})
        self.assertNotIn('pkg/Missing',emitted);self.assertTrue(held['pkg/Missing'])
    def test_nonstatic_captured_outer_and_static_boundary_refused(self):
        outer=row('pkg/Outer','<T:Ljava/lang/Object;>Ljava/lang/Object;');face=row('pkg/Face','<A:Ljava/lang/Object;>Ljava/lang/Object;')
        for static,method,expected in [(False,False,'captured outer'),(True,False,'unresolved variable'),(False,True,'executable lexical binder')]:
            inner=row('pkg/Outer$Inner','Ljava/lang/Object;Lpkg/Face<TT;>;',interfaces=['pkg/Face'],outer='pkg/Outer',static=static,method=method)
            rows=producer.bind_records(facts(outer,face,inner));emitted,held=producer.close_records(rows,{'pkg/Outer$Inner'})
            self.assertNotIn('pkg/Outer$Inner',emitted);self.assertIn(expected,str(held['pkg/Outer$Inner']))
    def test_nongeneric_concrete_parent_not_raw(self):
        rows=producer.bind_records(facts(row('pkg/Face','<T:Ljava/lang/Object;>Ljava/lang/Object;'),row('pkg/Plain','Ljava/lang/Object;Lpkg/Face<Lpkg/Plain;>;',interfaces=['pkg/Face'])))
        emitted,held=producer.close_records(rows,{'pkg/Plain'});self.assertIn('pkg/Plain',emitted)
        args=rows['pkg/Plain']['javaParents'][1]['segments'][0];self.assertFalse(args['raw']);self.assertEqual('pkg/Plain',args['arguments'][0]['binary']);self.assertFalse(held)
    def test_unsupported_edges_not_dropped(self):
        face=row('pkg/Face','<T:Ljava/lang/Object;>Ljava/lang/Object;')
        for argument,reason in [('*','wildcard'),('[I','primitive array')]:
            child=row('pkg/Child','Ljava/lang/Object;Lpkg/Face<'+argument+'>;',interfaces=['pkg/Face'])
            rows=producer.bind_records(facts(face,child));emitted,held=producer.close_records(rows,{'pkg/Child'})
            self.assertNotIn('pkg/Child',emitted);self.assertIn(reason,str(held['pkg/Child']))
    def test_inheritance_cycle_refused_but_argument_self_reference_allowed(self):
        rows=producer.bind_records(facts(row('pkg/A',superclass='pkg/B'),row('pkg/B',superclass='pkg/A')))
        emitted,held=producer.close_records(rows,{'pkg/A'});self.assertFalse(emitted);self.assertIn('cyclic',str(held['pkg/A']))
    def test_diamond_and_incomplete_sibling_whole_closure(self):
        f=facts(row('pkg/Base'),row('pkg/Left',interfaces=['pkg/Base']),row('pkg/Right',interfaces=['pkg/Base']),row('pkg/Child',interfaces=['pkg/Left','pkg/Right']))
        rows=producer.bind_records(f);emitted,held=producer.close_records(rows,{'pkg/Child'});self.assertIn('pkg/Child',emitted)
        f['records']['pkg/Right']['interfaces'].append('pkg/Missing')
        rows=producer.bind_records(f);emitted,held=producer.close_records(rows,{'pkg/Child'});self.assertNotIn('pkg/Child',emitted);self.assertIn('missing/cyclic parent hierarchy',str(held['pkg/Child']))

class PinnedCorpusTests(unittest.TestCase):
    def test_all_pinned_old_eight_and_emitted_closures(self):
        _,rows,coverage=producer.prepare();self.assertEqual(8,coverage['old_eight_exact']);self.assertGreater(len(rows),1000)
        self.assertIn('java/util/HashMap',rows);self.assertIn('java/util/Map',rows);self.assertIn('java/lang/StringBuilder',rows)
        for name,row in rows.items():
            reasons,references=producer.representable(row)
            self.assertFalse(reasons,name);self.assertFalse(references-rows.keys(),name)
        self.assertIn('wildcard parent recipe',str(coverage['held']['java/lang/Class']))

if __name__=='__main__':unittest.main()
