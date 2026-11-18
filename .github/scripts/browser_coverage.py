"""Same-source native Node evidence, with no invented cross-platform LCOV IDs.

GitHub workflow/job/artifact receipts authenticate expected pins. Their SHA only
binds bytes. Node reporter array indices are not cross-run branch identities;
this contract retains sanitized V8 source-offset ranges from the SAME test call.
The baseline is explicitly an owned current-Sonar measurement, not a provider
artifact; only three native originals have provider archive IDs. Original LCOV is immutable. The distinct canonical composite preserves all raw
range/line/function coordinates, including zeros, and only maxes exact counters.
The Sonar LCOV projection contains only block-precise conditions, never coarse
function-only observations. Every original reporter record must exactly match
the pinned Node24 merge/summary semantics before projection; int32 overflow is
refused, not silently dropped. Dynamic V8 function enumeration and real server
import/readback remain qualification limitations, not whole-product proof.
It never fills a missing range from an enclosing hit, excludes production code,
changes the 100% gate, or claims an original workflow/product/native pass.

begin/retain surround the EXISTING Node24 web suite invocation. They do not run
a second suite or acquire artifacts. The existing CI step/job owns process and
cancellation lifetime. Probe hashes are producer assertions, not retained binary
or publisher/atomic hash-to-exec attestation. Raw V8 URLs, source maps and all non-first-party records are never uploaded.
Original first-party LCOV keeps its reviewed source/fixture function labels;
computed V8 names are not copied into the canonical offset manifest. Only three fixed output
members are eligible: lcov.info, counters.json, metadata.json.
"""
from __future__ import annotations

import argparse
import bisect
import copy
import hashlib
import io
import json
import os
from pathlib import Path
import re
import signal
import stat
import subprocess
import sys
import threading
import time
import urllib.parse
import zipfile
import zlib

REPOSITORY = 'maldous/CircularQuay'
NODE_VERSION = 'v24.22.2'
PLATFORMS = {'linux': 'linux', 'macos': 'darwin', 'windows': 'win32'}
MAX_PROFILE = 16 << 20
MAX_METADATA = 4 << 20
MAX_ARCHIVE = 40 << 20
MAX_RAW = 64 << 20
MAX_SOURCE = 4 << 20
MAX_TOOL = 256 << 20
MAX_INT = (1 << 53) - 1
SONAR_MAX_HIT = (1 << 31) - 1
PROBE_DEADLINE = 20
MAX_PROBE_ERROR = 65536
REFUSAL = 'Invalid bounded browser coverage evidence.'
MEMBERS = {'lcov.info': MAX_PROFILE, 'counters.json': MAX_PROFILE, 'metadata.json': MAX_METADATA}


class InvalidEvidence(ValueError):
    """Fixed evidence cannot be authenticated as the requested invocation."""


def require(value):
    if not value:
        raise InvalidEvidence(REFUSAL)


def keys(value, expected):
    require(type(value) is dict and set(value) == set(expected))


def integer(value, lower=0, upper=MAX_INT):
    require(type(value) is int and lower <= value <= upper)
    return value


def hexadecimal(value, length=64):
    require(type(value) is str and re.fullmatch('[a-f0-9]{' + str(length) + '}', value) is not None)
    return value


def digest(value):
    return hashlib.sha256(value).hexdigest()


def encoded(value):
    return (json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=True,
                       allow_nan=False) + '\n').encode('utf-8')


def unique(pairs):
    output = {}
    for key, value in pairs:
        require(key not in output)
        output[key] = value
    return output


def invalid_constant(_value):
    raise InvalidEvidence(REFUSAL)


def strict_json(data):
    try:
        return json.loads(data.decode('utf-8'), object_pairs_hook=unique,
                          parse_constant=invalid_constant)
    except (UnicodeError, ValueError, OverflowError, RecursionError):
        raise InvalidEvidence(REFUSAL) from None


def identity(info):
    require(info.st_ino > 0)
    return (info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def private_read(path, limit):
    path = Path(path)
    before = path.lstat()
    require(stat.S_ISREG(before.st_mode) and not path.is_symlink()
            and before.st_nlink == 1 and 0 <= before.st_size <= limit)
    flags = os.O_RDONLY | getattr(os, 'O_BINARY', 0) | getattr(os, 'O_NOFOLLOW', 0)
    descriptor = os.open(path, flags)
    try:
        opened = os.fstat(descriptor)
        require(identity(before) == identity(opened))
        chunks = []
        total = 0
        while True:
            part = os.read(descriptor, min(65536, limit + 1 - total))
            if not part:
                break
            total += len(part)
            require(total <= limit)
            chunks.append(part)
        require(identity(opened) == identity(os.fstat(descriptor)) == identity(path.lstat()))
        require(total == opened.st_size)
        return b''.join(chunks)
    finally:
        os.close(descriptor)


def exclusive_write(path, data):
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, 'O_BINARY', 0), 0o600)
    try:
        with os.fdopen(descriptor, 'wb', closefd=False) as output:
            output.write(data)
            output.flush()
            os.fsync(descriptor)
    finally:
        os.close(descriptor)


def capture_stream(stream, limit, state, overflow):
    # Read at most limit+1 bytes, never communicate() an unbounded pipe. The
    # extra byte is an overflow sentinel, not retained or returned as success.
    try:
        while state['bytes'] <= limit:
            part = stream.read(min(65536, limit + 1 - state['bytes']))
            if not part:
                state['eof'] = True
                return
            state['bytes'] += len(part)
            if state['bytes'] > limit:
                overflow.set()
                return
            state['parts'].append(part)
    except OSError:
        state['error'] = True
        overflow.set()


def stop_probe(process):
    # These are fixed Git/Node identity probes, NOT test-suite launchers. Own
    # a POSIX session / Windows process group so refusal can stop the probe.
    if os.name == 'posix':
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
    elif process.poll() is None:
        process.kill()
    try:
        process.wait(timeout=3)
    except subprocess.TimeoutExpired:
        raise InvalidEvidence(REFUSAL) from None


def probe_completed(process, threads):
    return process.poll() is not None and all(not thread.is_alive() for thread in threads)


