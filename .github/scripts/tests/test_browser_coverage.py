"""Offline owned-input oracles. No Node/Git/test-suite/provider command executes; capture oracles use only owned Python children."""
import contextlib
import copy
import hashlib
import importlib.util
import io
import os
import runpy
import sys
from pathlib import Path
import stat
import tempfile
import unittest
from unittest import mock
import zipfile

SCRIPT = Path(__file__).resolve().parents[1] / 'browser_coverage.py'
SPEC = importlib.util.spec_from_file_location('browser_counter_contract', SCRIPT)
P = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(P)
COMMIT, TREE, HEAD = 'a' * 40, 'b' * 40, 'c' * 40
FILES = {'web/owned.js': {'sha256': 'd' * 64, 'bytes': 20, 'units': 20, 'starts': [0, 5, 10, 15], 'ends': [4, 9, 14, 19]}}
SOURCE = {'commit': COMMIT, 'tree': TREE, 'inventory_sha256': P.digest(P.encoded(P.public_inventory(FILES))),
          'package_sha256': 'e' * 64, 'implementation_sha256': 'f' * 64}


def node(role):
    return {'version': P.NODE_VERSION, 'platform': P.PLATFORMS['linux' if role == 'baseline' else role],
            'arch': 'x64', 'sha256': P.digest(role.encode()), 'bytes': 10}


def producer(role):
    return {'repository': P.REPOSITORY, 'run': '10' if role == 'baseline' else '20', 'attempt': '1',
            'job': 'sonar' if role == 'baseline' else 'browser', 'event': 'pull_request', 'head': HEAD,
            'platform': 'linux' if role == 'baseline' else role,
            'suite': 'full-browser' if role == 'baseline' else 'web', 'node': P.NODE_VERSION}


def report_script(hit):
    return {'path': 'web/owned.js', 'instance': '1' * 64, 'functions': [
        {'kind': 'script', 'name_sha256': P.digest(b''), 'anonymous': True, 'block': True,
         'ranges': [[0, 20, 1]]},
        {'kind': 'function', 'name_sha256': P.digest(b'owned'), 'anonymous': False, 'block': True,
         'ranges': [[5, 15, 1], [5, 10, hit]]}]}


def key(kind='function', start=5, end=15, block=True):
    return (kind, P.digest(b'' if kind == 'script' else b'owned'), start, end, block)


def raw_fixture(hit):
    return {'scripts': [report_script(hit)], 'observations': [[report_script(hit)]],
            'raw': {'web/owned.js': {key('script', 0, 20): {(0, 20): 1},
                                      key(): {(5, 15): 1, (5, 10): hit}}}}


def counter(hit):
    return {'schema': 'cq.native-browser.counters/3', 'identity': 'source-utf16-block-ranges',
            'reporter': 'node-v24.22.2-exact-original', 'observations': [[report_script(hit)]],
            'scripts': [report_script(hit)],
            'files': {'web/owned.js': {'source': P.public_inventory(FILES)['web/owned.js'],
                                     'lines': [[1, 1], [2, hit], [3, 1], [4, 1]], 'functions': [
                                         {'kind': 'script', 'name_sha256': P.digest(b''), 'block': True,
                                          'start': 0, 'end': 20, 'ranges': [[0, 20, 1]]},
                                         {'kind': 'function', 'name_sha256': P.digest(b'owned'), 'block': True,
                                          'start': 5, 'end': 15, 'ranges': [[5, 10, hit], [5, 15, 1]]}]}}}


def lcov(hit, role='baseline'):
    name = 'web/owned.js' if role == 'baseline' else 'owned.js'
    return (f'TN:\nSF:{name}\nFN:2,owned\nFNDA:1,owned\nFNF:1\nFNH:1\n'
            f'BRDA:1,0,0,1\nBRDA:2,1,0,1\nBRDA:2,2,0,{hit}\nBRF:3\nBRH:{2 + int(hit > 0)}\n'
            f'DA:1,1\nDA:2,{hit}\nDA:3,1\nDA:4,1\nLF:4\nLH:{3 + int(hit > 0)}\nend_of_record\n').encode()


def zip_members(members):
    output = io.BytesIO()
    with zipfile.ZipFile(output, 'w') as archive:
        for name, data in members.items():
            archive.writestr(name, data)
    return output.getvalue()


def archive(role, hit):
    original = lcov(hit, role)
    counts = P.encoded(counter(hit))
    tool = node(role)
    metadata = {'schema': 'cq.native-browser.retained/1', 'source_before': SOURCE, 'source_after': SOURCE,
                'node_before': tool, 'node_after': tool, 'producer': producer(role), 'suite_exit': 0,
                'files': P.public_inventory(FILES), 'lcov': {'sha256': P.digest(original), 'bytes': len(original)},
                'counters': {'sha256': P.digest(counts), 'bytes': len(counts)},
                'original_check_passed': True, 'global_release_qualified': False}
    encoded_metadata = P.encoded(metadata)
    data = zip_members({'lcov.info': original, 'counters.json': counts, 'metadata.json': encoded_metadata})
    entry = {'archive_sha256': P.digest(data), 'metadata_sha256': P.digest(encoded_metadata),
             'lcov_sha256': P.digest(original), 'counters_sha256': P.digest(counts),
             'producer': producer(role), 'source_commit': COMMIT, 'node_sha256': tool['sha256'],
             'artifact_id': str(100 + ['baseline', 'linux', 'macos', 'windows'].index(role)),
             'run_id': producer(role)['run'], 'job_id': str(200 + ['baseline', 'linux', 'macos', 'windows'].index(role))}
    if role == 'baseline':
        entry = {key: value for key, value in entry.items() if key not in {'archive_sha256', 'artifact_id'}}
        entry['kind'] = 'owned-current-sonar'
        data = {'lcov.info': original, 'counters.json': counts, 'metadata.json': encoded_metadata}
    return data, entry, metadata


def fixture(hit=1):
    values = {role: archive(role, hit if role == 'windows' else 0)
              for role in ('baseline', 'linux', 'macos', 'windows')}
    pins = {'schema': 'cq.native-browser.authorized-pins/2', 'head': HEAD, 'tree': TREE,
            'current_commit': COMMIT, 'source_hashes': {key: SOURCE[key] for key in
                                                       ('package_sha256', 'implementation_sha256')},
            'entries': {role: value[1] for role, value in values.items()}}
    return values, pins


