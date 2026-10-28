"""Historical one-time V1 -> V2 authoring script, not a current resource rebuild.

Do not rerun against current resources: subsequent economy/content repairs are
specified in docs/design/audit_repairs_2026_10_28.md and maintained in JSON.
"""
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RES = ROOT / 'MudGame-server/resources'


def read(path):
    return json.loads(path.read_text(encoding='utf-8'))


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n', encoding='utf-8', newline='\n')


def rows(text, start, end):
    return [[v.strip() for v in line.strip('|').split('|')]
            for line in text.split(start, 1)[1].split(end, 1)[0].splitlines() if line.startswith('| ')]


def main():
    proposal = (ROOT / 'docs/design/martial_arts_v2_proposal.md').read_text(encoding='utf-8')
    snapshot_path = RES / 'migrations/martial_v1.json'
    if snapshot_path.exists():
        raise SystemExit('V1 snapshot already exists; conversion is deliberately one-shot.')
    old_skills, old_items = read(RES / 'skills.json'), read(RES / 'items.json')
    mappings = {}
    names = {}
    for sid, name, action, target in rows(proposal, '### 8.1 全量技能映射', '### 8.2'):
        if not sid.startswith('`'):
            continue
        sid = sid.strip('`')
        dest = re.search(r'`([^`]+)`', target)[1] if action == '合并' else None if action == '退役' else sid
        mappings[sid] = dest
        if dest:
            names[dest] = target.split(' / ')[0]
    assert len(mappings) == 83
    kept = {sid for sid, dest in mappings.items() if sid == dest}
    legacy = {'version': 1, 'skills': {sid: {'name': v['name'], 'type': v['type'], 'target': mappings[sid]} for sid, v in old_skills.items()},
              'books': {iid: {'skill': v['effects']['learnSkill'], 'value': v['value'], 'name': v['name']}
                        for iid, v in old_items.items() if v.get('effects', {}).get('learnSkill')},
              'equipment': {iid: {'slot': v.get('equipmentSlot'), 'name': v['name']} for iid, v in old_items.items() if v.get('equipmentSlot')}}
    assert len(legacy['books']) == 69
    write(snapshot_path, legacy)
    npcs = read(RES / 'npcs.json')
    npc_by_name = {v['name']: k for k, v in npcs.items()}
    skills = {sid: dict(old_skills[sid], name=names[sid]) for sid in old_skills if sid in kept}
    basic_by_name = {v['name']: sid for sid, v in skills.items() if sid.startswith('basic_')}
    dungeons = {p: read(p) for p in (RES / 'dungeons').glob('*.json')}
    by_name = {v['name']: v for v in dungeons.values()}
    aliases = {'惊雁遗刻': '图录·后卷', '战神遗境': '战神图录·终卷'}
    teacher_names = {'混元功': '柳先生', '伏虎劲': '赵横江', '铁壁功': '雷万钧', '踏浪诀': '赵横江',
                     '悟真篇': '柳先生', '血战刀法': '秦校尉', '奔雷剑法': '秦校尉'}
    slot_map = {'内功': 'internal', '身法': 'movement', '护体': 'guard', '学识': 'study'}
    weapons = {'拳脚': 'unarmed', '刀': 'blade', '剑': 'sword', '棍': 'staff', '暗器': 'hidden_weapon', '枪': 'spear', '鞭': 'whip', '锏': 'mace', '钩': 'hook'}
    for name, tier, position, prereq, source, flavor in rows(proposal, '### 5.2 二十五门命名武学', '### 5.3'):
        if not re.match('[2-5]·', tier):
            continue
        sid = next(sid for sid, n in names.items() if n == name)
        skill = skills[sid]
        base_name, level = re.search(r'([^＋\d]+)(\d+)', prereq).groups()
        skill.update(type='ADVANCED', rarity=int(tier[0]), slot=slot_map.get(position, 'combat'), category=slot_map.get(position, 'combat'),
                     weaponType=weapons[position.split('·')[1]] if '·' in position else '',
                     teacherId=npc_by_name[teacher_names[name]] if name in teacher_names else '',
                     sourceDungeon='', manualId='manual_' + sid, source=source, description=flavor + '。' + source + '，获得完整传承后自主修炼。',
                     requirements={'prerequisite': {'skill': basic_by_name[base_name], 'level': int(level)}})
        if skill['rarity'] == 5:
            skill['requirements']['literacy'] = 80
        if name not in teacher_names:
            dun_name = re.search(r'《([^》]+)》', source)[1]
            skill['sourceDungeon'] = by_name[aliases.get(dun_name, dun_name)]['id']
    bonus_fields = {'攻击': 'attack', '防御': 'defense', '命中': 'hit_rate', '躲闪': 'dodge', '招架': 'parry',
                    '气血上限': 'max_hp', '内力上限': 'max_mp', '先手': 'initiative'}
    for name, description in rows(proposal, '| 武功 | 100级额外增益 |', '### 7.2'):
        sid = next(sid for sid, v in skills.items() if v['name'] == name)
        bonuses = {}
        for chinese, key in bonus_fields.items():
            match = re.search(chinese + r'\+(\d+)', description)
            if match:
                bonuses[key + '_per_100_levels'] = int(match[1])
        if name == '长生诀': bonuses['trained_mp_to_max_hp_percent'] = .15
        if name == '混元功': bonuses['meditate_recovery_per_100_levels'] = 10
        if name == '悟真篇': bonuses['learning_invest_bonus_per_100_levels'] = 2
        match = re.search(r'暴击率\+(\d+)个百分点', description)
        if match: bonuses['crit_rate_per_100_levels'] = int(match[1])
        skills[sid]['bonuses'] = bonuses
    for sid, skill in skills.items():
        if not sid.startswith('basic_'): continue
        category = {'basic_neigong': 'internal', 'basic_qinggong': 'movement', 'basic_parry': 'guard', 'basic_literacy': 'study'}.get(sid, 'combat')
        weapon = {'basic_unarmed': 'unarmed', 'basic_hidden_weapon': 'hidden_weapon'}.get(sid, sid.removeprefix('basic_')) if category == 'combat' else ''
        bonuses = {'basic_neigong': {'max_mp_per_100_levels': 1000, 'trained_mp_to_max_hp_percent': .3},
                   'basic_qinggong': {'post_agility_per_100_levels': 10}, 'basic_parry': {'parry_per_100_levels': 10},
                   'basic_literacy': {'post_perception_per_100_levels': 10}, 'basic_unarmed': {'attack_per_100_levels': 30}}.get(sid, {'attack_per_100_levels': 20})
        skill.update(type='BASIC', rarity=1, slot=category, category=category, weaponType=weapon,
                     requirements={}, bonuses=bonuses, source='世界老师入门；已学后在安全公共房间自主修炼。', description='各流派根基；只在对应流派下生效。' if category == 'combat' else '通用根基，可自主修炼。')
    items = {k: v for k, v in old_items.items() if k not in legacy['books']}
    for sid, skill in skills.items():
        if skill['type'] != 'ADVANCED': continue
        items[skill['manualId']] = {'name': skill['name'] + '秘本', 'description': skill['source'] + '的完整传承，一门武功一本秘本；参悟须满足根基与传承条件。',
                                    'type': 'consumable', 'subtype': 'manual', 'stackable': False, 'sellable': False, 'usable': True,
                                    'bind': 'character', 'value': 0, 'rarity': skill['rarity'], 'unit': '册', 'limit': 1,
                                    'effects': {'learnSkill': sid}, 'source': '旧版迁移保留；新入门由指定老师传授。' if skill['teacherId'] else skill['source']}
    new_weapon_rows = rows(proposal, '### 5.4', '## 6.')
    for iid, name_kind, numbers, source in new_weapon_rows:
        if not iid.startswith('`'): continue
        iid = iid.strip('`'); name, kind = name_kind.split(' / ')
        effects = {key: int(m[1]) for chinese, key in [('攻击', 'attack'), ('命中', 'hitRate'), ('躲闪', 'dodge'), ('招架', 'parry')]
                   if (m := re.search(chinese + r'\+(\d+)', numbers))}
        item = {'name': name, 'description': name + '，适合' + kind + '流派修炼。', 'type': 'equipment', 'subtype': kind,
                'weaponType': kind, 'equipmentSlot': 'WEAPON', 'effects': effects, 'value': int(re.search(r'value(\d+)', numbers)[1]),
                'rarity': int(re.search(r'品阶(\d+)', numbers)[1]), 'stackable': False, 'sellable': True, 'limit': -1, 'unit': '件', 'source': source}
        if '首次满通关' in source:
            item['firstClaim'] = True
            dun_name = re.search(r'《([^》]+)》', source)[1]
            by_name[dun_name].setdefault('rewards', {}).setdefault('items', {})[iid] = 1
        items[iid] = item
        npcs['heluo_blacksmith']['shopItems'].append(iid)
    for iid, item in items.items():
        if item.get('equipmentSlot') in ['WEAPON', 'HIDDEN_WEAPON']:
            item['equipmentSlot'] = 'WEAPON'
            if item['subtype'] not in ['relic', 'gathering']: item['weaponType'] = item['subtype']
    for iid in ['gongbian_jian', 'jianshou_jian', 'xinggong_jian', 'jianchi_m']:
        items[iid].update(subtype='sword', weaponType='sword')
    for iid in ['yunhe_g']: items[iid].update(subtype='blade', weaponType='blade')
    items['langyan_ling'].update(equipmentSlot='RING', effects={'maxHp': 20, 'defense': 3, 'hitRate': 2})
    for iid in ['xb2_relic', 'tg2_relic', 'th2_relic', 'wr2_relic']: items[iid]['equipmentSlot'] = 'NECKLACE'
    items['iron_pickaxe'].pop('equipmentSlot', None)
    items['darts']['effects']['attack'] = 6
    items['langya_chui'].update(name='狼牙重锏', description='一柄沉重铁锏，适合锏法流派。')
    early = {'datang_changshengjue': ({'herb': 4, 'dry_ration': 2}, {'simple_medicine': 2, 'dry_ration': 1}),
             'posui_jingyan': (None, {'simple_medicine': 2, 'dry_ration': 3}), 'datang_tiandao': (None, {'xiaohuandan': 1}),
             'liaoshui_lair': ({'simple_medicine': 3, 'dry_ration': 2}, None), 'datang_feima': ({'xiaohuandan': 1, 'herb': 10}, None)}
    top_pages = {'csj_changshengjue_page', 'jyg_zhanshentu_rubbing', 'tiandao_page', 'cihang_page', 'tianmo_page'}
    for dungeon in dungeons.values():
        if dungeon['id'] not in early: continue
        action_replacement, reward_replacement = early[dungeon['id']]
        for action in dungeon.get('puzzleActions', {}).values():
            give = action.get('giveItems', {})
            if action_replacement and top_pages.intersection(give):
                for old in top_pages: give.pop(old, None)
                give.update(action_replacement)
        give = dungeon.get('rewards', {}).get('items', {})
        if reward_replacement and top_pages.intersection(give):
            for old in top_pages: give.pop(old, None)
            give.update(reward_replacement)
    replacements = {v['name']: (names[mappings[sid]] if mappings[sid] else '江湖武学心得') for sid, v in old_skills.items() if v['name'] != names.get(mappings[sid])}
    replacements.update({v['name']: '修行心得' for v in legacy['books'].values()})
    replacements.update({'图录·后卷': '惊雁遗刻', '战神图录·终卷': '战神遗境', '狼牙巨锤': '狼牙重锏'})
    def rewrite(value):
        if isinstance(value, dict):
            for key in ['giveItems', 'items']:
                give = value.get(key)
                if not isinstance(give, dict): continue
                refund = sum(give.pop(iid, 0) * (v['value'] // 2) for iid, v in legacy['books'].items())
                if refund: value['giveMoney' if key == 'giveItems' else 'money'] = value.get('giveMoney' if key == 'giveItems' else 'money', 0) + refund
            if 'skills' in value and isinstance(value['skills'], dict):
                value['skills'] = {sid: level for sid, level in value['skills'].items() if sid.startswith('basic_') and sid in skills}
            if 'shopItems' in value: value['shopItems'] = [iid for iid in value['shopItems'] if iid not in legacy['books']]
            return {key: rewrite(item) for key, item in value.items()}
        if isinstance(value, list): return [rewrite(item) for item in value]
        if isinstance(value, str):
            for old, new in sorted(replacements.items(), key=lambda x: -len(x[0])): value = value.replace(old, new)
        return value
    for path in [RES / 'dialogues.json', RES / 'dungeon_dialogues.json', RES / 'quests.json', *(RES / 'worlds').glob('*.json')]:
        write(path, rewrite(read(path)))
    npcs = rewrite(npcs)
    npcs['db_zhang_xiu']['weaknessModifiers'].update(attack=-24, hitRate=-26)
    for sid, skill in skills.items():
        teacher = skill.get('teacherId')
        if teacher: npcs[teacher].setdefault('skills', {})[sid] = 1
    # The old keeper becomes a lore character, with no globally omniscient teaching list.
    npcs['ancient_scroll_keeper']['skills'] = {}
    npcs['ancient_scroll_keeper']['description'] = '翻阅旧闻的灰袍老者，愿意谈江湖掌故；真正的武学传承仍须亲历。'
    for path, dungeon in dungeons.items():
        dungeon = rewrite(dungeon)
        for obj in dungeon.get('objectives', []):
            if obj.get('targetId') in legacy['books']:
                obj['targetId'] = 'herb'
                obj['description'] = '打捞沉船中的止血药草'
        for sid, skill in skills.items():
            if skill.get('sourceDungeon') == dungeon['id']:
                dungeon.setdefault('rewards', {}).setdefault('items', {})[skill['manualId']] = 1
        write(path, dungeon)
    write(RES / 'skills.json', skills); write(RES / 'items.json', items); write(RES / 'npcs.json', npcs)
    print('Converted: 38 skills, 25 complete manuals (18 drop sources), 9 new weapons; legacy snapshot retained.')


if __name__ == '__main__': main()
