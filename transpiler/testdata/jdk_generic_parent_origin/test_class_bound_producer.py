import importlib.util
import pathlib
import sys
import unittest

sys.dont_write_bytecode = True
HERE = pathlib.Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('bound_producer', HERE / 'class_bound_producer.py')
producer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(producer)


class BoundProducerTest(unittest.TestCase):
    def test_absent_class_bound_is_not_object(self):
        p = producer.formal_bounds('<D::Ljava/lang/reflect/GenericDeclaration;>Ljava/lang/Object;')[0]
        self.assertIsNone(p['classBound'])
        self.assertEqual(p['interfaceBounds'][0]['binary'], 'java/lang/reflect/GenericDeclaration')
        q = producer.formal_bounds('<D:Ljava/lang/Object;>Ljava/lang/Object;')[0]
        self.assertEqual(q['classBound']['binary'], 'java/lang/Object')
        self.assertEqual(q['interfaceBounds'], [])

    def test_order_and_forward_dependent_slot(self):
        p = producer.formal_bounds('<B:TA;A:Ljava/lang/Object;>Ljava/lang/Object;')
        self.assertEqual([x['name'] for x in p], ['B', 'A'])
        self.assertEqual(p[0]['classBound'], {'kind': 'variable', 'name': 'A'})

    def test_intersection_preserved(self):
        p = producer.formal_bounds('<D:Ljava/lang/Number;:Ljava/io/Serializable;:Ljava/lang/Comparable<TD;>;>Ljava/lang/Object;')[0]
        self.assertEqual(p['classBound']['binary'], 'java/lang/Number')
        self.assertEqual([b['binary'] for b in p['interfaceBounds']], ['java/io/Serializable', 'java/lang/Comparable'])

    def test_malformed_duplicate_and_missing(self):
        for value in ('<D:>Ljava/lang/Object;', '<D:Ljava/lang/Object;D:Ljava/lang/Object;>Ljava/lang/Object;', '<D::Ljava/lang/Object;'):
            with self.subTest(value=value), self.assertRaises(ValueError):
                producer.formal_bounds(value)

    def test_held_nodes_keep_the_recipe(self):
        for value in ('[I', 'Louter/Owner<TT;>.Inner;', 'TT;'):
            node = producer.signature.Signature(value).reference()
            self.assertFalse(producer.lowerable(node))
            self.assertTrue(node)


if __name__ == '__main__':
    unittest.main()
