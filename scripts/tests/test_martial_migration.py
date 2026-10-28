import importlib.util
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[2]
RES = ROOT / 'MudGame-server/resources'
SPEC = importlib.util.spec_from_file_location('martial_migration', RES / 'migrations/migrate_martial_v2.py')
M = importlib.util.module_from_spec(SPEC)
exec(compile((RES / 'migrations/migrate_martial_v2.py').read_bytes(), str(RES / 'migrations/migrate_martial_v2.py'), 'exec'), M.__dict__)


def player(identifier='hero'):
    return {'id': identifier, 'characterName': '旧角色', 'userId': 'owner', 'money': 5000,
            'roomId': 'ly_00', 'state': 'LEARNING', 'title': 'preserve this metadata',
            'attributes': {'experience': 200000, 'potential': 321,
                           'base': {k: 16 for k in ['strength', 'constitution', 'agility', 'perception', 'willpower']},
                           'resources': {'hp': 100, 'maxHp': 160, 'mp': 40, 'trainedMaxMp': 500, 'maxMp': 1500}},
            'skills': {'basic_neigong': 120, 'basic_literacy': 80, 'basic_sword': 120},
            'skillLearningProgress': {}, 'items': {}, 'equipment': {'WEAPON': 'iron_sword'},
            'equippedSkills': {}, 'flags': {}, 'questStates': {'a': {'status': 'completed'}},
            'learningSkillId': 'basic_sword', 'learningTeacherId': 'teacher'}