def wait_for_probe(process, threads, states, overflow, deadline):
    end = time.monotonic() + deadline
    while True:
        require(not overflow.is_set() and time.monotonic() < end)
        if probe_completed(process, threads):
            break
        time.sleep(0.005)
    require(process.returncode == 0 and all(state['eof'] and not state['error'] for state in states)
            and states[1]['bytes'] == 0)
    return b''.join(states[0]['parts'])


def finish_probe(process, threads, streams):
    stop_probe(process)
    for thread in threads:
        if thread.ident is not None:
            thread.join(timeout=3)
    complete = all(not thread.is_alive() for thread in threads)
    for stream in streams:
        stream.close()
    require(complete)


def bounded_probe(argv, cwd, environment, stdout_limit, deadline=PROBE_DEADLINE):
    integer(stdout_limit, 1, MAX_RAW)
    require(type(deadline) in {int, float} and 0 < deadline <= PROBE_DEADLINE)
    options = {'start_new_session': True} if os.name == 'posix' else {
        'creationflags': subprocess.CREATE_NEW_PROCESS_GROUP}
    process = subprocess.Popen(argv, cwd=cwd, env=environment, stdin=subprocess.DEVNULL,
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE, bufsize=0, **options)
    overflow = threading.Event()
    states = [{'bytes': 0, 'parts': [], 'eof': False, 'error': False} for _ in range(2)]
    streams = (process.stdout, process.stderr)
    threads = [threading.Thread(target=capture_stream, args=(stream, limit, state, overflow), daemon=True)
               for stream, limit, state in zip(streams, (stdout_limit, MAX_PROBE_ERROR), states)]
    try:
        for thread in threads:
            thread.start()
        return wait_for_probe(process, threads, states, overflow, deadline)
    finally:
        # Reap on success or refusal; no partial output or diagnostics escape.
        finish_probe(process, threads, streams)

def command(argv, cwd, limit=MAX_METADATA):
    environment = dict(os.environ)
    environment.update(GIT_OPTIONAL_LOCKS='0', GIT_TERMINAL_PROMPT='0',
                       GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL=os.devnull)
    environment.pop('NODE_OPTIONS', None)
    environment.pop('NODE_V8_COVERAGE', None)
    return bounded_probe(argv, cwd, environment, limit)


def git(root, *arguments):
    return command(['git', '-c', 'core.fsmonitor=false', '-C', str(root), *arguments], root, MAX_RAW)


def eligible(path):
    return (type(path) is str and path.endswith(('.js', '.mjs'))
            and not path.endswith(('.test.js', '.test.mjs'))
            and path != 'examples/landing-motion/.local/worker-routes.mjs')


def source_geometry(text):
    require('node:coverage' not in text)  # no silently ignored source lines
    starts, ends, offset = [], [], 0
    # Node24 splits ONLY on LF, not all Unicode line separators.
    pieces = re.findall(r'[^\n]*\n|[^\n]+$', text)
    for line in pieces:
        starts.append(offset)
        units = len(line.encode('utf-16-le')) // 2
        newline = 2 if line.endswith('\r\n') else int(line.endswith('\n'))
        ends.append(offset + units - newline)
        offset += units
    require(starts)
    return starts, ends, offset


def inventory(root):
    output = {}
    for entry in git(root, 'ls-tree', '-r', '-z', 'HEAD').split(b'\0'):
        if not entry:
            continue
        mode, kind, rest = entry.split(b' ', 2)
        blob, raw_path = rest.split(b'\t', 1)
        path = raw_path.decode('utf-8')
        if not eligible(path):
            continue
        require(mode in {b'100644', b'100755'} and kind == b'blob')
        data = private_read(root / path, MAX_SOURCE)
        require(hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest().encode() == blob)
        text = data.decode('utf-8')
        starts, ends, offset = source_geometry(text)
        output[path] = {'sha256': digest(data), 'bytes': len(data), 'units': offset,
                        'starts': starts, 'ends': ends}
    require(output and len(output) <= 4096)
    return output


def public_inventory(files):
    return {path: {key: row[key] for key in ('sha256', 'bytes', 'units')}
            for path, row in sorted(files.items())}


def source_identity(root, expected):
    hexadecimal(expected, 40)
    require(root.is_dir() and not root.is_symlink())
    commit = git(root, 'rev-parse', 'HEAD').decode().strip()
    tree = git(root, 'rev-parse', 'HEAD^{tree}').decode().strip()
    hexadecimal(tree, 40)
    require(commit == expected and git(root, 'status', '--porcelain=v1', '--untracked-files=no') == b'')
    files = inventory(root)
    value = {'commit': commit, 'tree': tree, 'inventory_sha256': digest(encoded(public_inventory(files)))}
    for field, name in [('package_sha256', 'web/package.json'),
                        ('implementation_sha256', '.github/scripts/browser_coverage.py')]:
        data = private_read(root / name, MAX_SOURCE)
        blob = git(root, 'rev-parse', 'HEAD:' + name).decode().strip()
        require(hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest() == blob)
        value[field] = digest(data)
    return value, files


def node_identity(root, node, platform):
    require(platform in PLATFORMS)
    selected = Path(node)
    require(selected.is_absolute())
    probe = strict_json(command([str(selected), '-p', 'JSON.stringify({version:process.version,platform:process.platform,arch:process.arch,execPath:process.execPath})'], root))
    keys(probe, ('version', 'platform', 'arch', 'execPath'))
    require(probe['version'] == NODE_VERSION and probe['platform'] == PLATFORMS[platform]
            and probe['arch'] in {'x64', 'arm64'})
    require(Path(probe['execPath']).resolve() == selected.resolve())
    data = private_read(selected.resolve(), MAX_TOOL)
    return {'version': probe['version'], 'platform': probe['platform'], 'arch': probe['arch'],
            'sha256': digest(data), 'bytes': len(data)}


def producer_identity(environment, platform, suite):
    require(environment.get('GITHUB_REPOSITORY') == REPOSITORY and platform in PLATFORMS
            and suite in {'web', 'full-browser'})
    values = {key: environment.get(name, '') for key, name in
              [('run', 'GITHUB_RUN_ID'), ('attempt', 'GITHUB_RUN_ATTEMPT'), ('job', 'GITHUB_JOB'),
               ('event', 'GITHUB_EVENT_NAME'), ('head', 'CQ_PR_HEAD_SHA')]}
    for key in ('run', 'attempt'):
        require(re.fullmatch('[1-9][0-9]{0,19}', values[key]) is not None)
    require(re.fullmatch('[A-Za-z0-9_-]{1,80}', values['job']) is not None)
    require(values['event'] in {'push', 'pull_request'})
    hexadecimal(values['head'], 40)
    return {'repository': REPOSITORY, **values, 'platform': platform, 'suite': suite,
            'node': NODE_VERSION}