class GrammarContracts(unittest.TestCase):
    def test_strict_json_rejects_duplicate_nonfinite_invalid_utf8_and_deep_data(self):
        self.assertEqual(P.strict_json(b'{"public":1}'), {'public': 1})
        for data in (b'{"a":1,"a":2}', b'NaN', b'Infinity', b'\xff', b'{', b'[' * 1200):
            with self.subTest(data=data[:30]), self.assertRaises(P.InvalidEvidence):
                P.strict_json(data)

    def test_scalar_hash_key_integer_fences_are_closed(self):
        for value in (True, -1, 1.5, '1', P.MAX_INT + 1):
            with self.assertRaises(P.InvalidEvidence): P.integer(value)
        for value in ('A' * 64, 'a' * 63, None, 1):
            with self.assertRaises(P.InvalidEvidence): P.hexadecimal(value)
        for value in (None, [], {'extra': 1}):
            with self.assertRaises(P.InvalidEvidence): P.keys(value, ('owned',))
        self.assertEqual(P.decimal('0'), 0)
        for value in ('-1', '+1', '1.0', '1e3', '9' * 17):
            with self.assertRaises(P.InvalidEvidence): P.decimal(value)

    def test_path_normalization_is_bounded_repo_relative_not_absolute_or_drive_alias(self):
        self.assertEqual(P.relative_lcov_path('web/../sdk/test/verify.mjs'), 'sdk/test/verify.mjs')
        self.assertEqual(P.relative_lcov_path('web\\owned.js'), 'web/owned.js')
        for value in ('/outside', 'C:/outside', '../outside', 'web//owned', '', 'web/\0'):
            with self.assertRaises(P.InvalidEvidence): P.relative_lcov_path(value)

    def test_lcov_preserves_duplicate_records_and_zero_counters(self):
        data = lcov(0) + lcov(1)
        records = P.parse_lcov(data, FILES)
        self.assertEqual(len(records), 2)
        self.assertEqual(records[0]['lines'][2], 0)
        self.assertEqual(records[1]['lines'][2], 1)
        self.assertEqual(P.parse_lcov(lcov(0, 'linux'), FILES, 'web/')[0]['path'], 'web/owned.js')

    def test_every_lcov_summary_and_coordinate_refusal_is_not_truncated_success(self):
        original = lcov(0)
        changes = [lambda x: b'', lambda x: x[:-14], lambda x: x.replace(b'LF:4', b'LF:5'),
                   lambda x: x.replace(b'LH:3', b'LH:4'), lambda x: x.replace(b'FNF:1', b'FNF:0'),
                   lambda x: x.replace(b'FNH:1', b'FNH:0'), lambda x: x.replace(b'BRF:3', b'BRF:0'),
                   lambda x: x.replace(b'BRH:2', b'BRH:3'), lambda x: x.replace(b'DA:2,0', b'DA:2,-1'),
                   lambda x: x.replace(b'DA:2,0', b'DA:2,0\nDA:2,1'),
                   lambda x: x.replace(b'FNDA:1,owned', b'FNDA:1,other'),
                   lambda x: x.replace(b'BRDA:2,2,0,0', b'BRDA:99,2,0,0'),
                   lambda x: x.replace(b'SF:web/owned.js', b'SF:unknown.js'),
                   lambda x: x.replace(b'DA:2,0', b'UNKNOWN:2,0'), lambda x: x.replace(b'TN:', b'TN:private')]
        for edit in changes:
            with self.subTest(edit=edit), self.assertRaises((P.InvalidEvidence, ValueError)):
                P.parse_lcov(edit(original), FILES)

    def test_native_unknown_branch_marker_is_a_zero_not_a_hit(self):
        data = lcov(0).replace(b'BRDA:2,2,0,0', b'BRDA:2,2,0,-')
        self.assertEqual(P.parse_lcov(data, FILES)[0]['branches'][(2, 2, 0)], 0)

    def test_nested_utf16_ranges_refuse_crossing_duplicate_bool_and_out_of_source_offsets(self):
        accepted = [{'startOffset': 0, 'endOffset': 20, 'count': 1},
                    {'startOffset': 5, 'endOffset': 10, 'count': 0}]
        self.assertEqual(P.checked_ranges(accepted, 20), [[0, 20, 1], [5, 10, 0]])
        for edit in (lambda x: x.append(x[1]), lambda x: x[1].update(count=True),
                     lambda x: x[1].update(endOffset=21), lambda x: x[1].update(startOffset=-1),
                     lambda x: x.extend([{'startOffset': 6, 'endOffset': 12, 'count': 1}])):
            values = copy.deepcopy(accepted); edit(values)
            with self.assertRaises(P.InvalidEvidence): P.checked_ranges(values, 20)

    def test_sanitized_counter_schema_rejects_missing_or_mismatched_source_and_range_identity(self):
        for edit in (lambda x: x.update(identity='opaque-lcov-IDs'),
                     lambda x: x['files']['web/owned.js']['source'].update(sha256='0' * 64),
                     lambda x: x['files']['web/owned.js']['lines'].append([2, 1]),
                     lambda x: x['files']['web/owned.js']['functions'][1]['ranges'].append([5, 10, 1]),
                     lambda x: x['files']['web/owned.js']['functions'][1].update(kind='native'),
                     lambda x: x['files']['web/owned.js']['functions'][1].update(ranges=[])):
            value = counter(0); edit(value)
            with self.assertRaises(P.InvalidEvidence): P.validated_counters(value, FILES)


