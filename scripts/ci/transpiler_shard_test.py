import re
import subprocess
import unittest
from unittest import mock

import transpiler_shard as shard


class PartitionTests(unittest.TestCase):
    def test_all_tests_fuzz_and_examples_have_exactly_one_owner(self):
        names = [f"TestRegression{i}" for i in range(100)] + [
            "Test", "Testfuture", "Test_未来", "FuzzParser", "ExampleCompile",
        ]
        groups = shard.partition(names, 6)
        self.assertCountEqual([n for group in groups for n in group], names)
        for name in names:
            self.assertEqual(sum(name in group for group in groups), 1)
        self.assertEqual(groups, shard.partition(list(reversed(names)), 6))
        self.assertTrue(all(groups))
        patterns = [shard.test_pattern(group) for group in groups]
        for name in names:
            self.assertEqual(sum(bool(re.fullmatch(p, name)) for p in patterns), 1)

    def test_same_name_in_future_subpackages_stays_with_one_owner(self):
        names = ["TestShared", "TestShared", "TestNew"]
        groups = shard.partition(names, 6)
        self.assertCountEqual([n for group in groups for n in group], names)
        owners = [group for group in groups if "TestShared" in group]
        self.assertEqual(len(owners), 1)
        self.assertEqual(owners[0].count("TestShared"), 2)

    def test_exact_parent_filter_keeps_siblings_separate(self):
        pattern = shard.test_pattern(["TestCompiler", "ExampleCompile", "FuzzParser"])
        for name in ["TestCompiler", "ExampleCompile", "FuzzParser"]:
            self.assertIsNotNone(re.fullmatch(pattern, name))
        for name in ["TestCompilerMore", "OtherTestCompiler"]:
            self.assertIsNone(re.fullmatch(pattern, name))
        # Go applies this slash-free pattern to the parent, then runs all children.
        self.assertNotIn("/", pattern)

    def test_listing_covers_executable_kinds_and_excludes_benchmarks(self):
        output = "TestA\nFuzzB\nExampleC\nBenchmarkD\nok\texample/transpiler\t0.01s\n"
        self.assertEqual(shard.parse_listing(output), ["TestA", "FuzzB", "ExampleC"])
        with self.assertRaises(ValueError):
            shard.parse_listing("unexpected listing format\n")
        with self.assertRaises(ValueError):
            shard.parse_listing("?\texample/transpiler\t[no test files]\n")

    def test_empty_inventory_empty_shard_and_bad_count_fail(self):
        with self.assertRaises(ValueError):
            shard.partition([], 6)
        with self.assertRaises(ValueError):
            shard.partition(["TestOnly"], 0)
        with self.assertRaises(ValueError):
            shard.test_pattern([])
        with mock.patch.object(shard.subprocess, "run") as listing:
            listing.return_value.stdout = "TestOnly\nok\texample/transpiler\t0.01s\n"
            groups = shard.partition(["TestOnly"], 6)
            empty_index = next(i for i, group in enumerate(groups) if not group)
            with self.assertRaises(ValueError):
                shard.run_shard(empty_index, 6, "coverage.out")

    def test_listing_error_propagates_without_starting_test_run(self):
        failure = subprocess.CalledProcessError(1, ["go", "test"])
        with mock.patch.object(shard.subprocess, "run", side_effect=failure):
            with mock.patch.object(shard.subprocess, "call") as run:
                with self.assertRaises(subprocess.CalledProcessError):
                    shard.run_shard(0, 6, "coverage.out")
                run.assert_not_called()

    def test_runner_preserves_race_timeout_coverage_and_exit(self):
        names = [f"TestRegression{i}" for i in range(100)] + ["FuzzX", "ExampleY"]
        with mock.patch.object(shard.subprocess, "run") as listing:
            listing.return_value.stdout = "\n".join(names) + "\nok\texample/transpiler\t0.01s\n"
            with mock.patch.object(shard.subprocess, "call", return_value=7) as run:
                self.assertEqual(shard.run_shard(2, 6, "coverage-2.out"), 7)
            command = run.call_args.args[0]
            self.assertIn("-race", command)
            self.assertIn("20m", command)
            self.assertIn("-coverprofile=coverage-2.out", command)
            self.assertEqual(command[-1], "./transpiler/...")
            pattern = command[command.index("-run") + 1]
            self.assertEqual(
                [name for name in sorted(names) if re.fullmatch(pattern, name)],
                shard.partition(names, 6)[2],
            )
            listing_command = listing.call_args.args[0]
            self.assertIn("-race", listing_command)
            self.assertIn("-list", listing_command)
            self.assertTrue(listing.call_args.kwargs["check"])


if __name__ == "__main__":
    unittest.main()