def fixed_output(output, platform, suite, environment):
    require(platform in PLATFORMS and suite in {'web', 'full-browser'})
    temporary = Path(environment.get('RUNNER_TEMP', ''))
    require(temporary.is_absolute() and temporary.is_dir() and not temporary.is_symlink())
    require(output == temporary / ('cq-browser-' + suite + '-' + platform))


def begin(root, output, expected, node, platform, suite, environment):
    require(not os.environ.get('NODE_OPTIONS'))
    fixed_output(output, platform, suite, environment)
    source, files = source_identity(root, expected)
    tool = node_identity(root, node, platform)
    producer = producer_identity(environment, platform, suite)
    require(not output.exists())
    output.mkdir(mode=0o700)
    (output / 'raw').mkdir(mode=0o700)
    value = {'schema': 'cq.native-browser.begin/1', 'source': source, 'node': tool,
             'producer': producer, 'files': public_inventory(files)}
    exclusive_write(output / 'begin.json', encoded(value))
    return value


def source_path(value, root, files):
    require(type(value) is str)
    parsed = urllib.parse.urlsplit(value)
    if parsed.scheme != 'file' or parsed.netloc not in {'', 'localhost'}:
        return None
    try:
        require(re.search(r'%(?![A-Fa-f0-9]{2})', parsed.path) is None)
        decoded = urllib.parse.unquote(parsed.path, errors='strict')
        if os.name == 'nt' and re.match(r'^/[A-Za-z]:/', decoded):
            decoded = decoded[1:]
        path = Path(decoded)
        relative = path.relative_to(root).as_posix()
    except (ValueError, UnicodeError):
        return None
    return relative if relative in files else None


def checked_ranges(ranges, maximum):
    require(type(ranges) is list and 0 < len(ranges) <= 100000)
    output = []
    stack = []
    seen = set()
    previous = -1
    for row in ranges:
        keys(row, ('startOffset', 'endOffset', 'count'))
        start = integer(row['startOffset'], 0, maximum)
        end = integer(row['endOffset'], start + 1, maximum)
        count = integer(row['count'])
        require(start >= previous and (start, end) not in seen)
        while stack and start >= stack[-1]:
            stack.pop()
        if stack:
            require(end <= stack[-1])
        stack.append(end)
        previous = start
        seen.add((start, end))
        output.append([start, end, count])
    return output


def sanitized_function(function, maximum, index):
    require(type(function) is dict and type(function.get('isBlockCoverage')) is bool
            and type(function.get('functionName')) is str)
    name = function['functionName']
    require(len(name) <= 65536)
    return {'kind': 'script' if index == 0 else 'function',
            'name_sha256': digest(name.encode('utf-8')), 'anonymous': name == '',
            'block': function['isBlockCoverage'],
            'ranges': checked_ranges(function.get('ranges'), maximum)}


def append_range_identity(rows, value):
    if not any(value is row for row in rows):
        rows.append(value)


def zero_intersection(previous, incoming):
    if previous[2] != 0 or incoming[2] != 0:
        return None
    if previous[0] <= incoming[0] and previous[1] >= incoming[1]:
        return incoming
    if incoming[0] <= previous[0] and incoming[1] >= previous[1]:
        return previous
    return None


def merge_node_range(old, incoming, merged):
    for previous in old:
        if incoming[:2] == previous[:2]:
            previous[2] = integer(previous[2] + incoming[2])
            append_range_identity(merged, previous)
            return
        intersection = zero_intersection(previous, incoming)
        if intersection is not None:
            append_range_identity(merged, intersection)
    if incoming[2] > 0:
        append_range_identity(merged, incoming)


def merge_node_ranges(old, new):
    # Native within-original merge: exact counters SUM, zero intersections and
    # object insertion order follow pinned Node24. Cross-platform union is MAX.
    merged = [row for row in old if row[2] > 0]
    for incoming in new:
        merge_node_range(old, incoming, merged)
    return merged

def function_identity(function):
    first = function['ranges'][0]
    return function['name_sha256'], first[0], first[1]


def update_node_function(existing, incoming):
    if not incoming['block']:
        return
    if existing['block']:
        existing['ranges'] = merge_node_ranges(existing['ranges'], copy.deepcopy(incoming['ranges']))
    else:
        existing['ranges'] = copy.deepcopy(incoming['ranges'])
    existing['block'] = True


def merge_node_functions(old, new):
    for incoming in new:
        existing = next((row for row in old if function_identity(row) == function_identity(incoming)), None)
        if existing is None:
            old.append(copy.deepcopy(incoming))
        else:
            update_node_function(existing, incoming)
    return old

def original_function_ranges(function, maximum):
    original = function['ranges']
    checked = checked_ranges([{'startOffset': a, 'endOffset': b, 'count': count}
                              for a, b, count in original], maximum)
    require(checked == original)


def retain_raw_function(rows, function):
    first = function['ranges'][0]
    key = (function['kind'], function['name_sha256'], first[0], first[1], function['block'])
    spans = rows.setdefault(key, {})
    for start, end, count in function['ranges']:
        spans[(start, end)] = max(spans.get((start, end), 0), count)


def aggregate_native_script(scripts, raw, script, files):
    name, functions = script['path'], script['functions']
    for function in functions:
        original_function_ranges(function, files[name]['units'])
    current = scripts.setdefault((name, script['instance']), [])
    merge_node_functions(current, functions)
    rows = raw.setdefault(name, {})
    for function in functions:
        retain_raw_function(rows, function)


def aggregate_observations(observations, files):
    # One authoritative bounded leaf feeds BOTH native reporter replay and the
    # exact-offset raw union. Neither independently mutable view can add a hit.
    require(type(observations) is list and 0 < len(observations) <= 4096)
    scripts, raw = {}, {}
    for report in observations:
        validated_scripts(report, files)
        for script in report:
            aggregate_native_script(scripts, raw, script, files)
    require(scripts)
    return {'scripts': [{'path': name, 'instance': instance, 'functions': functions}
                        for (name, instance), functions in scripts.items()],
            'raw': raw, 'observations': observations}

