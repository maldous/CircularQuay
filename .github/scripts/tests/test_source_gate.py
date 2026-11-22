"""Owned evidence and explicit API fixtures, not hosted qualification assertions."""
from __future__ import annotations

import contextlib
import copy
import importlib.util
import io
import os
from pathlib import Path
import stat
import tempfile
import unittest
from unittest import mock
import urllib.error
import zipfile

SCRIPT = Path(__file__).resolve().parents[1] / 'source_gate.py'
SPEC = importlib.util.spec_from_file_location('source_gate_contract', SCRIPT)
assert SPEC is not None and SPEC.loader is not None
P = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(P)
COMMIT, TREE, HEAD = 'a' * 40, 'b' * 40, 'c' * 40
SOURCE = {'commit': COMMIT, 'tree': TREE, 'tracked_clean': True,
          'locks': {'go.mod': '1' * 64, 'go.sum': '2' * 64}}
MANIFEST = {'wiremock': {'sha256': '3' * 64, 'version': '3.13.2'}}


def tool(role='linux'):
    return {'version': P.PC.GO_VERSION, 'platform': {'linux': 'linux/amd64',
            'source': 'linux/amd64', 'macos': 'darwin/arm64', 'windows': 'windows/amd64'}[role],
            'sha256': '4' * 64, 'bytes': 123}


def raw():
    checks = [{'name': name, 'required': True, 'status': 'passed'} for name in sorted(P.REQUIRED)]
    checks += [{'name': name, 'required': False, 'status': 'not_run', 'code': 'tool_not_selected'}
               for name in sorted(P.OPTIONAL)]
    for item in checks:
        if item['name'] in P.HOLDS:
            item['status'], item['code'] = P.HOLDS[item['name']]
    return {'profile': 'source', 'production_qualified': False, 'retention_only': True,
            'passed': False, 'checks': checks}


def run(workflow='integration.yml'):
    return {'id': 70, 'run_attempt': 1, 'head_sha': HEAD, 'event': 'pull_request',
            'path': '.github/workflows/' + workflow, 'status': 'completed', 'conclusion': 'success',
            'repository': {'full_name': P.REPO, 'id': int(P.PC.REPOSITORY_ID)}}


def jobs():
    names = ['Download and verify pinned WireMock standalone JAR', 'Run tagged synthetic provider suite',
             'Bind tagged fixture identity before execution', 'Upload bounded tagged fixture receipt']
    return [{'id': i + 1, 'run_id': 70, 'run_attempt': 1, 'name': 'WireMock (' + matrix + ')',
             'status': 'completed', 'conclusion': 'success',
             'steps': [{'name': name, 'conclusion': 'success'} for name in names]}
            for i, matrix in enumerate(P.ROLES.values())]


def receipt(role='linux', locks=None):
    before = {'source': copy.deepcopy(SOURCE), 'go': tool(role), 'locks': locks or {},
              'wiremock_sha256': MANIFEST['wiremock']['sha256'], 'wiremock_version': '3.13.2',
              'build': {'selectors': dict(P.BUILD), 'goroot': 'selected-executable-parent', 'invocation': 'absolute-selected-go'}}
    return {'schema': 'cq.tagged-source/1', 'before': before, 'after': copy.deepcopy(before),
            'producer': {'repository': P.REPO, 'run': '70', 'attempt': '1', 'event': 'pull_request',
                         'head': HEAD, 'role': role, 'matrix': P.ROLES[role], 'job': 'wiremock'},
            'commands': ['go vet -tags=integration ./internal/integration',
                         'go test -p 1 -tags=integration ./internal/integration'],
            'vet_exit': 0, 'test_exit': 0, 'production_qualified': False}


def archive(members):
    output = io.BytesIO()
    with zipfile.ZipFile(output, 'w') as zipped:
        for name, content in members.items():
            info = zipfile.ZipInfo(name)
            info.create_system = 3
            info.external_attr = (stat.S_IFREG | 0o600) << 16
            zipped.writestr(info, content)
    return output.getvalue()


def producer(role):
    return {'repository': P.REPO, 'repository_id': P.PC.REPOSITORY_ID, 'run_id': '10', 'attempt': '1',
            'job': P.PC.ROLES[role], 'event': 'pull_request', 'pr_head': HEAD, 'step': 'platform_counters'}


