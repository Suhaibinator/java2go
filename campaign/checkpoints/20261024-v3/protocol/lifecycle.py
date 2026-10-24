"""Private exact-owned supervisor lifecycle; no compiler imports or execution."""
from contextlib import contextmanager
import json
import math
import os
from pathlib import Path
import signal
import subprocess
import time


def typed_receipt(raw):
    assert type(raw) is dict
    for name in ('pid', 'process_group', 'exit_code', 'runner_exit'):
        assert type(raw[name]) is int, (name, raw.get(name))
    assert raw['pid'] > 1 and raw['process_group'] == raw['pid']
    assert type(raw['timed_out']) is bool
    assert type(raw['duration_seconds']) in (int, float) and math.isfinite(raw['duration_seconds']) and raw['duration_seconds'] >= 0
    assert type(raw['timeout_seconds']) in (int, float) and math.isfinite(raw['timeout_seconds']) and raw['timeout_seconds'] > 0
    assert 'error' not in raw and 'interrupted_signal' not in raw
    assert type(raw['requested_argv']) is list and all(type(v) is str for v in raw['requested_argv'])
    assert type(raw['cwd']) is str and type(raw['env']) is dict
    cleanup = raw['cleanup']
    assert type(cleanup) is dict
    assert type(cleanup['leader_reaped']) is bool and type(cleanup['group_absent']) is bool
    assert type(cleanup['signals']) is list and all(type(v) is str for v in cleanup['signals'])
    for name in ('stdout', 'stderr'):
        record = raw['output'][name]
        assert type(record['bytes']) is int and 0 <= record['bytes'] <= 4 * 1024**2
        assert type(record['sha256']) is str and len(record['sha256']) == 64
        assert all(c in '0123456789abcdef' for c in record['sha256'])
    return raw


def identity(pid):
    result = subprocess.run(['/bin/ps', '-p', str(pid), '-o', 'ppid=,pgid=,lstart='],
                            capture_output=True, text=True, env={**os.environ, 'TZ': 'UTC'}, timeout=2)
    if result.returncode == 1 and not result.stdout.strip():
        return None
    assert result.returncode == 0 and not result.stderr.strip()
    fields = result.stdout.strip().split(maxsplit=2)
    assert len(fields) == 3
    return dict(pid=pid, ppid=int(fields[0]), pgid=int(fields[1]), starttime=fields[2])


class OwnedSupervisor:
    def __init__(self, command, env, raw_path, record):
        self.command, self.env, self.raw_path, self.record = command, env, Path(raw_path), record
        self.proc = None
        self.child_identity = None
        self.signals = []
        self.draining = False
        self.actions = []

    def learn_child(self):
        if not self.raw_path.exists():
            return
        raw = json.loads(self.raw_path.read_text())
        if 'pid' not in raw:
            return
        assert type(raw['pid']) is int and type(raw['process_group']) is int
        assert raw['pid'] > 1 and raw['pid'] == raw['process_group']
        if self.child_identity is None:
            current = identity(raw['pid'])
            if current is not None and current['ppid'] == self.proc.pid and current['pgid'] == current['pid']:
                self.child_identity = current

    def interrupt(self, signum, _frame):
        self.signals.append(signum)
        if self.draining:
            return  # Repeated signals cannot interrupt protected cleanup/reaping.
        self.draining = True
        if self.proc is not None and self.proc.poll() is None:
            try:
                self.proc.send_signal(signum)  # Exact unreaped Popen child only.
            except ProcessLookupError:
                pass
        raise InterruptedError('controller received signal ' + str(signum))

    def cleanup(self):
        self.draining = True
        errors = []
        if self.proc is None:
            return
        try:
            self.learn_child()
        except BaseException as exc:
            errors.append('learn-child: ' + repr(exc))
        if self.proc.poll() is None:
            try:
                self.proc.terminate()  # Pinned supervisor forwards through its own cleanup.
                self.actions.append('TERM-exact-supervisor')
                self.proc.wait(timeout=3)
            except BaseException as exc:
                errors.append('supervisor-drain: ' + repr(exc))
        # Fallback is restricted to a previously verified direct child leader
        # with unchanged PID/starttime/PGID. Never signal a group from raw alone.
        if self.child_identity is not None:
            try:
                current = identity(self.child_identity['pid'])
                if current is not None:
                    assert current['starttime'] == self.child_identity['starttime']
                    assert current['pgid'] == self.child_identity['pgid']
                    os.killpg(current['pgid'], signal.SIGTERM)
                    self.actions.append('TERM-verified-child-group')
                    until = time.monotonic() + 0.5
                    while time.monotonic() < until and identity(current['pid']) is not None:
                        time.sleep(0.02)
                    remaining = identity(current['pid'])
                    if remaining is not None:
                        assert remaining['starttime'] == current['starttime'] and remaining['pgid'] == current['pgid']
                        os.killpg(current['pgid'], signal.SIGKILL)
                        self.actions.append('KILL-verified-child-group')
            except ProcessLookupError:
                pass
            except BaseException as exc:
                errors.append('child-group-fallback: ' + repr(exc))
        if self.proc.poll() is None:
            try:
                self.proc.kill()  # Still exact, unreaped supervisor, not its process group.
                self.actions.append('KILL-exact-supervisor')
                self.proc.wait(timeout=2)
            except BaseException as exc:
                errors.append('supervisor-reap: ' + repr(exc))
        group_absent, child_absent = None, None
        try:
            raw = json.loads(self.raw_path.read_text())
            assert type(raw['pid']) is int and raw['pid'] == raw['process_group'] > 1
            try:
                os.kill(raw['pid'], 0)
                child_absent = False
            except ProcessLookupError:
                child_absent = True
            try:
                os.killpg(raw['process_group'], 0)
                group_absent = False
            except ProcessLookupError:
                group_absent = True
        except BaseException as exc:
            errors.append('independent-absence: ' + repr(exc))
        self.record(dict(child_absent=child_absent, group_absent=group_absent,
                         cleanup_state='VERIFIED_ABSENT' if self.proc.poll() is not None and child_absent is True and group_absent is True else 'UNVERIFIED',
                         supervisor_pid=self.proc.pid, supervisor_reaped=self.proc.poll() is not None,
                         verified_child_identity=self.child_identity, forwarded_or_drained_signals=self.signals,
                         actions=self.actions, errors=errors,
                         note='Process/group absence is separately required; no inferred clean PASS.'))

    @contextmanager
    def running(self, stdout, stderr):
        previous = {sig: signal.getsignal(sig) for sig in (signal.SIGTERM, signal.SIGINT)}
        for sig in previous:
            signal.signal(sig, self.interrupt)
        try:
            # During creation, retain signals without raising before the Popen
            # handle is assigned. No blocked signal mask leaks into the child.
            self.draining = True
            self.proc = subprocess.Popen(self.command, stdout=stdout, stderr=stderr, env=self.env)
            self.draining = False
            if self.signals:
                self.interrupt(self.signals[0], None)
            yield self
        finally:
            try:
                self.cleanup()
            finally:
                for sig, handler in previous.items():
                    signal.signal(sig, handler)