def sanitized_native_script(script, root, files):
    require(type(script) is dict and type(script.get('url')) is str)
    name = source_path(script['url'], root, files)
    if name is None:
        return None
    require(type(script.get('functions')) is list and 0 < len(script['functions']) <= 100000)
    # Never retain operational URLs, queries or computed name plaintext.
    return {'path': name, 'instance': digest(script['url'].encode('utf-8')),
            'functions': [sanitized_function(row, files[name]['units'], index)
                          for index, row in enumerate(script['functions'])]}


def read_native_report(data, root, files):
    value = strict_json(data)
    require(type(value) is dict and set(value) <= {'result', 'timestamp', 'source-map-cache'}
            and type(value.get('result')) is list)
    report = []
    for script in value['result']:
        selected = sanitized_native_script(script, root, files)
        if selected is not None:
            report.append(selected)
    return report


def raw_counters(directory, root, files):
    observations = []
    total = 0
    # Native opendir order is retained, never assumed equivalent to Node's
    # original internal directory: exact original-LCOV correspondence is required.
    entries = list(directory.iterdir())
    require(0 < len(entries) <= 4096)
    for path in entries:
        require(re.fullmatch(r'coverage-[0-9]+-[0-9]{13}-[0-9]+\.json', path.name) is not None)
        data = private_read(path, MAX_RAW)
        total += len(data)
        require(total <= MAX_RAW)
        report = read_native_report(data, root, files)
        if report:
            observations.append(report)
    return aggregate_observations(observations, files)

def relative_lcov_path(value):
    require(type(value) is str and '\0' not in value)
    value = value.replace('\\', '/')
    require(value and not value.startswith('/') and ':' not in value)
    parts = []
    for part in value.split('/'):
        require(part != '')
        if part == '..':
            require(parts)
            parts.pop()
        elif part != '.':
            parts.append(part)
    require(parts)
    return '/'.join(parts)


def parse_lcov_line(line, current, records, files, prefix):
    require(len(line) <= 16384)
    if line.startswith('TN:'):
        require(current is None and line == 'TN:')
    elif line.startswith('SF:'):
        require(current is None)
        name = relative_lcov_path(prefix + line[3:])
        require(name in files)
        current = {'path': name, 'lines': {}, 'fn': [], 'fnda': [], 'branches': {}, 'summary': {}}
    elif line == 'end_of_record':
        require(current is not None)
        validate_lcov_record(current)
        records.append(current)
        current = None
    else:
        require(current is not None and ':' in line)
        kind, value = line.split(':', 1)
        parse_lcov_row(current, kind, value, files[current['path']])
    return current


def parse_lcov(data, files, prefix=''):
    require(0 < len(data) <= MAX_PROFILE)
    try:
        text = data.decode('utf-8')
    except UnicodeError:
        raise InvalidEvidence(REFUSAL) from None
    records = []
    current = None
    for line in text.splitlines():
        current = parse_lcov_line(line, current, records, files, prefix)
    require(current is None and records and len(records) <= 4096)
    return records

def decimal(text):
    require(type(text) is str and re.fullmatch('[0-9]{1,16}', text) is not None)
    return integer(int(text))


def parse_lcov_line_counter(row, value, source):
    values = value.split(',')
    require(len(values) == 2)
    number, count = map(decimal, values)
    require(0 < number <= len(source['starts']) and number not in row['lines'])
    row['lines'][number] = count


def parse_lcov_function(row, value, source):
    number, name = value.split(',', 1)
    require(name and '\r' not in name)
    row['fn'].append((integer(decimal(number), 1, len(source['starts'])), name))


def parse_lcov_branch(row, value, source):
    values = value.split(',')
    require(len(values) == 4)
    number, block, branch = map(decimal, values[:3])
    require(0 < number <= len(source['starts']) and (number, block, branch) not in row['branches'])
    row['branches'][(number, block, branch)] = 0 if values[3] == '-' else decimal(values[3])


def parse_lcov_function_count(row, value, _source):
    count, name = value.split(',', 1)
    row['fnda'].append((decimal(count), name))


def parse_lcov_row(row, kind, value, source):
    handler = {'DA': parse_lcov_line_counter, 'FN': parse_lcov_function,
               'FNDA': parse_lcov_function_count, 'BRDA': parse_lcov_branch}.get(kind)
    if handler is not None:
        handler(row, value, source)
    else:
        require(kind in {'FNF', 'FNH', 'BRF', 'BRH', 'LF', 'LH'} and kind not in row['summary'])
        row['summary'][kind] = decimal(value)


def validate_lcov_record(row):
    summary = row['summary']
    keys(summary, ('FNF', 'FNH', 'BRF', 'BRH', 'LF', 'LH'))
    require(len(row['fn']) == len(row['fnda']) == summary['FNF'])
    require([name for _, name in row['fn']] == [name for _, name in row['fnda']])
    require(sum(count > 0 for count, _ in row['fnda']) == summary['FNH'])
    require(len(row['branches']) == summary['BRF'])
    require(sum(count > 0 for count in row['branches'].values()) == summary['BRH'])
    require(len(row['lines']) == summary['LF'])
    require(sum(count > 0 for count in row['lines'].values()) == summary['LH'])


def mapped_lines(source, span, line_counts):
    start, end, count = span
    index = bisect.bisect_right(source['starts'], start) - 1
    require(index >= 0 and start <= source['ends'][index])
    first = index + 1
    while index < len(source['starts']) and end > source['starts'][index]:
        if start <= source['starts'][index] and end >= source['ends'][index]:
            line_counts[index + 1] = count
        index += 1
    require(index >= first)
    return first


def native_function_report(function, source, lines, branches):
    mapped = []
    for span in function['ranges']:
        line = mapped_lines(source, span, lines)
        mapped.append(line)
        if function['block']:
            branches[(line, len(branches), 0)] = span[2]
    return mapped[0], max(span[2] for span in function['ranges'])


def reported_function_label(function, index):
    if function['anonymous']:
        return digest(('anonymous_' + str(index)).encode())
    return function['name_sha256']


def node_record_summary(functions, branches, lines):
    return {'FNF': len(functions) - 1,
            'FNH': sum(row['ranges'][0][2] > 0 for row in functions[1:]),
            'BRF': len(branches), 'BRH': sum(count > 0 for count in branches.values()),
            'LF': len(lines), 'LH': sum(count > 0 for count in lines.values())}