def pins():
    entries = {}
    for index, role in enumerate(P.PC.ROLES):
        entries[role] = {'artifact_id': str(20 + index), 'archive_sha256': '5' * 64,
            'descriptor': {'version': 1, 'role': role, 'source': SOURCE, 'toolchain': tool(role),
                'producer': producer(role), 'coverage': {'mode': 'atomic', 'bytes': 100, 'sha256': '6' * 64},
                'metadata_sha256': '7' * 64, 'metadata_bytes': 500}}
    return {'version': 1, 'kind': 'same-run-platform-coverage-pins', 'source': SOURCE,
            'entries': entries, 'original_job_results': {name: 'success' for name in P.PC.ROLES.values()}}


class Contracts(unittest.TestCase):
    def setUp(self):
        for target in ('subprocess.Popen', 'urllib.request.OpenerDirector.open'):
            patch = mock.patch(target, side_effect=AssertionError('unselected external effect'))
            patch.start()
            self.addCleanup(patch.stop)

    def test_raw_only_two_named_holds_are_observations_not_passes(self):
        value = raw()
        before = copy.deepcopy(value)
        self.assertEqual(P.raw_checks(value, 1), ['statement_coverage', 'tagged_integration'])
        self.assertEqual(value, before)
        for item in value['checks']:
            if item['name'] in P.HOLDS:
                item.update(status='passed')
                item.pop('code')
        value['passed'] = True
        self.assertEqual(P.raw_checks(value, 0), [])

    def test_every_other_required_failure_missing_or_not_run_refuses(self):
        for name in P.REQUIRED:
            for status in ('failed', 'not_run', 'skipped', 'cancelled'):
                value = raw()
                item = next(x for x in value['checks'] if x['name'] == name)
                item.update(status=status, code='other_failure')
                with self.subTest(name=name, status=status), self.assertRaises(ValueError):
                    P.raw_checks(value, 1)
            value = raw()
            value['checks'] = [x for x in value['checks'] if x['name'] != name]
            with self.assertRaises(ValueError):
                P.raw_checks(value, 1)

    def test_unknown_duplicate_required_flags_and_optional_failures_refuse(self):
        changes = [lambda x: x['checks'].append(dict(x['checks'][0])),
                   lambda x: x['checks'].append({'name': 'new_required', 'required': True, 'status': 'passed'}),
                   lambda x: x['checks'][0].update(required=False),
                   lambda x: next(r for r in x['checks'] if not r['required']).update(status='failed')]
        for edit in changes:
            value = raw()
            edit(value)
            with self.assertRaises(ValueError): P.raw_checks(value, 1)

    def test_raw_exit_and_original_pass_disagreement_crash_bool_refuse(self):
        for code in (0, 2, 125, True, None):
            with self.assertRaises(ValueError): P.raw_checks(raw(), code)
        for field in ('passed', 'production_qualified'):
            value = raw(); value[field] = True
            with self.assertRaises(ValueError): P.raw_checks(value, 1)

    def test_all_three_original_os_jobs_and_required_steps_must_succeed(self):
        api = mock.Mock()
        api.json.return_value = {'total_count': 3, 'jobs': jobs()}
        self.assertEqual(set(P.successful_jobs(api, run())), set(P.ROLES))
        for index in range(3):
            for status in ('failure', 'cancelled', 'skipped', None):
                rows = jobs(); rows[index]['conclusion'] = status
                api.json.return_value = {'total_count': 3, 'jobs': rows}
                with self.assertRaises(ValueError): P.successful_jobs(api, run())
        for index in range(4):
            rows = jobs(); rows[0]['steps'][index]['conclusion'] = 'skipped'
            api.json.return_value = {'total_count': 3, 'jobs': rows}
            with self.assertRaises(ValueError): P.successful_jobs(api, run())

    def test_discovery_original_source_event_attempt_and_ambiguity(self):
        api = mock.Mock(deadline=10**15)
        api.json.return_value = {'total_count': 1, 'workflow_runs': [run()]}
        self.assertEqual(P.discover(api, HEAD, 'pull_request')['id'], 70)
        for edit in (lambda x: x.update(run_attempt=2), lambda x: x.update(conclusion='failure'),
                     lambda x: x['repository'].update(full_name='other/repository')):
            value = run(); edit(value)
            api.json.return_value = {'total_count': 1, 'workflow_runs': [value]}
            with self.assertRaises(ValueError): P.discover(api, HEAD, 'pull_request')
        api.json.return_value = {'total_count': 2, 'workflow_runs': [run(), run()]}
        with self.assertRaises(ValueError): P.discover(api, HEAD, 'pull_request')

    def test_missing_run_deadline_and_incomplete_collection_refuse(self):
        api = mock.Mock(deadline=0)
        api.json.return_value = {'total_count': 0, 'workflow_runs': []}
        with self.assertRaises(ValueError): P.discover(api, HEAD, 'pull_request', sleep=lambda _: self.fail())
        api.json.return_value = {'total_count': 101, 'jobs': []}
        with self.assertRaises(ValueError): P.collection(api, P.PREFIX + '/actions/runs/70/jobs', 'jobs')

    def test_receipt_checks_every_source_tool_manifest_command_and_actual_exit(self):
        original = receipt()
        P.validate_receipt(original, SOURCE, {}, run(), 'linux', HEAD, 'pull_request', MANIFEST)
        changes = [lambda x: x.update(test_exit=1), lambda x: x.update(vet_exit=1),
                   lambda x: x.update(test_exit=False), lambda x: x.update(production_qualified=True),
                   lambda x: x['producer'].update(attempt='2'), lambda x: x['commands'].append('other'),
                   lambda x: x['before']['go'].update(platform='windows/amd64'),
                   lambda x: x['before'].update(wiremock_sha256='9'*64),
                   lambda x: x['after']['source'].update(tree='8'*40)]
        for edit in changes:
            value = copy.deepcopy(original); edit(value)
            with self.assertRaises(ValueError):
                P.validate_receipt(value, SOURCE, {}, run(), 'linux', HEAD, 'pull_request', MANIFEST)

    def test_provider_digest_and_fixed_archive_member_contract(self):
        data = archive({'tagged.json': P.PC.encoded(receipt())})
        artifact = {'digest': 'sha256:' + P.PC.digest(data)}
        value, digest = P.receipt_from_archive(data, artifact)
        self.assertEqual(value, receipt()); self.assertEqual(digest, P.PC.digest(P.PC.encoded(receipt())))
        with self.assertRaises(ValueError): P.receipt_from_archive(data, {'digest': 'sha256:' + '0'*64})
        for members in ({'../tagged.json': b'{}'}, {'tagged.json': b'{}', 'private': b'x'},
                        {'tagged.json': b'x' * (P.LIMIT + 1)}):
            data = archive(members)
            with self.assertRaises(ValueError):
                P.receipt_from_archive(data, {'digest': 'sha256:' + P.PC.digest(data)})

    def test_pre_post_identity_drift_and_real_owned_exclusive_retention(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve(); output = root / 'cq-tagged-linux'
            environment = {'RUNNER_TEMP': str(root), 'GITHUB_REPOSITORY': P.REPO, 'GITHUB_JOB': 'wiremock',
                           'GITHUB_RUN_ID': '70', 'GITHUB_RUN_ATTEMPT': '1', 'GITHUB_EVENT_NAME': 'pull_request',
                           'CQ_COVERAGE_PR_HEAD': HEAD}
            with mock.patch.dict(os.environ, environment), mock.patch.object(P, 'identity', return_value=receipt()['before']) as identity:
                P.tagged_begin(root, COMMIT, root/'go', root/'jar', 'linux', output)
                P.tagged_retain(root, COMMIT, root/'go', root/'jar', 'linux', output, 0, 0)
                self.assertEqual(P.read_json(output/'tagged.json')['test_exit'], 0)
                with self.assertRaises(FileExistsError): P.tagged_retain(root, COMMIT, root/'go', root/'jar', 'linux', output, 0, 0)
                identity.return_value = dict(receipt()['before'], wiremock_sha256='9'*64)
                with self.assertRaises(ValueError): P.tagged_retain(root, COMMIT, root/'go', root/'jar', 'linux', output, 0, 0)

    def test_tagged_scope_refuses_extra_packages_or_unwitnessed_sdk_test(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve()
            names = ['internal/integration/wiremock_test.go', 'internal/integration/wiremock_provider_test.go',
                     'sdk/go/official_mcp_integration_test.go']
            for path in names:
                target = root/path; target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(b'//go:build integration\npackage fixture\n' +
                    (b'func TestOfficialMCPClientSessionAgainstSyntheticCq(t *testing.T) {}\n' if path.startswith('sdk/') else b''))
            with mock.patch.object(P.PC, 'git', return_value=('\0'.join(names)+'\0').encode()):
                self.assertEqual(set(P.tagged_scope(root)), set(names))
                with (root/names[-1]).open('ab') as out: out.write(b'func TestUnwitnessed(t *testing.T) {}\n')
                with self.assertRaises(ValueError): P.tagged_scope(root)
            extra = root/'other_test.go'; extra.write_bytes(b'//go:build integration\n')
            with mock.patch.object(P.PC, 'git', return_value=('\0'.join(names+['other_test.go'])+'\0').encode()):
                with self.assertRaises(ValueError): P.tagged_scope(root)

    def test_api_bearer_fixed_repo_only_and_never_on_artifact_redirect(self):
        api = P.GitHub('owned-token', 10**15)
        api.opener = mock.Mock()
        api.opener.open.return_value = mock.Mock()
        api.request(P.PREFIX + '/actions/runs/70')
        request = api.opener.open.call_args.args[0]
        self.assertEqual(request.get_header('Authorization'), 'Bearer owned-token')
        with self.assertRaises(ValueError): api.request('/repos/other/repo/actions/runs/70')
        response = mock.MagicMock(status=200)
        response.__enter__.return_value = response
        response.read1.side_effect = [b'zip', b'']
        error = urllib.error.HTTPError('https://api.github.com', 302, 'redirect',
                                      {'Location': 'https://owned.blob.core.windows.net/archive?sig=private'}, None)
        with mock.patch.object(api, 'request', side_effect=error):
            api.opener.open.return_value = response
            self.assertEqual(api.archive(1), b'zip')
        redirected = api.opener.open.call_args.args[0]
        self.assertIsNone(redirected.get_header('Authorization'))
        self.assertIsNone(P.NoRedirect().redirect_request(None, None, 302, '', {}, 'https://other'))

    def test_api_redirect_host_credentials_and_size_refusals(self):
        for location in ('http://owned.blob.core.windows.net/x', 'https://evil.test/x',
                         'https://user@owned.blob.core.windows.net/x'):
            api = P.GitHub('owned', 10**15)
            error = urllib.error.HTTPError('https://api.github.com', 302, 'redirect', {'Location': location}, None)
            with mock.patch.object(api, 'request', side_effect=error), self.assertRaises(ValueError): api.archive(1)

    def test_http_body_cap_and_deadline_refuse_before_unbounded_retention(self):
        self.assertEqual(P.bounded_response(io.BytesIO(b'1234'), 4, 10**15), b'1234')
        with self.assertRaises(ValueError): P.bounded_response(io.BytesIO(b'12345'), 4, 10**15)
        with self.assertRaises(ValueError): P.bounded_response(io.BytesIO(b'1'), 4, 0)

    def test_observe_cli_fixed_refusal_and_no_token_to_tool_children(self):
        with tempfile.TemporaryDirectory() as name:
            path = Path(name).resolve()/'metadata.json'
            P.PC.write_new(path, P.PC.encoded(raw()))
            with mock.patch.dict(os.environ, {'CQ_TAGGED_API_TOKEN': 'private'}):
                self.assertEqual(P.main(['observe', '--metadata', str(path), '--audit-exit', '1']), 0)
                self.assertNotIn('CQ_TAGGED_API_TOKEN', os.environ)
            error = io.StringIO()
            with contextlib.redirect_stderr(error):
                self.assertEqual(P.main(['observe', '--metadata', str(path), '--audit-exit', '2']), 2)
            self.assertEqual(error.getvalue(), P.REFUSAL+'\n')


    def test_build_selectors_reject_overlay_modfile_workspace_and_wrong_tool_before_probe(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve(); binary = root/'bin'/'go'
            binary.parent.mkdir(); binary.write_bytes(b'owned inert executable')
            environment = dict(P.BUILD, GOROOT=str(root))
            self.assertEqual(P.build_selection(binary, environment)['invocation'], 'absolute-selected-go')
            edits = [('GOFLAGS', '-overlay=other.json'), ('GOFLAGS', '-modfile=other.mod'),
                     ('GOFLAGS', '-run=Nothing'), ('GOENV', 'other.env'), ('GOWORK', 'other.work'),
                     ('GOROOT', str(root/'other')), ('GOOS', 'other'), ('GOEXPERIMENT', 'other')]
            for key, value in edits:
                changed = dict(environment); changed[key] = value
                with mock.patch.dict(os.environ, changed), \
                     mock.patch.object(P.PC, 'source_identity', side_effect=AssertionError('no source probe')), \
                     mock.patch.object(P.PC, 'tool_identity', side_effect=AssertionError('no tool probe')), \
                     self.subTest(key=key, value=value), self.assertRaises(ValueError):
                    P.identity(root, COMMIT, binary, root/'jar', 'linux')
            with self.assertRaises(ValueError): P.build_selection(Path('relative-go'), environment)

    def test_workflow_uses_selected_absolute_go_once_and_closed_job_environment(self):
        text = SCRIPT.parents[1].joinpath('workflows/integration.yml').read_text()
        self.assertEqual(text.count('"$CQ_TAGGED_GO" vet -tags=integration ./internal/integration'), 1)
        self.assertEqual(text.count('"$CQ_TAGGED_GO" test -p 1 -tags=integration ./internal/integration'), 1)
        self.assertNotIn('\n          go vet ', text)
        self.assertNotIn('\n            go test ', text)
        for name, value in P.BUILD.items():
            literal = name+': '+('local' if name == 'GOTOOLCHAIN' else "'"+value+"'")
            self.assertIn(literal, text)
        self.assertLess(text.index('source_gate.py tagged-begin'), text.index('"$CQ_TAGGED_GO" vet'))
        self.assertLess(text.index('"$CQ_TAGGED_GO" test'), text.index('source_gate.py tagged-retain'))

    def test_sdk_pending_original_discovery_does_not_wait_for_unrelated_matrix(self):
        api = mock.Mock(deadline=10**15)
        observed = run('sdk.yml'); observed.update(status='in_progress', conclusion=None)
        api.json.return_value = {'total_count': 1, 'workflow_runs': [observed]}
        self.assertEqual(P.discover(api, HEAD, 'pull_request', sleep=lambda _: self.fail(),
                                    workflow='sdk.yml', successful=False), observed)
        self.assertIsNone(observed['conclusion'])

    def test_sdk_selected_steps_wait_but_fail_closed_on_terminal_missing_or_failed(self):
        api = mock.Mock(deadline=10**15)
        observed = run('sdk.yml'); observed.update(status='in_progress', conclusion=None)
        job = {'id': 5, 'name': 'TypeScript SDK (Node 24.22.2, ubuntu-24.04)', 'run_id': 70,
               'run_attempt': 1, 'status': 'in_progress', 'steps': [
                   {'name': name, 'number': 15+i, 'conclusion': None} for i, name in enumerate(P.SDK_STEPS)]}
        artifact = {'id': 77, 'name': 'go-mcp-coverage-'+COMMIT, 'expired': False, 'workflow_run': {'id': 70}}
        api.json.side_effect = lambda path: (observed if path.endswith('/runs/70') else
            ({'total_count': 1, 'jobs': [job]} if '/jobs?' in path else {'total_count': 1, 'artifacts': [artifact]}))
        def complete(_):
            for step in job['steps']: step['conclusion'] = 'success'
        actual, _ = P.sdk_ready(api, observed, SOURCE, sleep=complete)
        self.assertEqual(actual['id'], 5); self.assertEqual(observed['status'], 'in_progress')
        job['steps'][0]['conclusion'] = 'failure'
        with self.assertRaises(ValueError): P.sdk_ready(api, observed, SOURCE, sleep=lambda _: self.fail())
        job['steps'] = []; job['status'] = 'completed'
        with self.assertRaises(ValueError): P.sdk_ready(api, observed, SOURCE, sleep=lambda _: self.fail())


class CompleteQualificationContracts(unittest.TestCase):
    setUp = Contracts.setUp
    def prepared(self, root, hit=1):
        source = root/'source'; source.mkdir()
        for name in P.LOCKS:
            target = source/name; target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(P.PC.encoded(MANIFEST) if name == P.LOCKS[0] else b'owned-lock\n')
        locks = {name: P.PC.digest((source/name).read_bytes()) for name in P.LOCKS}
        artifact_root = root/'archives'; artifact_root.mkdir()
        value = pins()
        for role in P.PC.ROLES:
            content = ('mode: atomic\n'+P.PC.MODULE+'owned.go:1.1,2.1 1 '+str(hit)+'\n').encode()
            metadata = {'version': 1, 'kind': 'platform-counters', 'role': role, 'source': SOURCE,
                'toolchain': tool(role), 'producer': producer(role),
                'build': {'go_flags': '', 'cgo_selector': 'default', 'scope': 'default-all-packages-no-tags'},
                'source_before_after_equal': True,
                'coverage': {'mode': 'atomic', 'bytes': len(content), 'sha256': P.PC.digest(content)},
                'test': {'package_pattern': './...', 'coverpkg': './...', 'count': 1,
                         'package_parallelism': 1, 'tags': [], 'exit_code': 0}, 'production_qualified': False}
            if role == 'source':
                metadata = dict(raw(), version=1, commit=COMMIT,
                    source={'commit': COMMIT, 'tree': TREE, 'manifest_sha256': '8'*64},
                    original_report_sha256='9'*64, coverage_sha256=P.PC.digest(content),
                    coverage_semantics='Original audit result; not recomputed or promoted by retention',
                    go_version=P.PC.GO_VERSION, platform='linux/amd64', go_tool_sha256=tool()['sha256'])
            encoded = P.PC.encoded(metadata)
            entry = value['entries'][role]
            entry['descriptor']['coverage'] = {'mode': 'atomic', 'bytes': len(content), 'sha256': P.PC.digest(content)}
            entry['descriptor'].update(metadata_bytes=len(encoded), metadata_sha256=P.PC.digest(encoded))
            data = archive({'coverage.out': content,
                            ('metadata.json' if role == 'source' else 'platform-metadata.json'): encoded})
            entry['archive_sha256'] = P.PC.digest(data)
            directory = artifact_root/role; directory.mkdir()
            P.PC.write_new(directory/'owned.zip', data)
        pin_path = root/'pins.json'; P.PC.write_new(pin_path, P.PC.encoded(value))
        output = root/'union'
        with mock.patch.object(P.PC, 'source_identity', return_value=(SOURCE, {'owned.go'})), contextlib.redirect_stdout(io.StringIO()):
            P.PC.union(source, COMMIT, artifact_root, pin_path, P.PC.digest(pin_path.read_bytes()), output)
        api = mock.Mock()
        receipts = {role: archive({'tagged.json': P.PC.encoded(receipt(role, locks))}) for role in P.ROLES}
        artifacts = [{'id': index+100, 'name': 'tagged-source-'+role+'-'+COMMIT, 'expired': False,
                      'workflow_run': {'id': 70}, 'digest': 'sha256:'+P.PC.digest(receipts[role]),
                      'size_in_bytes': len(receipts[role])} for index, role in enumerate(P.ROLES)]
        api.json.side_effect = lambda path: ({'total_count': 3, 'jobs': jobs()} if '/jobs?' in path else
                                             {'total_count': 3, 'artifacts': artifacts})
        api.archive.side_effect = lambda identifier: receipts[list(P.ROLES)[identifier-100]]
        needs = {name: {'result': 'success'} for name in (*P.PC.ROLES.values(), 'cross-build-gate')}
        return source, output, needs, api, pin_path, artifact_root

    def qualified(self, prepared):
        source, output, needs, api, pin_path, artifact_root = prepared
        environment = {'GITHUB_RUN_ID': '10', 'GITHUB_RUN_ATTEMPT': '1',
                       'GITHUB_EVENT_NAME': 'pull_request', 'CQ_COVERAGE_PR_HEAD': HEAD}
        with mock.patch.dict(os.environ, environment), \
             mock.patch.object(P.PC, 'source_identity', return_value=(SOURCE, {'owned.go'})), \
             mock.patch.object(P, 'tagged_scope', return_value={'owned': 'a'*64}), \
             mock.patch.object(P, 'discover', return_value=run()), \
             mock.patch.object(P, 'sdk_witness', return_value={'qualified_fixture': True}) as sdk:
            result = P.qualify(source, COMMIT, output, needs, None, api, pin_path, artifact_root)
            sdk.assert_called_once()
            return result

    def test_full_union_and_both_witness_families_resolve_only_declared_holds(self):
        with tempfile.TemporaryDirectory() as name:
            prepared = self.prepared(Path(name).resolve())
            raw_path = prepared[1]/'platform-union.json'; before = raw_path.read_bytes()
            result = self.qualified(prepared)
            self.assertTrue(result['qualified']); self.assertFalse(result['production_qualified'])
            self.assertFalse(result['release_qualified'])
            self.assertEqual(result['resolved_holds'], ['statement_coverage', 'tagged_integration'])
            self.assertEqual(set(result['tagged']), set(P.ROLES))
            self.assertEqual(raw_path.read_bytes(), before)

    def test_canonical_shortfall_never_promotes_even_when_other_witnesses_pass(self):
        with tempfile.TemporaryDirectory() as name:
            prepared = self.prepared(Path(name).resolve(), hit=0)
            with self.assertRaises(ValueError): self.qualified(prepared)

    def test_every_native_source_and_crossbuild_required_result_refuses(self):
        with tempfile.TemporaryDirectory() as name:
            prepared = self.prepared(Path(name).resolve())
            for key in prepared[2]:
                for status in ('failure', 'cancelled', 'skipped'):
                    prepared[2][key]['result'] = status
                    with self.subTest(key=key, status=status), self.assertRaises(ValueError): self.qualified(prepared)
                prepared[2][key]['result'] = 'success'

    def test_artifact_digest_pin_or_union_tampering_never_promotes(self):
        with tempfile.TemporaryDirectory() as name:
            prepared = self.prepared(Path(name).resolve())
            target = prepared[-1]/'linux'/'owned.zip'
            original = target.read_bytes(); target.write_bytes(original+b'extra')
            with self.assertRaises(ValueError): self.qualified(prepared)
            target.write_bytes(original)
            target = prepared[1]/'platform-union.json'
            value = P.read_json(target); value['coverage']['total_statements'] = 2
            target.write_bytes(P.PC.encoded(value))
            with self.assertRaises(ValueError): self.qualified(prepared)

    def test_sdk_original_producer_success_not_overall_npm_success(self):
        content = b'mode: atomic\nowned:1.1,2.1 1 1\n'
        value = {'version': 1, 'kind': 'sdk-go-mcp', 'source': SOURCE,
                 'toolchain': {key: tool()[key] for key in ('version', 'platform', 'sha256')},
                 'binary': {'sha256': '9'*64, 'bytes': 123},
                 'producer': {'repository': P.REPO, 'run_id': '70', 'attempt': 1, 'job': 'typescript',
                              'matrix': 'ubuntu-24.04/node24.22.2', 'pr_head': HEAD, 'step': 'official_go_mcp_conformance'},
                 'test': {'package': P.GM.PACKAGE, 'name': P.GM.TEST, 'tags': 'integration', 'count': 1, 'exit_code': 0},
                 'coverage': {'mode': 'atomic', 'bytes': len(content), 'sha256': P.PC.digest(content)},
                 'production_qualified': False, 'source_before_after_equal': True, 'preflight_sha256': '8'*64}
        names = ['Exercise the official MCP TypeScript client against synthetic cq', 'Retain SDK-specific official Go MCP coverage']
        job = {'name': 'TypeScript SDK (Node 24.22.2, ubuntu-24.04)', 'id': 5, 'run_id': 70, 'run_attempt': 1,
               'steps': [{'name': name, 'number': 15+i, 'conclusion': 'success'} for i, name in enumerate(names)]}
        api = mock.Mock()
        def configure():
            data = archive({'coverage.out': content, 'sdk-metadata.json': P.PC.encoded(value)})
            artifact = {'id': 77, 'name': 'go-mcp-coverage-'+COMMIT, 'expired': False,
                        'workflow_run': {'id': 70}, 'size_in_bytes': len(data), 'digest': 'sha256:'+P.PC.digest(data)}
            api.archive.return_value = data
            api.json.side_effect = lambda path: (observed if path.endswith('/runs/70') else
                ({'total_count': 1, 'jobs': [job]} if '/jobs?' in path else
                 {'total_count': 1, 'artifacts': [artifact]}))
        observed = run('sdk.yml'); observed['conclusion'] = 'failure'
        configure()
        with mock.patch.object(P, 'discover', return_value=observed):
            result = P.sdk_witness(api, Path('/unused'), SOURCE, producer('source'), tool())
            self.assertEqual(result['overall_sdk_conclusion'], 'failure')
            self.assertIn('assertion', result['binary_provenance'])
            for code in (1, 125, False):
                value['test']['exit_code'] = code; configure()
                with self.assertRaises(ValueError): P.sdk_witness(api, Path('/unused'), SOURCE, producer('source'), tool())
            value['test']['exit_code'] = 0; configure()
            job['steps'][0]['conclusion'] = 'skipped'
            with self.assertRaises(ValueError): P.sdk_witness(api, Path('/unused'), SOURCE, producer('source'), tool())


if __name__ == '__main__':
    unittest.main()