class CounterMathContracts(unittest.TestCase):
    def test_exact_offset_max_union_keeps_original_zero_and_all_platform_only_coordinates(self):
        a = P.validated_counters(counter(0), FILES)
        b = P.validated_counters(counter(1), FILES)
        b['web/owned.js']['functions'][key()][(9, 10)] = 0
        combined = P.merge_counters([a, b, a])
        self.assertEqual(combined['web/owned.js']['functions'][key()],
                         {(5, 10): 1, (5, 15): 1, (9, 10): 0})
        self.assertEqual(combined['web/owned.js']['lines'], {1: 1, 2: 1, 3: 1, 4: 1})
        data, totals, gaps = P.render_composite(combined, FILES)
        self.assertEqual(totals['branches'], 4)
        self.assertEqual(totals['branches_hit'], 3)
        self.assertEqual(gaps, [{'path': 'web/owned.js', 'kind': 'block-range',
                                 'function': list(key()), 'start': 9, 'end': 10}])
        self.assertIn(b'BRH:3\n', data)
        self.assertNotIn('anonymous_', data.decode())

    def test_no_parent_hit_inheritance_fills_a_missing_child_range(self):
        a = P.validated_counters(counter(0), FILES)
        b = P.validated_counters(counter(1), FILES)
        del b['web/owned.js']['functions'][key()][(5, 10)]
        combined = P.merge_counters([a, b])
        self.assertEqual(combined['web/owned.js']['functions'][key()][(5, 10)], 0)
        _, totals, _ = P.render_composite(combined, FILES)
        self.assertEqual(totals['branches_hit'], 2)
        self.assertEqual(totals['branches'], 3)

    def test_raw_distinct_function_extent_is_never_aliased_by_opaque_reporter_number(self):
        a = P.validated_counters(counter(0), FILES)
        b = copy.deepcopy(a)
        b['web/owned.js']['functions'][key(start=6)] = {(6, 15): 1, (7, 8): 1}
        combined = P.merge_counters([a, b])
        self.assertIn(key(), combined['web/owned.js']['functions'])
        self.assertIn(key(start=6), combined['web/owned.js']['functions'])
        self.assertEqual(combined['web/owned.js']['functions'][key()][(5, 10)], 0)

    def test_first_party_source_paths_are_no_query_or_computed_name_disclosure(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve()
            files = {'web/owned space%.js': FILES['web/owned.js']}
            url = (root / 'web/owned space%.js').as_uri() + '?owned-instance=private-marker'
            self.assertEqual(P.source_path(url, root, files), 'web/owned space%.js')
            self.assertIsNone(P.source_path('https://provider.invalid/private', root, files))
            self.assertIsNone(P.source_path((root.parent / 'outside.js').as_uri(), root, files))
            self.assertIsNone(P.source_path('node:internal/private', root, files))

    def test_raw_input_filters_nonfirstparty_urls_and_removes_all_runtime_names_and_source_maps(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve(); raw = root / 'raw'; raw.mkdir()
            document = {'result': [{'scriptId': '1', 'url': (root / 'web/owned.js').as_uri() + '?PRIVATE',
                                   'functions': [{'functionName': 'PRIVATE_RUNTIME_NAME', 'isBlockCoverage': True,
                                                  'ranges': [{'startOffset': 0, 'endOffset': 20, 'count': 1}]},
                                                 {'functionName': 'owned', 'isBlockCoverage': True,
                                                  'ranges': [{'startOffset': 5, 'endOffset': 15, 'count': 1},
                                                             {'startOffset': 5, 'endOffset': 10, 'count': 0}]}]},
                                  {'scriptId': '2', 'url': 'node:private', 'functions': []}],
                        'source-map-cache': {'PRIVATE': 'not retained'}}
            (raw / 'coverage-1-1700000000000-0.json').write_bytes(P.encoded(document))
            counts = P.raw_counters(raw, root, FILES)
            value = P.canonical_counters(counts, P.parse_lcov(lcov(0), FILES), FILES)
            self.assertNotIn('PRIVATE', P.encoded(value).decode())
            self.assertEqual(set(value['files']), {'web/owned.js'})

    def test_raw_directory_and_record_caps_refuse_nonreports_and_empty_range_sources(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve()
            with self.assertRaises(P.InvalidEvidence): P.raw_counters(root, root, FILES)
            (root / 'not-coverage.json').write_text('{}')
            with self.assertRaises(P.InvalidEvidence): P.raw_counters(root, root, FILES)


class PinnedNodeProjectionContracts(unittest.TestCase):
    def test_exact_node_summary_agrees_with_literal_original_all_four_counter_kinds(self):
        records = P.parse_lcov(lcov(0), FILES)
        P.verify_reporter([report_script(0)], records, FILES)
        self.assertEqual(P.node_record(report_script(0), FILES['web/owned.js'])['lines'],
                         {1: 1, 2: 0, 3: 1, 4: 1})
        for field in ('lines', 'branches', 'fn', 'fnda', 'summary'):
            altered = copy.deepcopy(records)
            if field == 'lines': altered[0][field][2] = 1
            elif field == 'branches': altered[0][field][(2, 2, 0)] = 1
            elif field == 'fn': altered[0][field][0] = (3, 'owned')
            elif field == 'fnda': altered[0][field][0] = (7, 'owned')
            else: altered[0][field]['BRF'] = 4
            with self.subTest(field=field), self.assertRaises(P.InvalidEvidence):
                P.verify_reporter([report_script(0)], altered, FILES)

    def test_function_only_zero_is_not_branch_but_remains_a_function_gap(self):
        value = counter(0)
        value['files']['web/owned.js']['functions'][1].update(block=False, ranges=[[5, 15, 0]])
        profile = P.validated_counters(value, FILES)
        data, totals, gaps = P.render_composite(profile, FILES)
        self.assertEqual(totals['branches'], 1)  # script block only, no invented function condition
        self.assertEqual(totals['functions'], 1); self.assertEqual(totals['functions_hit'], 0)
        self.assertEqual(gaps, [{'path': 'web/owned.js', 'kind': 'function',
                                 'function': list(key()[:4])},
                                {'path': 'web/owned.js', 'kind': 'line', 'line': 2}])
        self.assertIn(b'BRF:1\n', data)

    def test_coarse_hit_cannot_fill_precise_zero_and_both_granularities_are_retained(self):
        precise = P.validated_counters(counter(0), FILES)
        coarse = copy.deepcopy(precise)
        coarse['web/owned.js']['functions'] = {key(block=False): {(5, 15): 99}}
        merged = P.merge_counters([precise, coarse])
        self.assertIn(key(block=False), merged['web/owned.js']['functions'])
        self.assertEqual(merged['web/owned.js']['functions'][key()][(5, 10)], 0)
        _, totals, gaps = P.render_composite(merged, FILES)
        self.assertEqual(totals['branches'], 3); self.assertEqual(totals['branches_hit'], 2)
        self.assertTrue(any(row.get('start') == 5 and row.get('end') == 10 for row in gaps))

    def test_node_native_merge_replaces_coarse_with_block_and_keeps_ordered_zero_intersection(self):
        old = report_script(0)['functions']
        old[1]['block'] = False; old[1]['ranges'] = [[5, 15, 0]]
        P.merge_node_functions(old, report_script(0)['functions'])
        self.assertTrue(old[1]['block']); self.assertEqual(old[1]['ranges'], [[5, 15, 1], [5, 10, 0]])
        merged = P.merge_node_ranges([[0, 20, 1], [2, 12, 0]], [[0, 20, 1], [5, 10, 0]])
        self.assertEqual(merged, [[0, 20, 2], [5, 10, 0]])
        # Native reporter merge is sum; cross-platform evidence union is max.
        self.assertEqual(P.merge_node_ranges([[0, 20, 2]], [[0, 20, 3]]), [[0, 20, 5]])

    def test_query_instances_and_anonymous_labels_match_original_multiplicity_not_opaque_ids(self):
        script = report_script(0)
        script['functions'][1].update(name_sha256=P.digest(b''), anonymous=True)
        original = lcov(0).replace(b',owned', b',anonymous_0')
        P.verify_reporter([script], P.parse_lcov(original, FILES), FILES)
        another = copy.deepcopy(script); another['instance'] = '2' * 64
        P.verify_reporter([script, another], P.parse_lcov(original + original, FILES), FILES)
        with self.assertRaises(P.InvalidEvidence):
            P.verify_reporter([script, another], P.parse_lcov(original, FILES), FILES)

    def test_sonar_int32_boundary_never_changes_exact_canonical_count_or_silently_drops_it(self):
        self.assertEqual(P.sonar_hit(0), 0)
        self.assertEqual(P.sonar_hit((1 << 31) - 1), (1 << 31) - 1)
        for count in (1 << 31, P.MAX_INT):
            value = counter(0)
            value['files']['web/owned.js']['lines'][1][1] = count
            profile = P.validated_counters(value, FILES)
            self.assertEqual(profile['web/owned.js']['lines'][2], count)
            with self.assertRaises(P.InvalidEvidence): P.render_composite(profile, FILES)
            value = counter(0)
            value['files']['web/owned.js']['functions'][0]['ranges'][0][2] = count
            profile = P.validated_counters(value, FILES)
            self.assertEqual(profile['web/owned.js']['functions'][key('script', 0, 20)][(0, 20)], count)
            with self.assertRaises(P.InvalidEvidence): P.render_composite(profile, FILES)

    def test_script_schema_refuses_lost_granularity_names_instance_and_source_offsets(self):
        changes = [lambda x: x[0].update(instance='PRIVATE'),
                   lambda x: x[0]['functions'][0].update(block=1),
                   lambda x: x[0]['functions'][0].pop('block'),
                   lambda x: x[0]['functions'][0].update(name_sha256='PRIVATE'),
                   lambda x: x[0]['functions'][0]['ranges'].append([0, 20, 1]),
                   lambda x: x[0]['functions'][0]['ranges'][0].__setitem__(1, 21)]
        for edit in changes:
            scripts = [report_script(0)]; edit(scripts)
            with self.assertRaises(P.InvalidEvidence): P.validated_scripts(scripts, FILES)

    def test_utf16_partial_line_never_inherits_the_enclosing_block_hit(self):
        counts = {1: 0, 2: 0, 3: 0, 4: 0}
        self.assertEqual(P.mapped_lines(FILES['web/owned.js'], [0, 20, 1], counts), 1)
        self.assertEqual(P.mapped_lines(FILES['web/owned.js'], [6, 8, 0], counts), 2)
        self.assertEqual(counts[2], 1)  # Node DA requires whole-line containment
        self.assertEqual(P.mapped_lines(FILES['web/owned.js'], [5, 10, 0], counts), 2)
        self.assertEqual(counts[2], 0)  # explicit precise whole-line zero overrides root


class AuthenticatedArchiveContracts(unittest.TestCase):
    def run_union(self, hit=1, change=None):
        values, pins = fixture(hit)
        if change: change(values, pins)
        data = P.encoded(pins)
        with mock.patch.object(P, 'source_identity', return_value=(SOURCE, FILES)), \
             mock.patch.object(P, 'node_identity', return_value=node('baseline')), \
             mock.patch.object(P, 'producer_identity', return_value=producer('baseline')):
            return P.union(Path('/owned'), COMMIT, data, P.digest(data),
                           {role: value[0] for role, value in values.items()}, '/owned/node', {})

    def test_full_pinned_composite_is_distinct_and_never_promotes_global_product_or_source(self):
        data, result, gaps, code = self.run_union()
        self.assertEqual(code, 0); self.assertTrue(result['coverage_passed'])
        self.assertTrue(result['evidence_accepted']); self.assertFalse(result['global_release_qualified'])
        self.assertFalse(result['production_promoted']); self.assertFalse(result['original_lcov_rewritten'])
        self.assertEqual(result['totals']['branches'], 3); self.assertEqual(gaps, [])
        self.assertEqual(result['composite']['sha256'], P.digest(data))

    def test_true_shortfall_returns_one_preserves_denominator_and_every_original_report(self):
        _, result, gaps, code = self.run_union(hit=0)
        self.assertEqual(code, 1); self.assertFalse(result['coverage_passed'])
        self.assertEqual(result['totals']['lines'], 4); self.assertEqual(result['totals']['branches'], 3)
        self.assertEqual(len(result['originals']), 4); self.assertTrue(gaps)

    def test_unmeasured_firstparty_source_never_becomes_implicit_zero_or_green(self):
        files = {**FILES, 'web/not-loaded.js': FILES['web/owned.js']}
        with mock.patch.object(P, 'source_identity', return_value=(SOURCE, files)):
            values, pins = fixture()
            with self.assertRaises(P.InvalidEvidence):
                data = P.encoded(pins)
                P.union(Path('/owned'), COMMIT, data, P.digest(data), {k: v[0] for k, v in values.items()})

    def test_every_currentness_pins_hash_schema_source_and_role_boundary_refuses(self):
        edits = [lambda v, p: p.update(schema='other'), lambda v, p: p.update(current_commit='0' * 40),
                 lambda v, p: p.update(tree='0' * 40), lambda v, p: p['entries'].pop('windows'),
                 lambda v, p: p['source_hashes'].update(package_sha256='0' * 64),
                 lambda v, p: p['entries']['linux'].update(archive_sha256='0' * 64),
                 lambda v, p: p['entries']['macos'].update(counters_sha256='0' * 64),
                 lambda v, p: p['entries']['windows'].update(node_sha256='0' * 64),
                 lambda v, p: p['entries']['windows']['producer'].update(head='0' * 40),
                 lambda v, p: p['entries']['macos'].update(source_commit='0' * 40)]
        for edit in edits:
            with self.subTest(edit=edit), self.assertRaises(P.InvalidEvidence): self.run_union(change=edit)

    def test_zip_exact_members_no_symlink_duplicates_traversal_or_parser_message_leak(self):
        for data in (b'not ZIP', zip_members({'../private': b'PRIVATE'}), zip_members({'lcov.info': b'PRIVATE'})):
            with self.assertRaisesRegex(P.InvalidEvidence, '^' + P.REFUSAL): P.read_archive(data)
        data, _, _ = archive('linux', 0)
        altered = bytearray(data)
        # Keep allowed central names, replace only the first equal-length local
        # filename. CPython's raw BadZipFile contains it; our refusal must not.
        start = altered.find(b'lcov.info'); altered[start:start + 9] = b'PRIVATE!!'
        with self.assertRaises(P.InvalidEvidence) as caught: P.read_archive(bytes(altered))
        self.assertNotIn('PRIVATE', str(caught.exception)); self.assertIsNone(caught.exception.__cause__)
        duplicate = io.BytesIO()
        with contextlib.redirect_stderr(io.StringIO()), zipfile.ZipFile(duplicate, 'w') as z:
            for key in ('lcov.info', 'lcov.info', 'metadata.json'): z.writestr(key, b'{}')
        with self.assertRaises(P.InvalidEvidence): P.read_archive(duplicate.getvalue())
        link = io.BytesIO()
        with zipfile.ZipFile(link, 'w') as archive:
            for name in ('lcov.info', 'counters.json', 'metadata.json'):
                item = zipfile.ZipInfo(name)
                item.create_system = 3
                item.external_attr = ((stat.S_IFLNK | 0o777) << 16)
                archive.writestr(item, b'not-followed')
        with self.assertRaises(P.InvalidEvidence): P.read_archive(link.getvalue())



class ProducerAndOwnedIOContracts(unittest.TestCase):
    def test_owned_regular_read_and_exclusive_write_never_overwrite_or_follow_a_directory(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve(); path = root / 'owned'
            P.exclusive_write(path, b'owned'); self.assertEqual(P.private_read(path, 5), b'owned')
            with self.assertRaises(FileExistsError): P.exclusive_write(path, b'other')
            with self.assertRaises(P.InvalidEvidence): P.private_read(path, 4)
            with self.assertRaises(P.InvalidEvidence): P.private_read(root, 10)
            self.assertEqual(path.read_bytes(), b'owned')

    def test_begin_fixed_output_and_retain_preserve_actual_nonzero_suite_exit_without_running_it(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve(); output = root / 'cq-browser-web-linux'
            environment = {'RUNNER_TEMP': str(root), 'GITHUB_REPOSITORY': P.REPOSITORY,
                           'GITHUB_RUN_ID': '20', 'GITHUB_RUN_ATTEMPT': '1', 'GITHUB_JOB': 'browser',
                           'GITHUB_EVENT_NAME': 'pull_request', 'CQ_PR_HEAD_SHA': HEAD}
            raw = raw_fixture(0)
            with mock.patch.dict(os.environ, {'NODE_OPTIONS': ''}), \
                 mock.patch.object(P, 'source_identity', return_value=(SOURCE, FILES)), \
                 mock.patch.object(P, 'node_identity', return_value=node('linux')):
                P.begin(root, output, COMMIT, '/owned/node', 'linux', 'web', environment)
                (output / 'lcov.info').write_bytes(lcov(0, 'linux'))
                with mock.patch.object(P, 'raw_counters', return_value=raw):
                    metadata = P.retain(root, output, COMMIT, '/owned/node', 'linux', 'web', 7, environment)
                self.assertEqual(metadata['suite_exit'], 7); self.assertFalse(metadata['original_check_passed'])
                self.assertFalse(metadata['global_release_qualified'])
                self.assertTrue((output / 'metadata.json').is_file())
                with self.assertRaises(P.InvalidEvidence): P.begin(root, output, COMMIT, '/owned/node', 'linux', 'web', environment)
            with self.assertRaises(P.InvalidEvidence): P.fixed_output(root / 'arbitrary', 'linux', 'web', environment)

    def test_metadata_and_node_probe_reject_changed_tool_platform_and_producer_not_use_a_fallback(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve(); tool = root / 'node'; tool.write_bytes(b'owned')
            probe = {'version': P.NODE_VERSION, 'platform': 'linux', 'arch': 'x64', 'execPath': str(tool)}
            with mock.patch.object(P, 'command', return_value=P.encoded(probe)) as run:
                self.assertEqual(P.node_identity(root, tool, 'linux')['sha256'], P.digest(b'owned'))
                self.assertEqual(run.call_count, 1)
            for field, bad in [('version', 'v20.0.0'), ('platform', 'win32'), ('arch', 'unknown'), ('execPath', str(root / 'other'))]:
                changed = {**probe, field: bad}
                with mock.patch.object(P, 'command', return_value=P.encoded(changed)), self.assertRaises(P.InvalidEvidence):
                    P.node_identity(root, tool, 'linux')
            with self.assertRaises(P.InvalidEvidence): P.producer_identity({}, 'linux', 'web')

    def test_main_failure_is_constant_exit_two_not_raw_path_error_or_success(self):
        with contextlib.redirect_stderr(io.StringIO()) as error, mock.patch.object(P, 'begin', side_effect=OSError('PRIVATE_DATA')):
            self.assertEqual(P.main(['begin', '--source', '/owned', '--output', '/owned-output', '--expected-commit', COMMIT]), 2)
        self.assertEqual(error.getvalue(), P.REFUSAL + '\n')


class CompleteBoundaryContracts(unittest.TestCase):
    def test_command_success_is_bounded_and_removes_only_probe_coverage_options(self):
        with mock.patch.object(P, 'bounded_probe', return_value=b'owned') as run, \
             mock.patch.dict(os.environ, {'NODE_OPTIONS': 'PRIVATE', 'NODE_V8_COVERAGE': 'PRIVATE'}):
            self.assertEqual(P.command(['owned-fixed-command'], Path('/owned')), b'owned')
        self.assertNotIn('NODE_OPTIONS', run.call_args.args[2])
        self.assertNotIn('NODE_V8_COVERAGE', run.call_args.args[2])
        self.assertEqual(run.call_args.args[3], P.MAX_METADATA)

    def test_probe_owned_child_exact_cap_nonzero_stderr_and_acquisition_overflow_refuse(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve()
            def run(body, limit=4):
                return P.bounded_probe([sys.executable, '-I', '-c', body], root, dict(os.environ), limit)
            self.assertEqual(run('import os;os.write(1,b"ABCD")'), b'ABCD')
            cases = ['import os;os.write(1,b"ABCDE")',
                     'import os;os.write(2,b"PRIVATE")',
                     'import sys;sys.exit(7)',
                     'import os;os.write(2,b"X"*131072)']
            for body in cases:
                with self.subTest(body=body), self.assertRaises(P.InvalidEvidence) as caught:
                    run(body)
                self.assertEqual(str(caught.exception), P.REFUSAL)

    def test_probe_deadline_reaps_owned_child_and_never_returns_partial_output(self):
        with tempfile.TemporaryDirectory() as name:
            with mock.patch.object(P, 'stop_probe', wraps=P.stop_probe) as stop, self.assertRaises(P.InvalidEvidence):
                P.bounded_probe([sys.executable, '-I', '-c', 'import time;time.sleep(5)'],
                                Path(name), dict(os.environ), 100, deadline=0.05)
            stop.assert_called_once()
            self.assertIsNotNone(stop.call_args.args[0].poll())

    def test_capture_stream_retains_no_more_than_cap_and_proves_eof_separately(self):
        state = {'bytes': 0, 'parts': [], 'eof': False, 'error': False}
        event = P.threading.Event()
        P.capture_stream(io.BytesIO(b'ABCDE'), 4, state, event)
        self.assertTrue(event.is_set()); self.assertFalse(state['eof'])
        self.assertLessEqual(sum(map(len, state['parts'])), 4)
        self.assertEqual(state['bytes'], 5)


    def test_inventory_holds_exact_git_blob_bytes_and_utf16_source_offsets(self):
        data = 'x😀\ny\n'.encode()
        blob = hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()
        listing = f'100644 blob {blob}\tweb/owned.js\0'.encode()
        with mock.patch.object(P, 'git', return_value=listing), mock.patch.object(P, 'private_read', return_value=data):
            files = P.inventory(Path('/owned'))
        self.assertEqual(files['web/owned.js']['units'], 6)
        self.assertEqual(files['web/owned.js']['sha256'], P.digest(data))
        self.assertNotIn('starts', P.public_inventory(files)['web/owned.js'])
        for changed in (listing.replace(b'100644', b'120000'), listing.replace(blob.encode(), b'0' * 40)):
            with mock.patch.object(P, 'git', return_value=changed), mock.patch.object(P, 'private_read', return_value=data), self.assertRaises(P.InvalidEvidence):
                P.inventory(Path('/owned'))
        for name in ('web/owned.test.mjs', 'examples/landing-motion/.local/worker-routes.mjs', 'owned.py'):
            self.assertFalse(P.eligible(name))
        self.assertTrue(P.eligible('sdk/typescript/test/verify-coverage.mjs'))

    def test_source_identity_is_clean_actual_commit_tree_and_exact_package_helper_bytes(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve(); package = b'owned package'; helper = b'owned helper'
            def read(path, limit): return package if str(path).endswith('package.json') else helper
            def git(_root, *args):
                if args == ('rev-parse', 'HEAD'): return COMMIT.encode() + b'\n'
                if args == ('rev-parse', 'HEAD^{tree}'): return TREE.encode() + b'\n'
                if args[0] == 'status': return b''
                value = package if args[1].endswith('package.json') else helper
                return hashlib.sha1(b'blob ' + str(len(value)).encode() + b'\0' + value).hexdigest().encode()
            with mock.patch.object(P, 'git', side_effect=git), mock.patch.object(P, 'inventory', return_value=FILES), mock.patch.object(P, 'private_read', side_effect=read):
                value, files = P.source_identity(root, COMMIT)
                self.assertEqual(value['package_sha256'], P.digest(package)); self.assertEqual(files, FILES)
                with self.assertRaises(P.InvalidEvidence): P.source_identity(root, '0' * 40)
            with mock.patch.object(P, 'git', side_effect=[COMMIT.encode(), TREE.encode(), b' M web/owned.js']), self.assertRaises(P.InvalidEvidence):
                P.source_identity(root, COMMIT)

    def test_all_producer_platforms_and_fields_are_measured_not_invented_numeric_job_ids(self):
        for role in ('baseline', 'linux', 'macos', 'windows'):
            expected = producer(role)
            environment = {'GITHUB_REPOSITORY': P.REPOSITORY, 'GITHUB_RUN_ID': expected['run'],
                           'GITHUB_RUN_ATTEMPT': '1', 'GITHUB_JOB': expected['job'],
                           'GITHUB_EVENT_NAME': 'pull_request', 'CQ_PR_HEAD_SHA': HEAD}
            self.assertEqual(P.producer_identity(environment, expected['platform'], expected['suite']), expected)
            for field in ('GITHUB_REPOSITORY', 'GITHUB_RUN_ID', 'GITHUB_RUN_ATTEMPT', 'GITHUB_JOB', 'GITHUB_EVENT_NAME', 'CQ_PR_HEAD_SHA'):
                changed = {**environment, field: 'invalid field'}
                with self.assertRaises(P.InvalidEvidence): P.producer_identity(changed, expected['platform'], expected['suite'])
        with self.assertRaises(P.InvalidEvidence): P.producer_identity(environment, 'unknown', 'web')

    def test_retained_metadata_refuses_suite_failure_false_source_change_and_tool_substitution(self):
        _, entry, metadata = archive('linux', 1)
        _, pins = fixture()
        edits = [lambda x: x.update(schema='other'), lambda x: x.update(suite_exit=7),
                 lambda x: x.update(suite_exit=False), lambda x: x.update(original_check_passed=False),
                 lambda x: x.update(global_release_qualified=True),
                 lambda x: x.update(source_after={**x['source_after'], 'commit': '0' * 40}),
                 lambda x: x.update(node_after={**x['node_after'], 'sha256': '0' * 64}),
                 lambda x: x.update(producer={**x['producer'], 'head': '0' * 40}),
                 lambda x: x.update(files={}), lambda x: x.update(node_after={**x['node_after'], 'extra': 'private'})]
        for edit in edits:
            changed = copy.deepcopy(metadata); edit(changed)
            with self.assertRaises(P.InvalidEvidence): P.validate_metadata(changed, entry, pins, 'linux', FILES)

    def test_archive_internal_descriptors_and_original_line_counters_must_match_external_member_pins(self):
        values, pins = fixture()
        data, entry, metadata = values['linux']
        original = P.read_archive(data)
        for filename in ('metadata.json', 'lcov.info', 'counters.json'):
            members = {**original, filename: original[filename] + b' '}
            changed = zip_members(members); new_entry = {**entry, 'archive_sha256': P.digest(changed)}
            with self.assertRaises(P.InvalidEvidence): P.accept_archive(changed, new_entry, pins, 'linux', FILES)
        # Even independently pinned member hashes cannot claim different DA hits
        # in the original and sanitized counter documents.
        altered = counter(1)
        members = {**original, 'counters.json': P.encoded(altered)}
        metadata = copy.deepcopy(metadata); metadata['counters'] = {'sha256': P.digest(members['counters.json']), 'bytes': len(members['counters.json'])}
        members['metadata.json'] = P.encoded(metadata); changed = zip_members(members)
        new_entry = {**entry, 'archive_sha256': P.digest(changed), 'counters_sha256': P.digest(members['counters.json']), 'metadata_sha256': P.digest(members['metadata.json'])}
        with self.assertRaises(P.InvalidEvidence): P.accept_archive(changed, new_entry, pins, 'linux', FILES)

    def test_raw_leaf_zero_cannot_be_changed_or_deleted_in_either_derived_counter_view(self):
        values, pins = fixture()
        _data, entry, metadata = values['linux']
        original = P.read_archive(values['linux'][0])
        # Re-pin all modified document bytes, so a hash mismatch cannot be the
        # cause of refusal. The unchanged authenticated leaf/reporter still
        # proves the original precise zero; the derived view may not contradict it.
        edits = [lambda x: x['files']['web/owned.js']['functions'][1]['ranges'][0].__setitem__(2, 1),
                 lambda x: x['files']['web/owned.js']['functions'][1]['ranges'].pop(0),
                 lambda x: x['scripts'][0]['functions'][1]['ranges'][1].__setitem__(2, 1),
                 lambda x: x['observations'][0][0]['functions'][1]['ranges'][1].__setitem__(2, 1),
                 lambda x: x['observations'][0][0]['functions'][1]['ranges'].pop(1)]
        for edit in edits:
            counts = counter(0); edit(counts)
            members = {**original, 'counters.json': P.encoded(counts)}
            current_metadata = copy.deepcopy(metadata)
            current_metadata['counters'] = {'sha256': P.digest(members['counters.json']), 'bytes': len(members['counters.json'])}
            members['metadata.json'] = P.encoded(current_metadata)
            data = zip_members(members)
            changed_entry = {**entry, 'archive_sha256': P.digest(data),
                             'counters_sha256': P.digest(members['counters.json']),
                             'metadata_sha256': P.digest(members['metadata.json'])}
            with self.subTest(edit=edit), self.assertRaises(P.InvalidEvidence):
                P.accept_archive(data, changed_entry, pins, 'linux', FILES)

    def test_same_sanitized_native_leaf_drives_both_granularity_views_and_exact_reporter_replay(self):
        observations = [[report_script(0)]]
        actual = P.aggregate_observations(observations, FILES)
        self.assertEqual(actual, raw_fixture(0))
        document = P.canonical_counters(actual, P.parse_lcov(lcov(0), FILES), FILES)
        self.assertEqual(document, counter(0))
        self.assertEqual(observations, [[report_script(0)]])  # replay owns copies
        # Native sum does not erase the retained original leaf zero.
        next_report = report_script(1)
        mixed = P.aggregate_observations(observations + [[next_report]], FILES)
        self.assertEqual(mixed['observations'][0][0]['functions'][1]['ranges'][1], [5, 10, 0])
        self.assertEqual(mixed['raw']['web/owned.js'][key()][(5, 10)], 1)
        self.assertEqual(mixed['scripts'][0]['functions'][1]['ranges'][0], [5, 15, 2])

    def test_duplicate_role_artifact_job_or_split_original_attempts_cannot_fill_platform_slots(self):
        values, pins = fixture()
        for field in ('artifact_id', 'run_id', 'job_id'):
            entry = copy.deepcopy(pins['entries']['linux']); entry[field] = '0'
            with self.assertRaises(P.InvalidEvidence): P.checked_entry(entry)
        for field in ('job_id', 'artifact_id'):
            changed = copy.deepcopy(pins)
            changed['entries']['windows'][field] = changed['entries']['linux'][field]
            data = P.encoded(changed)
            with mock.patch.object(P, 'source_identity', return_value=(SOURCE, FILES)), \
                 mock.patch.object(P, 'node_identity', return_value=node('baseline')), \
                 mock.patch.object(P, 'producer_identity', return_value=producer('baseline')), self.assertRaises(P.InvalidEvidence):
                P.union(Path('/owned'), COMMIT, data, P.digest(data),
                        {role: row[0] for role, row in values.items()}, '/owned/node', {})

    def test_local_baseline_has_no_provider_id_and_requires_actual_current_source_tool_and_run(self):
        values, pins = fixture()
        members, entry, metadata = values['baseline']
        self.assertNotIn('artifact_id', entry); self.assertNotIn('archive_sha256', entry)
        with mock.patch.object(P, 'producer_identity', return_value=producer('baseline')), \
             mock.patch.object(P, 'node_identity', return_value=node('baseline')):
            profile, actual = P.accept_local_baseline(Path('/owned'), members, entry, pins, FILES,
                                                       SOURCE, '/owned/node', {})
            self.assertEqual(actual, metadata); self.assertEqual(profile['web/owned.js']['lines'][2], 0)
            for altered in ({**entry, 'artifact_id': '123'}, {**entry, 'archive_sha256': 'a' * 64},
                            {**entry, 'kind': 'externally-authenticated-artifact'},
                            {**entry, 'source_commit': '0' * 40}):
                with self.assertRaises(P.InvalidEvidence):
                    P.accept_local_baseline(Path('/owned'), members, altered, pins, FILES, SOURCE, '/owned/node', {})
        for field, value in [('producer_identity', {**producer('baseline'), 'run': '999'}),
                             ('node_identity', {**node('baseline'), 'sha256': '0' * 64})]:
            with mock.patch.object(P, 'producer_identity', return_value=producer('baseline')), \
                 mock.patch.object(P, 'node_identity', return_value=node('baseline')), \
                 mock.patch.object(P, field, return_value=value), self.assertRaises(P.InvalidEvidence):
                P.accept_local_baseline(Path('/owned'), members, entry, pins, FILES, SOURCE, '/owned/node', {})
        with self.assertRaises(P.InvalidEvidence): P.checked_entry(entry)
        with self.assertRaises(P.InvalidEvidence): P.checked_local_entry(values['linux'][1])

    def test_local_baseline_reads_only_fixed_three_owned_members_not_raw_or_invented_archive(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve(); directory = root / 'cq-browser-full-browser-linux'; directory.mkdir()
            for path, value in [('lcov.info', b'LCOV'), ('counters.json', b'COUNTERS'), ('metadata.json', b'META')]:
                (directory / path).write_bytes(value)
            (directory / 'raw').mkdir(); (directory / 'begin.json').write_bytes(b'NOT-A-MEMBER')
            members = P.read_local_members(directory, {'RUNNER_TEMP': str(root)})
            self.assertEqual(members, {'lcov.info': b'LCOV', 'counters.json': b'COUNTERS', 'metadata.json': b'META'})
            with self.assertRaises(P.InvalidEvidence): P.read_local_members(root, {'RUNNER_TEMP': str(root)})

    def test_unmeasured_source_is_reported_as_missing_not_fabricated_hits(self):
        files = {**FILES, 'web/not-loaded.js': FILES['web/owned.js']}
        values, pins = fixture()
        normalized = P.validated_counters(counter(1), FILES)
        source = {**SOURCE, 'inventory_sha256': P.digest(P.encoded(P.public_inventory(files)))}
        with mock.patch.object(P, 'source_identity', return_value=(source, files)), \
             mock.patch.object(P, 'accept_archive', return_value=(normalized, values['linux'][2])), \
             mock.patch.object(P, 'accept_local_baseline', return_value=(normalized, values['baseline'][2])):
            data = P.encoded(pins)
            _, result, _, code = P.union(Path('/owned'), COMMIT, data, P.digest(data), {role: row[0] for role, row in values.items()})
        self.assertEqual(code, 1); self.assertEqual(result['unmeasured_first_party_files'], ['web/not-loaded.js'])
        self.assertFalse(result['coverage_passed'])

    def test_main_begin_retain_and_union_use_fixed_output_and_return_distinct_zero_one_two(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve()
            common = ['--source', str(root), '--expected-commit', COMMIT]
            with mock.patch.object(P, 'begin') as begin:
                self.assertEqual(P.main(['begin', *common, '--output', str(root / 'begin')]), 0)
                begin.assert_called_once()
            with mock.patch.object(P, 'retain') as retain:
                self.assertEqual(P.main(['retain', *common, '--output', str(root / 'retain')]), 0)
                retain.assert_called_once()
            for code in (0, 1):
                output = root / 'cq-browser-composite'
                argv = ['union', *common, '--output', str(output), '--pins', str(root / 'pins'), '--expected-pins-sha256', '0' * 64]
                for role in ('baseline', 'linux', 'macos', 'windows'): argv.extend(['--' + role, str(root / role)])
                with mock.patch.dict(os.environ, {'RUNNER_TEMP': str(root)}), mock.patch.object(P, 'private_read', return_value=b'owned'), \
                     mock.patch.object(P, 'read_local_members', return_value={'lcov.info': b'owned', 'counters.json': b'owned', 'metadata.json': b'owned'}), \
                     mock.patch.object(P, 'union', return_value=(b'LCOV', {'passed': code == 0}, [], code)):
                    self.assertEqual(P.main(argv), code)
                self.assertEqual((output / 'lcov.info').read_bytes(), b'LCOV')
                for path in output.iterdir(): path.unlink()
                output.rmdir()


    def test_actual_script_entry_refuses_missing_owned_begin_without_any_tool_or_network(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve()
            args = [str(SCRIPT), 'retain', '--source', str(root), '--expected-commit', COMMIT,
                    '--output', str(root / 'cq-browser-web-linux'), '--platform', 'linux',
                    '--suite', 'web', '--node', str(root / 'node'), '--test-exit', '0']
            with mock.patch.object(sys, 'argv', args), mock.patch.dict(os.environ, {'RUNNER_TEMP': str(root)}), \
                 mock.patch('subprocess.Popen', side_effect=AssertionError('no tool may run')), \
                 contextlib.redirect_stderr(io.StringIO()) as error, self.assertRaises(SystemExit) as stopped:
                runpy.run_path(str(SCRIPT), run_name='__main__')
            self.assertEqual(stopped.exception.code, 2)
            self.assertEqual(error.getvalue(), P.REFUSAL + '\n')


if __name__ == '__main__':
    unittest.main()