def node_record(script, source):
    # Pinned Node24 mapRangeToLines + summary + lcov.js semantics. Validation
    # of the immutable original is NOT an alternate test-suite reporter.
    lines = {i + 1: int(a == b) for i, (a, b) in enumerate(zip(source['starts'], source['ends']))}
    functions, hits, branches = [], [], {}
    for index, function in enumerate(script['functions']):
        first_line, maximum = native_function_report(function, source, lines, branches)
        if index > 0:
            label = reported_function_label(function, len(functions))
            functions.append((first_line, label))
            hits.append((maximum, label))
    return {'path': script['path'], 'lines': lines, 'fn': functions, 'fnda': hits,
            'branches': branches, 'summary': node_record_summary(script['functions'], branches, lines)}

def record_signature(record, hash_names=False):
    value = copy.deepcopy(record)
    if hash_names:
        value['fn'] = [(line, digest(name.encode())) for line, name in value['fn']]
        value['fnda'] = [(count, digest(name.encode())) for count, name in value['fnda']]
    # JSON doesn't support tuple dict keys; retain ordered branch tuples.
    value['branches'] = [list(key) + [count] for key, count in value['branches'].items()]
    return digest(encoded(value))


def verify_reporter(scripts, records, files):
    predicted = [node_record(script, files[script['path']]) for script in scripts]
    require(sorted(record_signature(row) for row in predicted)
            == sorted(record_signature(row, True) for row in records))


def original_line_union(records):
    lines = {}
    for record in records:
        rows = lines.setdefault(record['path'], {})
        max_coordinates(rows, record['lines'])
    return lines


def canonical_function(key, rows):
    return {'kind': key[0], 'name_sha256': key[1], 'start': key[2], 'end': key[3], 'block': key[4],
            'ranges': [[a, b, count] for (a, b), count in sorted(rows.items())]}


def canonical_file(source, lines, functions):
    ordered = sorted(functions.items(), key=lambda item: (item[0][0] != 'script', *item[0][1:]))
    return {'source': {key: source[key] for key in ('sha256', 'bytes', 'units')},
            'lines': [[number, count] for number, count in sorted(lines.items())],
            'functions': [canonical_function(key, rows) for key, rows in ordered]}


def canonical_counters(raw, records, files):
    validated_scripts(raw['scripts'], files)
    verify_reporter(raw['scripts'], records, files)
    lines = original_line_union(records)
    require(set(lines) == set(raw['raw']))
    output = {name: canonical_file(files[name], lines[name], functions)
              for name, functions in sorted(raw['raw'].items())}
    return {'schema': 'cq.native-browser.counters/3', 'identity': 'source-utf16-block-ranges',
            'reporter': 'node-v24.22.2-exact-original', 'observations': raw['observations'],
            'scripts': raw['scripts'], 'files': output}

def retain(root, output, expected, node, platform, suite, test_exit, environment):
    integer(test_exit, 0, 255)
    fixed_output(output, platform, suite, environment)
    before = strict_json(private_read(output / 'begin.json', MAX_METADATA))
    keys(before, ('schema', 'source', 'node', 'producer', 'files'))
    require(before['schema'] == 'cq.native-browser.begin/1')
    source, files = source_identity(root, expected)
    tool = node_identity(root, node, platform)
    producer = producer_identity(environment, platform, suite)
    require(before['source'] == source and before['files'] == public_inventory(files)
            and before['node'] == tool and before['producer'] == producer)
    original = private_read(output / 'lcov.info', MAX_PROFILE)
    records = parse_lcov(original, files, 'web/' if suite == 'web' else '')
    counters = encoded(canonical_counters(raw_counters(output / 'raw', root, files), records, files))
    require(len(counters) <= MAX_PROFILE)
    metadata = {'schema': 'cq.native-browser.retained/1', 'source_before': before['source'],
                'source_after': source, 'node_before': before['node'], 'node_after': tool,
                'producer': producer, 'suite_exit': test_exit, 'files': public_inventory(files),
                'lcov': {'sha256': digest(original), 'bytes': len(original)},
                'counters': {'sha256': digest(counters), 'bytes': len(counters)},
                'original_check_passed': test_exit == 0, 'global_release_qualified': False}
    require(len(encoded(metadata)) <= MAX_METADATA)
    exclusive_write(output / 'counters.json', counters)
    exclusive_write(output / 'metadata.json', encoded(metadata))
    return metadata


def validate_reporter_ranges(ranges, maximum):
    require(type(ranges) is list and 0 < len(ranges) <= 100000)
    coordinates = set()
    for span in ranges:
        require(type(span) is list and len(span) == 3)
        start = integer(span[0], 0, maximum - 1)
        end = integer(span[1], start + 1, maximum)
        integer(span[2])
        require((start, end) not in coordinates)
        coordinates.add((start, end))


def validate_reporter_function(function, kind, maximum):
    keys(function, ('kind', 'name_sha256', 'anonymous', 'block', 'ranges'))
    require(function['kind'] == kind and type(function['anonymous']) is bool and type(function['block']) is bool)
    hexadecimal(function['name_sha256'])
    require(function['anonymous'] == (function['name_sha256'] == digest(b'')))
    validate_reporter_ranges(function['ranges'], maximum)


def validate_native_script(script, files):
    keys(script, ('path', 'instance', 'functions'))
    require(script['path'] in files)
    hexadecimal(script['instance'])
    require(type(script['functions']) is list and 0 < len(script['functions']) <= 100000)
    functions_seen = set()
    for index, function in enumerate(script['functions']):
        kind = 'script' if index == 0 else 'function'
        validate_reporter_function(function, kind, files[script['path']]['units'])
        identity = (function['kind'], *function_identity(function))
        require(identity not in functions_seen)
        functions_seen.add(identity)


def validated_scripts(scripts, files):
    require(type(scripts) is list and 0 < len(scripts) <= 4096)
    seen = set()
    for script in scripts:
        validate_native_script(script, files)
        key = script['path'], script['instance']
        require(key not in seen)
        seen.add(key)
    return scripts

def validated_lines(rows, maximum):
    require(type(rows) is list and rows)
    output = {}
    for entry in rows:
        require(type(entry) is list and len(entry) == 2)
        number = integer(entry[0], 1, maximum)
        require(number not in output)
        output[number] = integer(entry[1])
    return output


