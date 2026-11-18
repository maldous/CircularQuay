"""Qualify retained Source observations only with real current complete witnesses.

Raw reports and exits are immutable observations. Only a strict same-run full
platform union and authenticated current three-OS tagged receipts can resolve
their two declared holds. This is not release or native activation authority.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import re
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import zipfile
import zlib

SPEC = importlib.util.spec_from_file_location('cq_source_gate_platform', Path(__file__).with_name('platform_coverage.py'))
assert SPEC is not None and SPEC.loader is not None
PC = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PC)
GM_SPEC = importlib.util.spec_from_file_location('cq_source_gate_sdk', Path(__file__).with_name('go_mcp_coverage.py'))
assert GM_SPEC is not None and GM_SPEC.loader is not None
GM = importlib.util.module_from_spec(GM_SPEC)
GM_SPEC.loader.exec_module(GM)
REPO = 'maldous/CircularQuay'
PREFIX = '/repos/' + REPO
LIMIT = 65536
ARCHIVE_LIMIT = 20 << 20
ROLES = {'linux': 'ubuntu-24.04', 'macos': 'macos-15', 'windows': 'windows-2025'}
REQUIRED = {'source_revision', 'go_toolchain', 'source_identity_before_checks',
            'repository_policy', 'format', 'tagged_integration', 'unit_tests',
            'statement_coverage', 'go_vet', 'source_identity_after_checks'}
OPTIONAL = {'staticcheck', 'gosec', 'govulncheck', 'secret_scan', 'workflow_lint',
            'sonarqube', 'native_reboot_and_restore', 'independent_review'}
HOLDS = {'statement_coverage': ('failed', 'coverage_below_100_percent'),
         'tagged_integration': ('not_run', 'requires_qualified_tagged_integration_lane')}
REFUSAL = 'Source qualification evidence refused.'
BUILD = {'GOENV': 'off', 'GOWORK': 'off', 'GOFLAGS': '-mod=readonly',
         'GOTOOLCHAIN': 'local', 'GOOS': '', 'GOARCH': '', 'GOEXPERIMENT': '',
         'CGO_ENABLED': '', 'GO111MODULE': 'on', 'GOAMD64': '', 'GOARM64': ''}
LOCKS = ('tools/integration-tools.json', '.github/workflows/integration.yml',
         '.github/scripts/source_gate.py', '.github/scripts/platform_coverage.py',
         '.github/workflows/sdk.yml', '.github/scripts/go_mcp_coverage.py', 'sdk/conformance/go-mcp.sh')


def require(value):
    if not value:
        raise ValueError(REFUSAL)


def read_json(path):
    return PC.strict_json(PC.safe_file(Path(path), PC.MAX_METADATA, True))


def raw_hold(check):
    name = check['name']
    require(check['required'] == (name in REQUIRED))
    if name in HOLDS and (check['status'], check.get('code')) == HOLDS[name]:
        return True
    if name in REQUIRED:
        require(check['status'] == 'passed' and 'code' not in check)
    else:
        require(check['status'] in ('passed', 'not_run'))
    return False


def build_selection(go, environment):
    require(go.is_absolute() and go.resolve(strict=True) == go)
    require({name: environment.get(name) for name in BUILD} == BUILD)
    require(environment.get('GOROOT') == str(go.parent.parent))
    return {'selectors': dict(BUILD), 'goroot': 'selected-executable-parent',
            'invocation': 'absolute-selected-go'}


def raw_checks(value, exit_code):
    require(type(exit_code) is int and exit_code in (0, 1))
    require(value.get('profile') == 'source' and value.get('production_qualified') is False
            and value.get('retention_only') is True and type(value.get('passed')) is bool)
    checks = PC.safe_checks(value.get('checks'))
    require({x['name'] for x in checks} == REQUIRED | OPTIONAL)
    holds = []
    for check in checks:
        if raw_hold(check):
            holds.append(check['name'])
    require(value['passed'] == (not holds) and exit_code == int(bool(holds)))
    return sorted(holds)


def identity(source, expected, go, jar, role):
    require(role in ROLES)
    build = build_selection(go, os.environ)
    source_value, _ = PC.source_identity(source, expected)
    tool = PC.tool_identity(go, role)
    locks = {path: PC.digest(PC.safe_file(source / path, PC.MAX_METADATA)) for path in LOCKS}
    manifest = PC.strict_json(PC.safe_file(source / LOCKS[0], PC.MAX_METADATA))
    jar_hash = PC.digest(PC.safe_file(jar, 256 << 20))
    require(jar_hash == manifest['wiremock']['sha256'])
    return {'source': source_value, 'go': tool, 'locks': locks, 'build': build,
            'wiremock_sha256': jar_hash, 'wiremock_version': manifest['wiremock']['version']}


def origin(environment, role):
    require(environment.get('GITHUB_REPOSITORY') == REPO and role in ROLES)
    out = {key: environment.get(name, '') for key, name in (
        ('run', 'GITHUB_RUN_ID'), ('attempt', 'GITHUB_RUN_ATTEMPT'),
        ('event', 'GITHUB_EVENT_NAME'), ('head', 'CQ_COVERAGE_PR_HEAD'))}
    require(environment.get('GITHUB_JOB') == 'wiremock')
    for key in ('run', 'attempt'):
        require(re.fullmatch('[1-9][0-9]{0,19}', out[key]) is not None)
    PC.hex_value(out['head'], 40)
    require(out['event'] in ('pull_request', 'push'))
    return dict(out, repository=REPO, role=role, matrix=ROLES[role], job='wiremock')


def fixed_directory(output, role):
    root = Path(os.environ['RUNNER_TEMP'])
    require(root.is_absolute() and root.resolve(strict=True) == root)
    require(output == root / ('cq-tagged-' + role))


def tagged_begin(source, expected, go, jar, role, output):
    fixed_directory(output, role)
    before = identity(source, expected, go, jar, role)
    producer = origin(os.environ, role)
    output.mkdir(mode=0o700)
    PC.write_new(output / 'begin.json', PC.encoded({'identity': before, 'producer': producer}))


def tagged_retain(source, expected, go, jar, role, output, vet_exit, test_exit):
    require(type(vet_exit) is int and 0 <= vet_exit <= 255)
    require(type(test_exit) is int and 0 <= test_exit <= 255)
    fixed_directory(output, role)
    before = read_json(output / 'begin.json')
    after = identity(source, expected, go, jar, role)
    producer = origin(os.environ, role)
    require(before == {'identity': after, 'producer': producer})
    value = {'schema': 'cq.tagged-source/1', 'before': before['identity'], 'after': after,
             'producer': producer, 'commands': ['go vet -tags=integration ./internal/integration',
             'go test -p 1 -tags=integration ./internal/integration'],
             'vet_exit': vet_exit, 'test_exit': test_exit, 'production_qualified': False}
    data = PC.encoded(value)
    require(len(data) <= LIMIT)
    PC.write_new(output / 'tagged.json', data)


def bounded_response(response, maximum, deadline):
    chunks, size = [], 0
    # HTTPResponse.read1 performs one buffered/raw read, rather than waiting to
    # fill the entire artifact across an unbounded sequence of socket reads.
    while True:
        require(time.monotonic() < deadline)
        part = response.read1(min(65536, maximum + 1 - size))
        require(type(part) is bytes and time.monotonic() < deadline)
        if not part:
            return b''.join(chunks)
        size += len(part)
        require(size <= maximum)
        chunks.append(part)


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class GitHub:
    """One public-repository read authority; bearer never follows artifact redirect."""
    def __init__(self, token, deadline):
        require(type(token) is str and 0 < len(token) <= 4096 and not any(x.isspace() for x in token))
        self.token, self.deadline, self.calls = token, deadline, 0
        self.opener = urllib.request.build_opener(NoRedirect())

    def request(self, path):
        require(path.startswith(PREFIX + '/actions/') and '..' not in path and '#' not in path)
        self.calls += 1
        remaining = self.deadline - time.monotonic()
        require(self.calls <= 128 and remaining > 0)
        request = urllib.request.Request('https://api.github.com' + path, headers={
            'Authorization': 'Bearer ' + self.token, 'Accept': 'application/vnd.github+json',
            'X-GitHub-Api-Version': '2022-11-28', 'User-Agent': 'cq-source-gate'})
        return self.opener.open(request, timeout=min(15, remaining))

    def json(self, path):
        with self.request(path) as response:
            require(response.status == 200)
            data = bounded_response(response, 1 << 20, self.deadline)
        require(len(data) <= 1 << 20)
        return PC.strict_json(data)

    def archive(self, identifier):
        require(type(identifier) is int and identifier > 0)
        try:
            response = self.request(PREFIX + '/actions/artifacts/' + str(identifier) + '/zip')
        except urllib.error.HTTPError as error:
            require(error.code == 302)
            location = error.headers.get('Location', '')
            error.close()
        else:
            response.close()
            raise ValueError(REFUSAL)
        url = urllib.parse.urlsplit(location)
        require(url.scheme == 'https' and url.hostname is not None and
                url.hostname.endswith('.blob.core.windows.net') and not url.username and
                not url.password and url.port in (None, 443) and not url.fragment)
        remaining = self.deadline - time.monotonic()
        require(remaining > 0)
        # New request has no Authorization header; further redirects refused.
        with self.opener.open(urllib.request.Request(location), timeout=min(15, remaining)) as response:
            require(response.status == 200)
            data = bounded_response(response, ARCHIVE_LIMIT, self.deadline)
        require(0 < len(data) <= ARCHIVE_LIMIT)
        return data


def collection(api, path, field):
    value = api.json(path + ('&' if '?' in path else '?') + 'per_page=100')
    require(type(value.get('total_count')) is int and 0 <= value['total_count'] <= 100)
    rows = value.get(field)
    require(type(rows) is list and len(rows) == value['total_count'])
    return rows


def validate_original_run(run, successful):
    require(type(run.get('id')) is int and 0 < run['id'] < 10**20)
    require((not successful or run.get('conclusion') == 'success') and run.get('run_attempt') == 1)
    require(run.get('repository', {}).get('full_name') == REPO
            and str(run.get('repository', {}).get('id')) == PC.REPOSITORY_ID)
    require(run.get('status') in ('queued', 'pending', 'waiting', 'in_progress', 'completed', 'requested'))


def discover(api, head, event, sleep=time.sleep, workflow="integration.yml", successful=True):
    PC.hex_value(head, 40)
    require(event in ('push', 'pull_request'))
    require(workflow in ('integration.yml', 'sdk.yml'))
    while True:
        rows = collection(api, PREFIX + '/actions/workflows/' + workflow + '/runs?head_sha=' + head, 'workflow_runs')
        matches = [row for row in rows if row.get('head_sha') == head and row.get('event') == event
                   and row.get('path') == '.github/workflows/' + workflow]
        require(len(matches) <= 1)
        if matches and (not successful or matches[0].get('status') == 'completed'):
            run = matches[0]
            validate_original_run(run, successful)
            return run
        require(time.monotonic() + 5 < api.deadline)
        sleep(5)


def successful_jobs(api, run):
    jobs = collection(api, PREFIX + '/actions/runs/' + str(run['id']) + '/attempts/1/jobs', 'jobs')
    require(len(jobs) == 3)
    out = {}
    for role, matrix in ROLES.items():
        matched = [job for job in jobs if job.get('name') == 'WireMock (' + matrix + ')']
        require(len(matched) == 1)
        job = matched[0]
        require(job.get('conclusion') == 'success' and job.get('status') == 'completed'
                and job.get('run_id') == run['id'] and job.get('run_attempt') == 1)
        for name in ('Download and verify pinned WireMock standalone JAR', 'Run tagged synthetic provider suite',
                     'Bind tagged fixture identity before execution', 'Upload bounded tagged fixture receipt'):
            steps = [s for s in job.get('steps', []) if s.get('name') == name]
            require(len(steps) == 1 and steps[0].get('conclusion') == 'success')
        require(type(job.get('id')) is int and 0 < job['id'] < 10**20)
        out[role] = job['id']
    require(len(set(out.values())) == 3)
    return out


def receipt_from_archive(data, artifact):
    require(artifact.get('digest') == 'sha256:' + PC.digest(data))
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        entries = archive.infolist()
        require(len(entries) == 1)
        entry = entries[0]
        require(entry.filename == 'tagged.json' and not entry.is_dir() and not entry.flag_bits & 1)
        require(0 < entry.file_size <= LIMIT and entry.compress_size <= ARCHIVE_LIMIT)
        require((entry.external_attr >> 16) & 0o170000 in (0, 0o100000))
        with archive.open(entry) as stream:
            raw = stream.read(LIMIT + 1)
        require(len(raw) == entry.file_size)
    return PC.strict_json(raw), PC.digest(raw)


def validate_receipt(value, source, locks, run, role, head, event, manifest):
    PC.keys(value, {'schema', 'before', 'after', 'producer', 'commands', 'vet_exit', 'test_exit', 'production_qualified'})
    require(value['schema'] == 'cq.tagged-source/1' and value['before'] == value['after'])
    before = value['before']
    PC.keys(before, {'source', 'go', 'locks', 'build', 'wiremock_sha256', 'wiremock_version'})
    require(before['source'] == source and before['locks'] == locks)
    require(before['build'] == {'selectors': BUILD, 'goroot': 'selected-executable-parent',
                                'invocation': 'absolute-selected-go'})
    tool = before['go']
    PC.keys(tool, {'version', 'platform', 'sha256', 'bytes'})
    require(tool['version'] == PC.GO_VERSION and tool['platform'] in PC.PLATFORMS[role])
    PC.hex_value(tool['sha256'])
    require(type(tool['bytes']) is int and 0 < tool['bytes'] <= PC.MAX_TOOL)
    require(before['wiremock_sha256'] == manifest['wiremock']['sha256'] and
            before['wiremock_version'] == manifest['wiremock']['version'])
    require(value['producer'] == {'repository': REPO, 'run': str(run['id']), 'attempt': '1',
        'event': event, 'head': head, 'role': role, 'matrix': ROLES[role], 'job': 'wiremock'})
    require(value['commands'] == ['go vet -tags=integration ./internal/integration',
                                'go test -p 1 -tags=integration ./internal/integration'])
    require(type(value['vet_exit']) is int and value['vet_exit'] == 0 and
            type(value['test_exit']) is int and value['test_exit'] == 0 and value['production_qualified'] is False)


def tagged_scope(source):
    # A new tagged package/test requires an explicit reviewed witness contract.
    expected = {'internal/integration/wiremock_test.go', 'internal/integration/wiremock_provider_test.go',
                'sdk/go/official_mcp_integration_test.go'}
    names = PC.git(source, 'ls-files', '-z').decode().split('\0')
    observed = set()
    for name in names:
        if not name.endswith('.go'):
            continue
        data = PC.safe_file(source / name, 4 << 20)
        if re.search(rb'(?m)^//(?:go:build|\s*\+build) .*\bintegration\b', data):
            observed.add(name)
    require(observed == expected)
    sdk = PC.safe_file(source / 'sdk/go/official_mcp_integration_test.go', 4 << 20)
    require(re.findall(rb'(?m)^func ((?:Test|Fuzz|Benchmark)[A-Za-z0-9_]*)\(', sdk) ==
            [b'TestOfficialMCPClientSessionAgainstSyntheticCq'])
    return {name: PC.digest(PC.safe_file(source / name, 4 << 20)) for name in sorted(expected)}


SDK_STEPS = ('Exercise the official MCP TypeScript client against synthetic cq',
             'Retain SDK-specific official Go MCP coverage')


def sdk_steps_ready(job):
    ready = []
    for number, name in enumerate(SDK_STEPS, 15):
        rows = [step for step in job.get('steps', []) if step.get('name') == name]
        require(len(rows) <= 1)
        if not rows:
            return False
        step = rows[0]
        require(step.get('number') == number)
        require(step.get('conclusion') in (None, 'success'))
        ready.append(step.get('conclusion') == 'success')
    return all(ready)


def sdk_job(api, run):
    jobs = collection(api, PREFIX + '/actions/runs/' + str(run['id']) + '/attempts/1/jobs', 'jobs')
    selected = [job for job in jobs if job.get('name') == 'TypeScript SDK (Node 24.22.2, ubuntu-24.04)']
    require(len(selected) <= 1)
    if not selected:
        require(run.get('status') != 'completed')
        return None
    job = selected[0]
    require(type(job.get('id')) is int and 0 < job['id'] < 10**20)
    require(job.get('run_id') == run['id'] and job.get('run_attempt') == 1)
    if sdk_steps_ready(job):
        return job
    require(job.get('status') != 'completed')
    return None


def sdk_ready(api, run, current, sleep=time.sleep):
    while True:
        # Reconcile only the same original run and exact selected witness.
        latest = api.json(PREFIX + '/actions/runs/' + str(run['id']))
        for key in ('id', 'head_sha', 'event', 'path', 'run_attempt'):
            require(latest.get(key) == run.get(key))
        require(latest.get('repository', {}).get('full_name') == REPO)
        require(str(latest.get('repository', {}).get('id')) == PC.REPOSITORY_ID)
        job = sdk_job(api, latest)
        artifacts = collection(api, PREFIX + '/actions/runs/' + str(run['id']) + '/artifacts', 'artifacts')
        found = [row for row in artifacts if row.get('name') == 'go-mcp-coverage-' + current['commit']]
        require(len(found) <= 1)
        if job is not None and found:
            artifact = found[0]
            require(artifact.get('expired') is False and artifact.get('workflow_run', {}).get('id') == run['id'])
            run['status'], run['conclusion'] = latest.get('status'), latest.get('conclusion')
            return job, artifact
        require(latest.get('status') != 'completed' and time.monotonic() + 5 < api.deadline)
        sleep(5)


def sdk_members(data):
    members = {}
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        entries = archive.infolist()
        require(len(entries) == 2 and {entry.filename for entry in entries} == {'coverage.out', 'sdk-metadata.json'})
        for entry in entries:
            cap = PC.MAX_PROFILE if entry.filename == 'coverage.out' else PC.MAX_METADATA
            require(not entry.is_dir() and not entry.flag_bits & 1 and 0 < entry.file_size <= cap
                    and (entry.external_attr >> 16) & 0o170000 in (0, 0o100000))
            with archive.open(entry) as stream:
                raw = stream.read(cap + 1)
            require(len(raw) == entry.file_size)
            members[entry.filename] = raw
    return members


def sdk_witness(api, source, current, reference, source_tool):
    head, event = reference['pr_head'], reference['event']
    run = discover(api, head, event, workflow='sdk.yml', successful=False)
    job, artifact = sdk_ready(api, run, current)
    data = api.archive(artifact['id'])
    require(type(artifact.get('size_in_bytes')) is int and len(data) == artifact['size_in_bytes']
            and artifact.get('digest') == 'sha256:' + PC.digest(data))
    members = sdk_members(data)
    value = PC.strict_json(members['sdk-metadata.json'])
    require(value['source'] == current)
    expected_producer = {'repository': REPO, 'run_id': str(run['id']), 'attempt': 1,
                        'job': 'typescript', 'matrix': 'ubuntu-24.04/node24.22.2',
                        'pr_head': head, 'step': 'official_go_mcp_conformance'}
    require(value['producer'] == expected_producer)
    # The independently API-bound artifact authenticates the producer assertion;
    # it does not turn the reported binary digest into retained executable bytes.
    pins = {'sdk_commit': current['commit'], 'tree': current['tree'], 'locks': current['locks'],
            'toolchain': value['toolchain'], 'binary': value['binary'], 'producer': expected_producer}
    GM.validate_sdk(value, pins, members['coverage.out'])
    require(value['toolchain'] == {key: source_tool[key] for key in ('version', 'platform', 'sha256')})
    return {'run': run['id'], 'attempt': 1, 'job': job['id'], 'artifact': artifact['id'],
            'archive_digest': artifact['digest'], 'members': {key: PC.digest(data) for key, data in members.items()},
            'binary_provenance': 'authenticated producer measurement assertion; executable bytes not retained',
            'binary': value['binary'], 'overall_sdk_conclusion': run.get('conclusion')}


def verify_union_archives(raw, pins, current, tracked, artifacts, union_dir):
    # Reuse, do not modify, the reviewed full four-role validator and math.
    # Bind the aggregate to those exact authenticated archives, not JSON claims.
    profiles = []
    for role in PC.ROLES:
        entry = pins['entries'][role]
        content, metadata = PC.archive_members(PC.archive_file(artifacts / role), role, entry)
        if role == 'source':
            PC.validate_baseline(metadata, current, content)
            require(metadata == raw['original_source_metadata_unchanged'])
        else:
            PC.validate_platform(metadata, entry['descriptor'], content)
        profiles.append(PC.profile(content, tracked))
    canonical, measured, _ = PC.merged_profiles(profiles)
    require(raw.get('coverage') == measured and
            PC.safe_file(union_dir / 'coverage.out', PC.MAX_PROFILE, True) == canonical)


def qualification_prerequisites(raw, needs):
    coverage = raw['coverage']
    require(coverage.get('coverage_gate_passed') is True and coverage.get('threshold_percent') == 100 and
            type(coverage.get('total_statements')) is int and coverage['total_statements'] > 0 and
            coverage.get('covered_statements') == coverage['total_statements'])
    require(raw.get('production_qualified') is False and raw.get('source_report_promoted') is False)
    require(set(needs) == {'tests', 'native-macos', 'native-windows', 'assurance', 'cross-build-gate'})
    require(all(row.get('result') == 'success' for row in needs.values()))
    require(all(x == 'success' for x in raw['original_job_results_unchanged'].values()))
    return coverage


def wiremock_witnesses(api, source, expected, current, locks, head, event):
    run = discover(api, head, event)
    jobs = successful_jobs(api, run)
    artifacts = collection(api, PREFIX + '/actions/runs/' + str(run['id']) + '/artifacts', 'artifacts')
    bindings = {}
    for role in ROLES:
        name = 'tagged-source-' + role + '-' + expected
        found = [x for x in artifacts if x.get('name') == name]
        require(len(found) == 1)
        artifact = found[0]
        require(artifact.get('expired') is False and type(artifact.get('size_in_bytes')) is int
                and 0 < artifact['size_in_bytes'] <= 131072)
        require(artifact.get('workflow_run', {}).get('id') == run['id'])
        data = api.archive(artifact['id'])
        require(len(data) == artifact['size_in_bytes'] and len(data) <= 131072)
        value, member_hash = receipt_from_archive(data, artifact)
        validate_receipt(value, current, locks, run, role, head, event,
                         PC.strict_json(PC.safe_file(source / LOCKS[0], PC.MAX_METADATA)))
        bindings[role] = {'run': run['id'], 'attempt': 1, 'job': jobs[role], 'artifact': artifact['id'],
                          'archive_digest': artifact['digest'], 'receipt_sha256': member_hash}
    return bindings


def qualify(source, expected, union_dir, needs, audit_exit, api, pins_file, artifacts):
    current, tracked = PC.source_identity(source, expected)
    raw = read_json(union_dir / 'platform-union.json')
    pins_bytes = PC.safe_file(pins_file, PC.MAX_METADATA, True)
    pins = PC.strict_json(pins_bytes)
    PC.validate_pins(pins, current)
    require(raw.get('pins_sha256') == PC.digest(pins_bytes) and raw.get('producers') == pins['entries']
            and raw.get('original_job_results_unchanged') == pins['original_job_results'])
    scope = tagged_scope(source)
    verify_union_archives(raw, pins, current, tracked, artifacts, union_dir)
    require(raw.get('source') == current and raw.get('kind') == 'same-revision-platform-source-coverage')
    require(raw.get('coverage_sha256') == PC.digest(PC.safe_file(union_dir / 'coverage.out', PC.MAX_PROFILE, True)))
    coverage = qualification_prerequisites(raw, needs)
    metadata = raw['original_source_metadata_unchanged']
    # The observation job already verified the captured actual audit exit.
    # Here it remains an immutable expected observation, not a rewritten pass.
    expected_exit = 0 if metadata.get('passed') is True else 1
    require(audit_exit is None or audit_exit == expected_exit)
    holds = raw_checks(metadata, expected_exit)
    reference = raw['producers']['source']['descriptor']['producer']
    head, event = reference['pr_head'], reference['event']
    require(reference['run_id'] == os.environ['GITHUB_RUN_ID'] and reference['attempt'] == os.environ['GITHUB_RUN_ATTEMPT']
            and head == os.environ['CQ_COVERAGE_PR_HEAD'] and event == os.environ['GITHUB_EVENT_NAME'])
    locks = {path: PC.digest(PC.safe_file(source / path, PC.MAX_METADATA)) for path in LOCKS}
    bindings = wiremock_witnesses(api, source, expected, current, locks, head, event)
    sdk = sdk_witness(api, source, current, reference, pins['entries']['source']['descriptor']['toolchain'])
    require(PC.source_identity(source, expected)[0] == current)
    return {'schema': 'cq.authoritative-source-gate/1', 'source': current, 'qualified': True,
            'resolved_holds': holds, 'tagged': bindings, 'sdk_tagged': sdk, 'tagged_scope': scope, 'coverage': coverage,
            'raw_metadata_sha256': PC.digest(PC.encoded(raw['original_source_metadata_unchanged'])),
            'platform_union_sha256': PC.digest(PC.safe_file(union_dir / 'platform-union.json', PC.MAX_METADATA, True)),
            'production_qualified': False, 'release_qualified': False}


def main(argv=None):
    token = os.environ.pop('CQ_TAGGED_API_TOKEN', '')
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('tagged-begin', 'tagged-retain', 'observe', 'qualify'))
    for key in ('source', 'expected-commit', 'go', 'jar', 'role', 'output', 'metadata', 'needs', 'union', 'pins', 'artifacts'):
        parser.add_argument('--' + key)
    for key in ('vet-exit', 'test-exit', 'audit-exit'):
        parser.add_argument('--' + key, type=int)
    args = parser.parse_args(argv)
    try:
        if args.action == 'observe':
            raw_checks(read_json(args.metadata), args.audit_exit)
            return 0
        source, output = Path(args.source), Path(args.output)
        if args.action == 'tagged-begin':
            tagged_begin(source, args.expected_commit, Path(args.go), Path(args.jar), args.role, output)
        elif args.action == 'tagged-retain':
            tagged_retain(source, args.expected_commit, Path(args.go), Path(args.jar), args.role, output, args.vet_exit, args.test_exit)
        else:
            api = GitHub(token, time.monotonic() + 600)
            result = qualify(source, args.expected_commit, Path(args.union), read_json(args.needs), args.audit_exit, api, Path(args.pins), Path(args.artifacts))
            PC.write_new(output, PC.encoded(result))
        return 0
    except (OSError, ValueError, TypeError, KeyError, AttributeError, RecursionError, zipfile.BadZipFile, zlib.error, EOFError, RuntimeError, subprocess.SubprocessError):
        print(REFUSAL, file=__import__('sys').stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
