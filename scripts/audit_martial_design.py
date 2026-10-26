"""Read-only arithmetic audit of the V2 proposal and current game resources.

This is a design worksheet, not a combat engine or a playthrough simulation.
Run from any directory: python scripts/audit_martial_design.py
"""

import json
import math
import re
from collections import Counter, defaultdict
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RESOURCES = ROOT / 'MudGame-server/resources'
PROPOSAL = ROOT / 'docs/design/martial_arts_v2_proposal.md'


def read_json(path):
    return json.loads(path.read_text(encoding='utf-8'))


def rows_between(text, start, end):
    section = text.split(start, 1)[1].split(end, 1)[0]
    return [[cell.strip() for cell in line.strip('|').split('|')]
            for line in section.splitlines() if line.startswith('| ')]


def cost_for_level(level, bands):
    band = (level - 1) // 20
    return bands[band] if band < len(bands) else math.ceil(level ** 3 / 128000)


def expected_damage(attacker, defender, parry_floor):
    """Exact mean for current integer damage rolls; probability floor is a scenario input."""
    hit_floor = .15 if parry_floor == 0 else .45
    parry_ceiling = .60 if parry_floor == 0 else .95
    hit = min(.95, max(hit_floor, attacker['hit'] / (attacker['hit'] + defender['dodge'])))
    parry = min(parry_ceiling, max(parry_floor, defender['parry'] / (defender['parry'] + attacker['hit'])))
    base = max(2, attacker['attack'] - defender['defense'] // 3 + 14)
    total = 0.0
    for fluctuation in range(90, 111):
        normal = base * fluctuation // 100
        critical = normal * 2
        crit = min(.25, attacker.get('crit', 1) / 100)
        total += ((1 - crit) * ((1 - parry) * normal + parry * (normal // 2))
                  + crit * ((1 - parry) * critical + parry * (critical // 2))) / 21
    return total * hit


def named_stats(text):
    rows = rows_between(text, '| 武功 | 100级额外增益 |', '### 7.2')
    fields = {'攻击': 'attack', '命中': 'hit', '躲闪': 'dodge',
              '招架': 'parry', '防御': 'defense', '气血上限': 'hp'}
    result = {}
    for name, description in rows:
        values = {}
        for chinese, english in fields.items():
            match = re.search(chinese + r'\+(\d+)', description)
            if match:
                values[english] = int(match[1])
        match = re.search(r'暴击率\+([\d.]+)个百分点', description)
        if match:
            values['crit'] = float(match[1])
        result[name] = values
    assert len(result) == 25, len(result)
    return result


def main():
    live_skills = read_json(RESOURCES / 'skills.json')
    legacy = read_json(RESOURCES / 'migrations/martial_v1.json')
    skills = legacy['skills']
    items = read_json(RESOURCES / 'items.json')
    npcs = read_json(RESOURCES / 'npcs.json')
    text = PROPOSAL.read_text(encoding='utf-8')
    source = (ROOT / 'MudGame-server/internal/model/skill_progression.go').read_text(encoding='utf-8')
    bands = [int(x) for x in re.findall(r'\d+', source.split('skillPotentialCostsByTwentyLevels = []int{')[1].split('}')[0])]
    assert bands == [1, 2, 4, 7, 11, 16, 23, 32, 44, 60, 80, 105, 135, 170, 210]
    mappings = [row for row in rows_between(text, '### 8.1 全量技能映射', '### 8.2') if row[0].startswith('`')]
    mapping_ids = [row[0].strip('`') for row in mappings]
    assert len(mapping_ids) == len(set(mapping_ids)) == len(skills) == 83
    assert set(mapping_ids) == set(skills)
    mapping_counts = Counter('merged' if row[2] == '合并' else 'retired' if row[2] == '退役' else 'retained' for row in mappings)
    assert mapping_counts == {'retained': 38, 'merged': 15, 'retired': 30}
    retained = {row[0].strip('`') for row in mappings if row[2] != '合并' and row[2] != '退役'}
    assert retained == set(live_skills), 'catalog differs from implemented resources'
    for row in mappings:
        if row[2] == '合并':
            assert re.search(r'`([^`]+)`', row[3])[1] in retained, row
    catalog = rows_between(text, '### 5.2 二十五门命名武学', '### 5.3')
    arts = [row for row in catalog if re.match('[2-5]·', row[1])]
    assert len({row[0] for row in arts}) == len(arts) == 25
    assert Counter(row[1][0] for row in arts) == {'2': 7, '3': 9, '4': 4, '5': 5}
    new_weapons = [row[0].strip('`') for row in rows_between(text, '### 5.4', '## 6.') if row[0].startswith('`')]
    assert len(new_weapons) == len(set(new_weapons)) == 9
    assert all(re.fullmatch('[a-z][a-z0-9_]*', key) and key in items for key in new_weapons)
    book_rows = [row for row in rows_between(text, '### 8.2 旧秘籍物品的完整处理表', '## 9.') if row[0].startswith('`')]
    book_ids = [row[0].strip('`') for row in book_rows]
    expected_books = set(legacy['books'])
    assert len(book_ids) == len(set(book_ids)) == len(expected_books) == 69
    assert set(book_ids) == expected_books
    assert not re.search(r'(?m)^\|[^\n]*\|\n\n\|', text), 'Broken Markdown table'

    grouped = defaultdict(list)
    for path in (RESOURCES / 'dungeons').glob('*.json'):
        dungeon = read_json(path)
        grouped[dungeon['order']].append(dungeon)
    progress = []
    exp = potential = 0
    for order, dungeons in sorted(grouped.items()):
        exp += min(d.get('rewards', {}).get('experience', 0) for d in dungeons)
        potential += min(d.get('rewards', {}).get('potential', 0) for d in dungeons)
        if order in [9, 17, 26, 36, 46, 56, 71]:
            progress.append({'order': order, 'min_experience': exp,
                             'min_potential_with_initial_500': potential + 500})
    training = []
    for level in [20, 40, 80, 100, 120, 140, 170]:
        costs = [cost_for_level(n, bands) for n in range(2, level + 1)]
        training.append({'level': level, 'potential': sum(costs),
                         'one_level_per_tick_seconds': sum(math.ceil(c / 10) for c in costs) * 10,
                         'carry_budget_10_seconds': math.ceil(sum(costs) / 10) * 10,
                         'carry_budget_12_seconds': math.ceil(sum(costs) / 12) * 10})

    equipment = [item for item in items.values() if item.get('equipmentSlot') in ['WEAPON', 'HIDDEN_WEAPON']]
    subtype_counts = Counter(item.get('subtype') for item in equipment)
    missing = [subtype for subtype in ['blade', 'sword', 'staff', 'hidden_weapon', 'spear', 'whip', 'mace', 'hook'] if not subtype_counts[subtype]]
    # Normalized comparison: base five stats 16; all relevant roots 120;
    # combat arts 100; hypothetical equal weapon attack +12 (empty hand +0).
    # No armor, inner art, movement art, medicine, weakness or enhancement.
    # These weapon assumptions deliberately separate art balance from missing items.
    reference = npcs['gb2_boss']['attributes']
    enemy = {'attack': reference['derived']['attack'], 'defense': reference['derived']['defense'],
             'hit': reference['derived']['hitRate'], 'dodge': reference['derived']['dodge'],
             'parry': reference['derived']['parry'], 'crit': reference['derived']['critRate']}
    current = named_stats(text)
    old = {'破虚指': {'attack': 28, 'hit': 14}, '天刀八诀': {'attack': 34, 'hit': 22},
           '慈航剑典': {'attack': 20, 'hit': 26, 'parry': 10}, '霸王枪': {'attack': 24, 'hit': 6},
           '游龙鞭': {'attack': 10, 'dodge': 16}, '裂石锏': {'attack': 16, 'parry': 16},
           '锁浪钩': {'attack': 10, 'hit': 20}, '疯魔棍': {'attack': 18, 'parry': 12},
           '漫天花雨': {'attack': 18, 'hit': 20, 'crit': 1}}
    basic_old = {'破虚指': {'attack': 24, 'defense': 6, 'parry': 12}, '天刀八诀': {'attack': 12},
                 '慈航剑典': {'hit': 12}, '霸王枪': {'attack': 12}, '游龙鞭': {'dodge': 12},
                 '裂石锏': {'parry': 12}, '锁浪钩': {'hit': 12},
                 '疯魔棍': {'attack': 12, 'parry': 12}, '漫天花雨': {'hit': 12, 'crit': 1.2}}
    combat = []
    for name, original in old.items():
        entry = {'art': name}
        for revised in [False, True]:
            actor = {'attack': 56, 'defense': 53, 'hit': 134, 'dodge': 100, 'parry': 88, 'crit': 1}
            actor['attack'] += 0 if name == '破虚指' else 12
            art = next(skill for skill in live_skills.values() if skill['name'] == name)
            root_def = live_skills[art['requirements']['prerequisite']['skill']]
            root = {'attack': math.floor(root_def['bonuses']['attack_per_100_levels'] * 120 / 100)} if revised else basic_old[name]
            for bonuses in [root, current[name] if revised else original]:
                for field, value in bonuses.items():
                    actor[field] = actor.get(field, 0) + value
            outgoing = expected_damage(actor, enemy, 0 if revised else .45)
            incoming = expected_damage(enemy, actor, 0 if revised else .45)
            label = 'revised' if revised else 'draft'
            entry[label + '_mean_damage_per_round'] = round(outgoing, 2)
            entry[label + '_enemy_hp_over_mean_damage'] = round(reference['resources']['maxHp'] / outgoing, 2)
            entry[label + '_mean_incoming_per_round'] = round(incoming, 2)
        combat.append(entry)
    old_spread = max(x['draft_mean_damage_per_round'] for x in combat) / min(x['draft_mean_damage_per_round'] for x in combat)
    new_spread = max(x['revised_mean_damage_per_round'] for x in combat) / min(x['revised_mean_damage_per_round'] for x in combat)
    inners = []
    inner_rows = rows_between(text, '| 武功 | 100级额外增益 |', '### 7.2')
    inner_descriptions = {name: description for name, description in inner_rows}
    for name in ['混元功', '洞庭心法', '战神图录', '长生诀', '天魔策']:
        actor = {'attack': 56 + 12 + 24, 'defense': 53, 'hit': 134, 'dodge': 100, 'parry': 88, 'crit': 1}
        for bonuses in [current['天刀八诀'], current[name]]:
            for field, value in bonuses.items():
                actor[field] = actor.get(field, 0) + value
        extra_mp = int(re.search(r'内力上限\+(\d+)', inner_descriptions[name])[1])
        trained = 5 * (28 + 16) + 120 * 10 + extra_mp
        conversion = .45 if name == '长生诀' else .30
        max_hp = 60 + 16 * 4 + math.ceil(trained * conversion) + current[name].get('hp', 0)
        damage = expected_damage(actor, enemy, 0)
        incoming = expected_damage(enemy, actor, 0)
        inners.append({'art': name, 'fully_trained_mp': trained, 'max_hp': max_hp,
                       'mean_outgoing': round(damage, 2), 'mean_incoming': round(incoming, 2),
                       'hp_over_incoming': round(max_hp / incoming, 2),
                       'enemy_hp_over_outgoing': round(reference['resources']['maxHp'] / damage, 2)})
    silver = .6 * (items['bronze_fragment']['value'] // 2) + .4 * (items['dry_ration']['value'] // 2)
    ordinary_silver = .6 * (items['herb']['value'] // 2) + .4 * (items['dry_ration']['value'] // 2)
    # P=16+12 literacy, W=16; basic internal 120; Changsheng 100.
    mp_cap = 5 * (28 + 16) + 120 * 10 + 600
    meditation_ticks = math.ceil((mp_cap - 80) / 12)
    result = {'kind': 'read_only_design_arithmetic_not_live_playtest',
              'mapping': dict(mapping_counts), 'legacy_books': len(book_rows),
              'missing_orders': sorted(set(range(1, max(grouped) + 1)) - set(grouped)),
              'live_weapon_counts': dict(subtype_counts), 'missing_weapon_types': missing,
              'untyped_primary_items': [key for key, item in items.items()
                                       if item.get('equipmentSlot') == 'WEAPON'
                                       and item.get('subtype') in ['relic', 'gathering']],
              'progression': progress, 'training': training,
              'normalized_combat': combat,
              'normalized_inner_comparison': inners,
              'damage_max_over_min': {'draft': round(old_spread, 3), 'revised': round(new_spread, 3)},
              'meditation': {'hypothetical_max_mp': mp_cap, 'start_trained_mp': 80,
                             'old_growth_seconds': (mp_cap - 80) * 10,
                             'revised_growth_seconds': meditation_ticks * 10,
                             'excludes_current_mp_recovery_time': True},
              'gathering': {'baseline_copper_per_minute': silver * 2,
                            'proposed_copper_per_minute': round(ordinary_silver * 2, 2),
                            'labor_copper_per_minute': 20}}
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