def validated_counter_file(row, source):
    keys(row, ('source', 'lines', 'functions'))
    require(row['source'] == {key: source[key] for key in ('sha256', 'bytes', 'units')})
    lines = validated_lines(row['lines'], len(source['starts']))
    require(type(row['functions']) is list and row['functions'])
    functions = {}
    for function in row['functions']:
        key, ranges = validated_function(function, source['units'])
        require(key not in functions)
        functions[key] = ranges
    return {'lines': lines, 'functions': functions}


def validated_counters(value, files):
    keys(value, ('schema', 'identity', 'reporter', 'observations', 'scripts', 'files'))
    require(value['schema'] == 'cq.native-browser.counters/3'
            and value['identity'] == 'source-utf16-block-ranges'
            and value['reporter'] == 'node-v24.22.2-exact-original'
            and type(value['files']) is dict and value['files'])
    validated_scripts(value['scripts'], files)
    output = {}
    for path, row in value['files'].items():
        require(path in files)
        output[path] = validated_counter_file(row, files[path])
    return output

def validated_function(function, maximum):
    keys(function, ('kind', 'name_sha256', 'start', 'end', 'block', 'ranges'))
    hexadecimal(function['name_sha256'])
    require(type(function['block']) is bool)
    require(function['kind'] in {'script', 'function'})
    start = integer(function['start'], 0, maximum)
    end = integer(function['end'], start + 1, maximum)
    key = (function['kind'], function['name_sha256'], start, end, function['block'])
    require(type(function['ranges']) is list and function['ranges'])
    ranges = {}
    for entry in function['ranges']:
        require(type(entry) is list and len(entry) == 3)
        a = integer(entry[0], start, end - 1)
        b = integer(entry[1], a + 1, end)
        require((a, b) not in ranges)
        ranges[(a, b)] = integer(entry[2])
    require((start, end) in ranges)
    return key, ranges


def max_coordinates(current, incoming):
    for coordinate, count in incoming.items():
        current[coordinate] = max(current.get(coordinate, 0), count)


def merge_counter_file(current, incoming):
    max_coordinates(current['lines'], incoming['lines'])
    for function, incoming_ranges in incoming['functions'].items():
        ranges = current['functions'].setdefault(function, {})
        max_coordinates(ranges, incoming_ranges)


def merge_counters(profiles):
    require(profiles)
    output = {}
    for profile in profiles:
        for path, incoming in profile.items():
            current = output.setdefault(path, {'lines': {}, 'functions': {}})
            merge_counter_file(current, incoming)
    return output

def range_line(source, offset):
    integer(offset, 0, source['units'])
    return bisect.bisect_right(source['starts'], offset)


def sonar_hit(count):
    # SonarJS LCOVParser uses Java Integer for DA/BRDA. Never let an exact
    # canonical counter overflow that importer and silently disappear.
    return integer(count, 0, SONAR_MAX_HIT)


def function_hits(row):
    functions = {}
    for key, ranges in row['functions'].items():
        if key[0] == 'function':
            identity = key[:4]
            functions[identity] = max(functions.get(identity, 0), max(ranges.values()))
    return functions


def render_functions(row, source):
    functions = function_hits(row)
    output = []
    for key in sorted(functions):
        output.append(f'FN:{range_line(source, key[2])},function_{key[2]}_{key[3]}_{key[1]}')
    for key, count in sorted(functions.items()):
        output.append(f'FNDA:{sonar_hit(count)},function_{key[2]}_{key[3]}_{key[1]}')
    found, hit = len(functions), sum(count > 0 for count in functions.values())
    output.extend([f'FNF:{found}', f'FNH:{hit}'])
    return output, {'functions': found, 'functions_hit': hit}


def render_branches(path, row, source):
    output, gaps = [], []
    branches = [(key, start, end, count) for key, ranges in sorted(row['functions'].items()) if key[4]
                for (start, end), count in sorted(ranges.items())]
    for index, (key, start, end, count) in enumerate(branches):
        output.append(f'BRDA:{range_line(source, start)},{index},0,{sonar_hit(count)}')
        if count == 0:
            gaps.append({'path': path, 'kind': 'block-range', 'function': list(key), 'start': start, 'end': end})
    found, hit = len(branches), sum(entry[3] > 0 for entry in branches)
    output.extend([f'BRF:{found}', f'BRH:{hit}'])
    return output, {'branches': found, 'branches_hit': hit}, gaps


def render_lines(path, row):
    output = []
    gaps = []
    for number, count in sorted(row['lines'].items()):
        output.append(f'DA:{number},{sonar_hit(count)}')
        if count == 0:
            gaps.append({'path': path, 'kind': 'line', 'line': number})
    found = len(row['lines'])
    hit = sum(count > 0 for count in row['lines'].values())
    output.extend([f'LF:{found}', f'LH:{hit}', 'end_of_record'])
    return output, {'lines': found, 'lines_hit': hit}, gaps


def render_composite(profile, files):
    output = ['TN:']
    totals = {'lines': 0, 'lines_hit': 0, 'functions': 0, 'functions_hit': 0,
              'branches': 0, 'branches_hit': 0}
    gaps = []
    for path, row in sorted(profile.items()):
        functions = function_hits(row)
        gaps.extend({'path': path, 'kind': 'function', 'function': list(key)}
                    for key, count in functions.items() if count == 0)
        output.append('SF:' + path)
        function_text, functions = render_functions(row, files[path])
        branch_text, branches, branch_gaps = render_branches(path, row, files[path])
        line_text, lines, line_gaps = render_lines(path, row)
        output.extend(function_text + branch_text + line_text)
        for key, value in {**functions, **branches, **lines}.items():
            totals[key] += value
        gaps.extend(branch_gaps + line_gaps)
    data = ('\n'.join(output) + '\n').encode('utf-8')
    require(0 < len(data) <= MAX_PROFILE and totals['lines'] > 0)
    return data, totals, gaps


