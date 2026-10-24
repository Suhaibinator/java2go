"""Run an exhaustive, deterministic shard of Go's compiler test inventory."""

import argparse
from collections import Counter
import hashlib
import re
import subprocess
import sys


def parse_listing(output):
    names = []
    for line in output.splitlines():
        if re.fullmatch(r"(?:Test|Fuzz|Example)\w*", line):
            names.append(line)
        elif re.fullmatch(r"Benchmark\w*", line) or re.match(r"^(?:ok|\?)\s", line):
            continue  # Benchmarks were never executed by the unit test command.
        elif line.strip():
            raise ValueError(f"unexpected go test listing: {line!r}")
    if not names:
        raise ValueError("compiler test inventory is empty")
    return names


def partition(names, count):
    if not names or count < 1:
        raise ValueError("nonempty inventory and positive shard count required")
    groups = [[] for _ in range(count)]
    for name in sorted(names):
        owner = int.from_bytes(hashlib.sha256(name.encode()).digest(), "big") % count
        groups[owner].append(name)
    # Repeated names in future subpackages share an owner and still run there.
    if Counter(name for group in groups for name in group) != Counter(names):
        raise ValueError("compiler partition lost or duplicated an inventory entry")
    if any(set(a) & set(b) for i, a in enumerate(groups) for b in groups[i + 1:]):
        raise ValueError("compiler partition assigns a name to multiple shards")
    return groups


def test_pattern(names):
    if not names:
        raise ValueError("requested compiler shard is empty")
    # A slash-free, anchored parent selector runs every subtest of each owner.
    return "^(" + "|".join(re.escape(name) for name in sorted(set(names))) + ")$"


def run_shard(index, count, coverprofile):
    if count < 1 or not 0 <= index < count:
        raise ValueError("shard index must be within the positive shard count")
    listing = subprocess.run(
        ["go", "test", "-race", "-timeout", "20m", "-list", ".", "./transpiler/..."],
        check=True, stdout=subprocess.PIPE, text=True,
    )
    names = parse_listing(listing.stdout)
    groups = partition(names, count)
    pattern = test_pattern(groups[index])
    digest = hashlib.sha256(("\n".join(sorted(names)) + "\n").encode()).hexdigest()
    print(f"Compiler inventory: {len(names)} entries; sha256={digest}", flush=True)
    print(f"Shard counts: {[len(group) for group in groups]}; selected={index}", flush=True)
    return subprocess.call([
        "go", "test", "-v", "-race", "-timeout", "20m",
        f"-coverprofile={coverprofile}", "-run", pattern, "./transpiler/...",
    ])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--index", type=int, required=True)
    parser.add_argument("--count", type=int, required=True)
    parser.add_argument("--coverprofile", required=True)
    args = parser.parse_args()
    return run_shard(args.index, args.count, args.coverprofile)


if __name__ == "__main__":
    sys.exit(main())