class MartialMigrationTests(unittest.TestCase):

    def test_item_aliases_merge_and_refund_once(self):
        original = player()
        original['items'] = {'langfan_jian': 2, 'langfan_j': 1}
        original['enhancedItems'] = {'langfan_jian': 2, 'langfan_j': 3}
        aliases = json.loads((RES / 'item_aliases.json').read_text(encoding='utf-8'))
        result = M.normalize_item_aliases(original, aliases)
        self.assertEqual(result['items']['langfan_j'], 3)
        self.assertEqual(result['enhancedItems'], {'langfan_j': 3})
        self.assertEqual(result['money'], 5600)
        self.assertEqual(result['items']['bronze_fragment'], 3)
        before = json.dumps(result, sort_keys=True)
        self.assertEqual(json.dumps(M.normalize_item_aliases(result, aliases), sort_keys=True), before)
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.base = Path(self.temporary.name)
        self.data = self.base / 'data'
        (self.data / 'players').mkdir(parents=True)

    def save(self, p):
        path = self.data / 'players' / (p['id'] + '.json')
        path.write_text(json.dumps(p, ensure_ascii=False), encoding='utf-8')
        return path

    def test_preview_is_read_only_and_apply_is_idempotent(self):
        original = player(); path = self.save(original); before = path.read_bytes()
        preview = M.run(self.data, RES)
        self.assertEqual(preview['changed'], 1)
        self.assertEqual(path.read_bytes(), before)
        self.assertEqual(list(self.base.iterdir()), [self.data])
        result = M.run(self.data, RES, True, True, self.base / 'backup')
        changed = path.read_bytes()
        self.assertEqual((self.base / 'backup/data/players/hero.json').read_bytes(), before)
        self.assertFalse((self.base / 'backup/data/.martial-migration.lock').exists())
        new = json.loads(changed)
        self.assertEqual(new['martialVersion'], 2)
        self.assertEqual(new['title'], original['title'])
        self.assertEqual(new['questStates'], original['questStates'])
        self.assertEqual(new['state'], 'IDLE')
        self.assertEqual(result['changed'], 1)
        self.assertEqual(M.run(self.data, RES, True, True)['changed'], 0)
        self.assertEqual(path.read_bytes(), changed)

    def test_merge_conserves_investment_and_keeps_one_art(self):
        p = player()
        p['skills'].update(taiyi_jianfa=50, taiyi_xiajuan=60)
        p['skillLearningProgress'] = {'taiyi_jianfa': 2}
        p['flags']['dungeon_cleared_datang_jingling'] = True
        p['equippedSkills']['basic_sword'] = 'taiyi_xiajuan'
        self.save(p)
        M.run(self.data, RES, True, True)
        new = M.read(self.data / 'players/hero.json')
        before = p['attributes']['potential'] + M.spent(50, 2, True) + M.spent(60, 0, True)
        # Non-free basic_sword gets its paid first-level credit back in the new curve.
        before += M.spent(120, 0, False)
        after = new['attributes']['potential'] + M.spent(new['skills']['taiyi_jianfa'], new['skillLearningProgress']['taiyi_jianfa'], True)
        after += M.spent(new['skills']['basic_sword'], new['skillLearningProgress']['basic_sword'], True)
        # Literacy is another formerly paid BASIC first level, and remains conserved.
        before += M.spent(80, 0, False)
        after += M.spent(new['skills']['basic_literacy'], new['skillLearningProgress']['basic_literacy'], True)
        self.assertEqual(before, after)
        self.assertNotIn('taiyi_xiajuan', new['skills'])
        self.assertEqual(new['activeArts']['combat'], 'taiyi_jianfa')

    def test_early_absolute_art_refunds_and_manuals_do_not_bypass_gate(self):
        p = player(); p['skills']['csj_changshengjue'] = 80
        p['items'] = {'csj_changshengjue_page': 3}
        self.save(p); preview = M.run(self.data, RES)
        self.assertIn('csj_changshengjue', preview['players'][0]['refundedUnavailable'])
        M.run(self.data, RES, True, True)
        new = M.read(self.data / 'players/hero.json')
        self.assertNotIn('csj_changshengjue', new['skills'])
        self.assertEqual(new['items']['manual_csj_changshengjue'], 1)
        self.assertEqual(new['money'], 5500)
        self.assertFalse(new['flags'].get('dungeon_cleared_datang_xinggong'))

    def test_dual_weapon_and_tools_are_returned_without_loss(self):
        p = player(); p['equipment'].update(HIDDEN_WEAPON='darts', RING='tianmo_pei')
        self.save(p); M.run(self.data, RES, True, True)
        new = M.read(self.data / 'players/hero.json')
        self.assertEqual(new['equipment']['WEAPON'], 'iron_sword')
        self.assertEqual(new['items']['darts'], 1)
        self.assertNotIn('HIDDEN_WEAPON', new['equipment'])

    def test_tool_is_returned_before_primary_dark_weapon_is_selected(self):
        p = player()
        p['equipment'] = {'HIDDEN_WEAPON': 'darts', 'WEAPON': 'iron_pickaxe'}
        self.save(p); M.run(self.data, RES, True, True)
        updated = M.read(self.data / 'players/hero.json')
        self.assertEqual(updated['equipment']['WEAPON'], 'darts')
        self.assertEqual(updated['items']['iron_pickaxe'], 1)

    def test_missing_legacy_trained_field_uses_old_cap_before_new_cap(self):
        p = player()
        p['attributes']['resources'].pop('trainedMaxMp')
        p['attributes']['resources']['maxMp'] = 200
        self.save(p); M.run(self.data, RES, True, True)
        updated = M.read(self.data / 'players/hero.json')
        self.assertEqual(updated['attributes']['resources']['trainedMaxMp'], 100)
        self.assertEqual(updated['attributes']['resources']['mp'], 40)

    def test_bad_progress_prevents_all_writes(self):
        a = self.save(player('one')); p = player('two')
        p['skillLearningProgress'] = {'basic_neigong': 100000}
        b = self.save(p); before = [a.read_bytes(), b.read_bytes()]
        with self.assertRaises(ValueError): M.run(self.data, RES, True, True)
        self.assertEqual([a.read_bytes(), b.read_bytes()], before)
        self.assertEqual(list(self.base.iterdir()), [self.data])

    def test_failed_second_replace_restores_first_player(self):
        paths = [self.save(player('one')), self.save(player('two'))]
        before = [p.read_bytes() for p in paths]
        replace = M.os.replace; calls = 0
        def fail_once(source, destination):
            nonlocal calls
            calls += 1
            if calls == 2: raise OSError('injected write failure')
            return replace(source, destination)
        with mock.patch.object(M.os, 'replace', side_effect=fail_once):
            with self.assertRaises(OSError): M.run(self.data, RES, True, True)
        self.assertEqual([p.read_bytes() for p in paths], before)
        self.assertFalse((self.data / '.martial-migration.lock').exists())

    def test_apply_requires_explicit_stopped_service(self):
        self.save(player())
        with self.assertRaisesRegex(ValueError, 'server-stopped'): M.run(self.data, RES, True)


if __name__ == '__main__': unittest.main()