def read_archive(data):
    require(0 < len(data) <= MAX_ARCHIVE)
    try:
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            rows = archive.infolist()
            require(len(rows) == 3 and {row.filename for row in rows} == set(MEMBERS))
            output = {}
            for row in rows:
                require(row.filename not in output and row.orig_filename == row.filename
                        and not row.is_dir() and not row.flag_bits & 1
                        and row.compress_type in {zipfile.ZIP_STORED, zipfile.ZIP_DEFLATED}
                        and row.file_size <= MEMBERS[row.filename])
                mode = row.external_attr >> 16
                require(stat.S_IFMT(mode) in {0, stat.S_IFREG})
                with archive.open(row) as member:
                    value = member.read(MEMBERS[row.filename] + 1)
                require(len(value) == row.file_size and len(value) <= MEMBERS[row.filename])
                output[row.filename] = value
            return output
    except (zipfile.BadZipFile, zlib.error, RuntimeError, EOFError, NotImplementedError,
            ValueError, OSError):
        raise InvalidEvidence(REFUSAL) from None


def validate_metadata(metadata, entry, pins, role, files):
    keys(metadata, ('schema', 'source_before', 'source_after', 'node_before', 'node_after',
                    'producer', 'suite_exit', 'files', 'lcov', 'counters',
                    'original_check_passed', 'global_release_qualified'))
    require(metadata['schema'] == 'cq.native-browser.retained/1'
            and metadata['source_before'] == metadata['source_after']
            and metadata['node_before'] == metadata['node_after']
            and metadata['suite_exit'] == 0 and type(metadata['suite_exit']) is int
            and metadata['original_check_passed'] is True
            and metadata['global_release_qualified'] is False)
    source = metadata['source_after']
    keys(source, ('commit', 'tree', 'inventory_sha256', 'package_sha256', 'implementation_sha256'))
    require(source['commit'] == entry['source_commit'] and source['tree'] == pins['tree']
            and source['inventory_sha256'] == digest(encoded(public_inventory(files)))
            and metadata['files'] == public_inventory(files))
    for field in ('package_sha256', 'implementation_sha256'):
        hexadecimal(source[field])
        require(source[field] == pins['source_hashes'][field])
    producer = metadata['producer']
    require(producer == entry['producer'] and producer['repository'] == REPOSITORY
            and producer['head'] == pins['head'] and producer['node'] == NODE_VERSION
            and producer['suite'] == ('full-browser' if role == 'baseline' else 'web')
            and producer['platform'] == ('linux' if role == 'baseline' else role))
    tool = metadata['node_after']
    keys(tool, ('version', 'platform', 'arch', 'sha256', 'bytes'))
    require(tool['version'] == NODE_VERSION and tool['platform'] == PLATFORMS[producer['platform']]
            and tool['arch'] in {'x64', 'arm64'} and tool['sha256'] == entry['node_sha256'])
    integer(tool['bytes'], 1, MAX_TOOL)
    hexadecimal(tool['sha256'])


def checked_producer_entry(entry):
    for field in ('metadata_sha256', 'lcov_sha256', 'counters_sha256', 'node_sha256'):
        hexadecimal(entry[field])
    hexadecimal(entry['source_commit'], 40)
    for field in ('run_id', 'job_id'):
        require(type(entry[field]) is str and re.fullmatch('[1-9][0-9]{0,19}', entry[field]))
    producer = entry['producer']
    keys(producer, ('repository', 'run', 'attempt', 'job', 'event', 'head', 'platform', 'suite', 'node'))
    require(producer['repository'] == REPOSITORY and producer['node'] == NODE_VERSION
            and producer['run'] == entry['run_id'])
    for field in ('run', 'attempt'):
        require(type(producer[field]) is str and re.fullmatch('[1-9][0-9]{0,19}', producer[field]))
    require(producer['event'] in {'push', 'pull_request'})
    require(type(producer['job']) is str and re.fullmatch('[A-Za-z0-9_-]{1,80}', producer['job']))
    hexadecimal(producer['head'], 40)


def checked_entry(entry):
    # Native Browser originals retain every external provider-artifact fence.
    keys(entry, ('archive_sha256', 'metadata_sha256', 'lcov_sha256', 'counters_sha256',
                 'producer', 'source_commit', 'node_sha256', 'artifact_id', 'run_id', 'job_id'))
    hexadecimal(entry['archive_sha256'])
    require(type(entry['artifact_id']) is str and re.fullmatch('[1-9][0-9]{0,19}', entry['artifact_id']))
    checked_producer_entry(entry)


def checked_local_entry(entry):
    # The current Sonar subsection has NO provider artifact yet. A local hash
    # is byte binding only; do not invent an ID or externally completed step.
    keys(entry, ('kind', 'metadata_sha256', 'lcov_sha256', 'counters_sha256',
                 'producer', 'source_commit', 'node_sha256', 'run_id', 'job_id'))
    require(entry['kind'] == 'owned-current-sonar')
    checked_producer_entry(entry)


def read_local_members(directory, environment):
    fixed_output(directory, 'linux', 'full-browser', environment)
    require(directory.is_dir() and not directory.is_symlink())
    return {name: private_read(directory / name, limit) for name, limit in MEMBERS.items()}


def accept_members(members, entry, pins, role, files):
    keys(members, MEMBERS)
    require(all(type(members[name]) is bytes and 0 < len(members[name]) <= limit
                for name, limit in MEMBERS.items()))
    for name, field in [('metadata.json', 'metadata_sha256'), ('lcov.info', 'lcov_sha256'),
                        ('counters.json', 'counters_sha256')]:
        require(digest(members[name]) == entry[field])
    metadata = strict_json(members['metadata.json'])
    validate_metadata(metadata, entry, pins, role, files)
    for name, descriptor in [('lcov.info', 'lcov'), ('counters.json', 'counters')]:
        keys(metadata[descriptor], ('sha256', 'bytes'))
        require(metadata[descriptor] == {'sha256': digest(members[name]), 'bytes': len(members[name])})
    records = parse_lcov(members['lcov.info'], files, '' if role == 'baseline' else 'web/')
    document = strict_json(members['counters.json'])
    counters = validated_counters(document, files)
    # Reconstruct both serialized views from the exact same bounded leaf,
    # including every original zero. Whole-document equality also binds
    # granularity, function identity, omitted coordinates and reporter order.
    derived = aggregate_observations(document['observations'], files)
    require(canonical_counters(derived, records, files) == document)
    original_lines = original_line_union(records)
    require(set(original_lines) == set(counters))
    require(all(original_lines[path] == counters[path]['lines'] for path in counters))
    return counters, metadata


