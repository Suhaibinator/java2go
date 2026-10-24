#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
probe_jdk=/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home
probe_out="${1:-$(mktemp -d /private/tmp/string-join-iterable-oracle.XXXXXXXX)}"
if [[ -n "$(find "$probe_out" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]]; then
  printf 'Refusing nonempty oracle directory: %s\n' "$probe_out" >&2
  exit 2
fi
mkdir -p "$probe_out/classes"

# This manifest was recorded when the probe was prepared, before any Java execution.
sha256sum -c SOURCE_HASHES.sha256 > "$probe_out/inputs.before.check"
sha256sum SOURCE_HASHES.sha256 > "$probe_out/source-manifest.sha256"

printf '%s\n' 'python3 run-bounded.py 300 "$probe_out/javac.stdout" "$probe_out/javac.stderr" "$probe_jdk/bin/javac" --release 21 -encoding UTF-8 -d "$probe_out/classes" src/probe/join/state/Trace.java src/probe/join/source/TrackedText.java src/probe/join/source/LiveSource.java src/probe/join/app/Main.java' > "$probe_out/commands.txt"
if python3 run-bounded.py 300 "$probe_out/javac.stdout" "$probe_out/javac.stderr" \
  "$probe_jdk/bin/javac" --release 21 -encoding UTF-8 -d "$probe_out/classes" \
  src/probe/join/state/Trace.java \
  src/probe/join/source/TrackedText.java \
  src/probe/join/source/LiveSource.java \
  src/probe/join/app/Main.java; then
  printf 'javac.exit=0\n' > "$probe_out/status.txt"
else
  probe_status=$?
  printf 'javac.exit=%s\n' "$probe_status" > "$probe_out/status.txt"
  printf 'Java input did not compile; see %s/javac.stderr\n' "$probe_out" >&2
  exit "$probe_status"
fi

for probe_seed in 17 41 97; do
  for probe_repeat in 1 2 3; do
    printf 'python3 run-bounded.py 60 "$probe_out/seed-%s.repeat-%s.stdout" "$probe_out/seed-%s.repeat-%s.stderr" "$probe_jdk/bin/java" -cp "$probe_out/classes" probe.join.app.Main %s\n' \
      "$probe_seed" "$probe_repeat" "$probe_seed" "$probe_repeat" "$probe_seed" >> "$probe_out/commands.txt"
    if python3 run-bounded.py 60 \
      "$probe_out/seed-$probe_seed.repeat-$probe_repeat.stdout" \
      "$probe_out/seed-$probe_seed.repeat-$probe_repeat.stderr" \
      "$probe_jdk/bin/java" -cp "$probe_out/classes" probe.join.app.Main "$probe_seed"; then
      printf 'seed=%s repeat=%s exit=0\n' "$probe_seed" "$probe_repeat" >> "$probe_out/status.txt"
    else
      probe_status=$?
      printf 'seed=%s repeat=%s exit=%s\n' "$probe_seed" "$probe_repeat" "$probe_status" >> "$probe_out/status.txt"
      printf 'JVM run failed; see %s/status.txt\n' "$probe_out" >&2
      exit "$probe_status"
    fi
  done
  cmp "$probe_out/seed-$probe_seed.repeat-1.stdout" "$probe_out/seed-$probe_seed.repeat-2.stdout"
  cmp "$probe_out/seed-$probe_seed.repeat-1.stdout" "$probe_out/seed-$probe_seed.repeat-3.stdout"
  cmp "$probe_out/seed-$probe_seed.repeat-1.stderr" "$probe_out/seed-$probe_seed.repeat-2.stderr"
  cmp "$probe_out/seed-$probe_seed.repeat-1.stderr" "$probe_out/seed-$probe_seed.repeat-3.stderr"
  cp "$probe_out/seed-$probe_seed.repeat-1.stdout" "$probe_out/expected.seed-$probe_seed.stdout"
  cp "$probe_out/seed-$probe_seed.repeat-1.stderr" "$probe_out/expected.seed-$probe_seed.stderr"
  printf 'seed=%s repeats_byte_identical=true\n' "$probe_seed" >> "$probe_out/status.txt"
done

sha256sum -c SOURCE_HASHES.sha256 > "$probe_out/inputs.after.check"
sha256sum "$probe_out"/seed-*.stdout "$probe_out"/seed-*.stderr > "$probe_out/outputs.sha256"

python3 - "$probe_out" <<'PY'
from pathlib import Path
import hashlib, json, sys
out = Path(sys.argv[1])
def sha(path): return hashlib.sha256(path.read_bytes()).hexdigest()
seeds = {}
for seed in (17, 41, 97):
    runs = []
    for repeat in (1, 2, 3):
        runs.append({
            'repeat': repeat,
            'exit': 0,
            'stdout_sha256': sha(out / f'seed-{seed}.repeat-{repeat}.stdout'),
            'stderr_sha256': sha(out / f'seed-{seed}.repeat-{repeat}.stderr'),
        })
    seeds[str(seed)] = {
        'runs': runs,
        'expected_stdout': f'expected.seed-{seed}.stdout',
        'expected_stderr': f'expected.seed-{seed}.stderr',
    }
manifest = {
    'kind': 'supplemental-string-join-iterable-jvm-oracle',
    'jdk_home': '/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home',
    'compile_release': '21',
    'source_manifest_sha256': sha(Path('SOURCE_HASHES.sha256')),
    'input_hashes_match_prepared_before_and_after': True,
    'compile_exit': 0,
    'compile_stdout_sha256': sha(out / 'javac.stdout'),
    'compile_stderr_sha256': sha(out / 'javac.stderr'),
    'seeds': seeds,
}
(out / 'oracle.json').write_text(json.dumps(manifest, indent=2, sort_keys=True) + '\n')
PY
printf 'oracle_root=%s\n' "$probe_out"