def accept_archive(data, entry, pins, role, files):
    require(role in {'linux', 'macos', 'windows'})
    checked_entry(entry)
    require(digest(data) == entry['archive_sha256'])
    return accept_members(read_archive(data), entry, pins, role, files)


def accept_local_baseline(root, members, entry, pins, files, source, node, environment):
    checked_local_entry(entry)
    require(node is not None and environment is not None)
    require(entry['source_commit'] == pins['current_commit'] == source['commit'])
    profile, metadata = accept_members(members, entry, pins, 'baseline', files)
    require(metadata['source_after'] == source)
    require(metadata['producer'] == producer_identity(environment, 'linux', 'full-browser'))
    require(metadata['node_after'] == node_identity(root, node, 'linux'))
    return profile, metadata


def union(root, expected, pins_data, expected_pins_sha256, archives, node=None, environment=None):
    hexadecimal(expected_pins_sha256)
    require(digest(pins_data) == expected_pins_sha256)
    pins = strict_json(pins_data)
    keys(pins, ('schema', 'head', 'tree', 'current_commit', 'source_hashes', 'entries'))
    require(pins['schema'] == 'cq.native-browser.authorized-pins/2')
    hexadecimal(pins['head'], 40)
    source, files = source_identity(root, expected)
    require(pins['current_commit'] == expected and pins['tree'] == source['tree'])
    keys(pins['source_hashes'], ('package_sha256', 'implementation_sha256'))
    require(all(pins['source_hashes'][field] == source[field] for field in pins['source_hashes']))
    keys(pins['entries'], ('baseline', 'linux', 'macos', 'windows'))
    keys(archives, ('baseline', 'linux', 'macos', 'windows'))
    profile, metadata = accept_local_baseline(root, archives['baseline'], pins['entries']['baseline'],
                                              pins, files, source, node, environment)
    profiles = [profile]
    originals = {'baseline': metadata}
    for role in ('linux', 'macos', 'windows'):
        profile, metadata = accept_archive(archives[role], pins['entries'][role], pins, role, files)
        profiles.append(profile)
        originals[role] = metadata
    # Every platform is a distinct actual original; repeating one artifact or
    # producer cannot fill a missing platform's acceptance.
    require(len({pins['entries'][role]['archive_sha256'] for role in ('linux', 'macos', 'windows')}) == 3)
    require(len({pins['entries'][role]['artifact_id'] for role in ('linux', 'macos', 'windows')}) == 3)
    require(len({(pins['entries'][role]['producer']['run'], pins['entries'][role]['producer']['attempt'])
                 for role in ('linux', 'macos', 'windows')}) == 1)
    require(len({pins['entries'][role]['job_id'] for role in pins['entries']}) == 4)
    combined = merge_counters(profiles)
    lcov, totals, gaps = render_composite(combined, files)
    unmeasured = sorted(set(files) - set(combined))
    passed = not gaps and not unmeasured
    result = {'schema': 'cq.native-browser.composite/3', 'source': source,
              'head': pins['head'], 'pins_sha256': expected_pins_sha256,
              'counter_identity': 'source-utf16-block-ranges',
              'sonar_projection': 'exact-block-offsets-int32-refusal',
              'sonar_import_qualified': False,
              'static_function_census_complete': False,
              'counter_leaf_binding': 'whole-document-derived-from-sanitized-observations',
              'baseline_authority': 'owned-current-sonar-measurement-no-provider-artifact',
              'native_authority': 'three-external-API-authenticated-original-artifacts',
              'original_reporter_correspondence': 'exact-node-v24.22.2-DA-FN-FNDA-BRDA-totals', 'original_lcov_rewritten': False,
              'range_hit_inheritance': False, 'threshold_percent': 100,
              'denominator_policy': 'retain every observed original zero/unmatched raw range and line',
              'totals': totals, 'unmeasured_first_party_files': unmeasured,
              'evidence_accepted': True, 'coverage_passed': passed,
              'global_release_qualified': False, 'production_promoted': False,
              'composite': {'sha256': digest(lcov), 'bytes': len(lcov)},
              'originals': originals}
    require(source_identity(root, expected)[0] == source)
    return lcov, result, gaps, 0 if passed else 1


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('begin', 'retain', 'union'))
    parser.add_argument('--source', required=True)
    parser.add_argument('--expected-commit', required=True)
    parser.add_argument('--output', required=True)
    parser.add_argument('--node')
    parser.add_argument('--platform', choices=tuple(PLATFORMS))
    parser.add_argument('--suite', choices=('web', 'full-browser'))
    parser.add_argument('--test-exit', type=int)
    parser.add_argument('--pins')
    parser.add_argument('--expected-pins-sha256')
    for role in ('baseline', 'linux', 'macos', 'windows'):
        parser.add_argument('--' + role)
    args = parser.parse_args(argv)
    try:
        root = Path(args.source)
        require(root.is_absolute() and not root.is_symlink())
        output = Path(args.output)
        require(output.is_absolute())
        if args.action == 'begin':
            begin(root, output, args.expected_commit, args.node, args.platform, args.suite, os.environ)
            return 0
        if args.action == 'retain':
            retain(root, output, args.expected_commit, args.node, args.platform, args.suite,
                   args.test_exit, os.environ)
            return 0
        require(args.pins is not None and not output.exists())
        temporary = Path(os.environ.get('RUNNER_TEMP', ''))
        require(temporary.is_absolute() and temporary.is_dir()
                and not temporary.is_symlink() and output == temporary / 'cq-browser-composite')
        archives = {role: private_read(getattr(args, role), MAX_ARCHIVE)
                    for role in ('linux', 'macos', 'windows')}
        archives['baseline'] = read_local_members(Path(args.baseline), os.environ)
        lcov, result, gaps, code = union(root, args.expected_commit,
                                         private_read(args.pins, MAX_METADATA),
                                         args.expected_pins_sha256, archives, args.node, os.environ)
        output.mkdir(mode=0o700)
        exclusive_write(output / 'lcov.info', lcov)
        exclusive_write(output / 'composite.json', encoded(result))
        exclusive_write(output / 'gaps.json', encoded(gaps))
        return code
    except (InvalidEvidence, OSError, ValueError, TypeError, KeyError, OverflowError,
            RecursionError, RuntimeError, subprocess.SubprocessError):
        print(REFUSAL, file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
