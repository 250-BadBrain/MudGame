"""Offline, explicit and idempotent martial migration. Default: report only."""
import argparse
import copy
import hashlib
import json
import math
import os
import shutil
import sys
import tempfile
import uuid
from datetime import datetime, timezone
from pathlib import Path


VERSION = 2
BANDS = [1, 2, 4, 7, 11, 16, 23, 32, 44, 60, 80, 105, 135, 170, 210]
FREE_ROOTS = {'basic_unarmed', 'basic_parry', 'basic_neigong', 'basic_qinggong'}


def read(path):
    return json.loads(path.read_text(encoding='utf-8-sig'))


def cost(level):
    band = (level - 1) // 20
    return BANDS[band] if band < len(BANDS) else math.ceil(level ** 3 / 128000)


def integer(value, field):
    if isinstance(value, bool) or not isinstance(value, int) or value < 0 or value > 9223372036854775807:
        raise ValueError(f'{field}: expected a nonnegative integer')
    return value


def spent(level, progress, free_first):
    integer(level, 'skill level'); integer(progress, 'learning progress')
    if level > 10000 or progress >= cost(level + 1):
        raise ValueError('unexpected level/progress; manual review required')
    return sum(cost(n) for n in range(2 if free_first else 1, level + 1)) + progress


def allocate(credit, cap):
    level = 1
    while level < cap and credit >= cost(level + 1):
        credit -= cost(level + 1); level += 1
    progress = min(credit, cost(level + 1) - 1) if level < cap else 0
    return level, progress, credit - progress


def experience_cap(experience):
    root = 0
    while 40 * (root + 1) ** 3 <= experience:
        root += 1
    return 20 + 10 * root


def eligible(p, skill):
    req = skill.get('requirements', {})
    pre = req.get('prerequisite', {})
    if p['skills'].get(pre.get('skill'), 0) < pre.get('level', 0): return False
    if p['skills'].get('basic_literacy', 0) < req.get('literacy', 0): return False
    source = skill.get('sourceDungeon')
    if source and not p['flags'].get('dungeon_cleared_' + source): return False
    if skill.get('teacherId') and not p['flags'].get('martial_taught_' + skill['id']): return False
    return True


def weapon_type(p, items):
    iid = p.get('equipment', {}).get('WEAPON')
    return items[iid].get('weaponType', 'unknown') if iid else 'unarmed'


def clamp_resources(p, skills, items):
    """Match the V2 cap ordering; never reduce permanent trainedMaxMp."""
    a = p['attributes']; base = a.get('base', {}); growth = a.get('growth', {})
    stats = {key: base.get(key, 0) + growth.get('post' + key[0].upper() + key[1:], 0)
             for key in ['strength', 'constitution', 'agility', 'perception', 'willpower']}
    hp_bonus = mp_bonus = 0
    conversion = 0.0
    for sid, level in p['skills'].items():
        skill = skills[sid]
        if skill['type'] != 'BASIC' and p['activeArts'].get(skill['slot']) != sid: continue
        if skill['slot'] == 'combat' and skill.get('weaponType') != weapon_type(p, items): continue
        if not eligible(p, skill): continue
        for key, value in skill['bonuses'].items():
            amount = math.floor(value * level / 100)
            if key == 'max_hp_per_100_levels': hp_bonus += amount
            elif key == 'max_mp_per_100_levels': mp_bonus += amount
            elif key == 'post_perception_per_100_levels': stats['perception'] += amount
            elif key == 'post_agility_per_100_levels': stats['agility'] += amount
            elif key == 'trained_mp_to_max_hp_percent': conversion += value if skill['type'] == 'BASIC' else value * min(level, 100) / 100
    for iid in p.get('equipment', {}).values():
        effects = dict(items[iid].get('effects', {}))
        enhancement = p.get('enhancedItems', {}).get(iid, 0)
        if enhancement > 0:
            effects = {key: value + enhancement * 2 for key, value in effects.items()}
        hp_bonus += effects.get('maxHp', 0); mp_bonus += effects.get('maxMp', 0)
        for field in stats: stats[field] += effects.get(field, 0) + effects.get('post' + field[0].upper() + field[1:], 0)
    resources = a.setdefault('resources', {})
    max_mp = max(50, (stats['perception'] + stats['willpower']) * 5) + mp_bonus
    permanent = integer(resources.get('trainedMaxMp', 0), 'trainedMaxMp')
    if permanent == 0:
        old_max = resources.get('maxMp', 0)
        if old_max <= 0:
            old_max = max(50, (base.get('perception', 0) + base.get('willpower', 0)) * 5)
        permanent = min(old_max, max(30, old_max // 2))
    resources['trainedMaxMp'] = permanent
    trained = min(permanent, max_mp)
    max_hp = 60 + stats['constitution'] * 4 + math.ceil(trained * min(.45, conversion)) + hp_bonus
    resources['maxHp'], resources['maxMp'] = max_hp, max_mp
    resources['hp'] = min(integer(resources.get('hp', 80), 'hp'), max_hp)
    resources['mp'] = min(integer(resources.get('mp', 0), 'mp'), trained)
    if p.get('state') != 'DEAD': resources['hp'] = max(1, resources['hp'])


def normalize_item_aliases(p, aliases):
    inventory = p.setdefault('items', {})
    upgrades = p.setdefault('enhancedItems', {})
    for old, canonical in aliases.items():
        if old in inventory:
            inventory[canonical] = integer(inventory.get(canonical, 0) + inventory.pop(old), 'alias quantity')
        for slot, iid in p.get('equipment', {}).items():
            if iid == old: p['equipment'][slot] = canonical
        if old in upgrades:
            level, other = upgrades.pop(old), upgrades.get(canonical, 0)
            if not all(type(v) is int and 0 <= v <= 5 for v in [level, other]): raise ValueError('invalid alias enhancement')
            duplicate = min(level, other)
            p['money'] = integer(p.get('money', 0) + 100 * duplicate * (duplicate + 1), 'alias money')
            if duplicate: inventory['bronze_fragment'] = integer(inventory.get('bronze_fragment', 0) + duplicate * (duplicate + 1) // 2, 'alias bronze')
            upgrades[canonical] = max(level, other)
        flags = p.setdefault('flags', {})
        key = 'item_claimed_' + old
        if key in flags:
            old_claim = flags.pop(key)
            flags['item_claimed_' + canonical] = flags.get('item_claimed_' + canonical, False) or old_claim
    return p


def migrate(player, legacy, skills, items, aliases=None):
    p = copy.deepcopy(player)
    if p.get('martialVersion', 0) == VERSION:
        return p, {'id': p['id'], 'status': 'already_v2'}
    if p.get('martialVersion', 0) > VERSION: raise ValueError('newer martial version; refusing downgrade')
    attributes = p['attributes']
    potential = integer(attributes.get('potential', 0), 'potential')
    experience = integer(attributes.get('experience', 0), 'experience')
    cap = experience_cap(experience)
    if aliases is None: aliases = read(Path(__file__).resolve().parents[1] / 'item_aliases.json')
    normalize_item_aliases(p, aliases)
    old_skills = p.get('skills') or {}; old_progress = p.get('skillLearningProgress') or {}
    credits = {}; owned = set(); retired = 0
    old_total = potential
    for sid in set(old_skills) | set(old_progress):
        if sid not in legacy['skills']: raise ValueError(f'unknown legacy skill: {sid}')
        definition = legacy['skills'][sid]
        level = old_skills.get(sid, 0)
        credit = spent(level, old_progress.get(sid, 0), sid in FREE_ROOTS or definition['type'] == 'ADVANCED')
        old_total += credit
        target = definition['target']
        if target:
            credits[target] = credits.get(target, 0) + credit
            if level > 0: owned.add(target)
        else: retired += credit
    potential += retired
    p['skills'], p['skillLearningProgress'] = {}, {}
    p['flags'] = p.get('flags') or {}
    refunds = []; kept_credit = 0
    for sid in sorted(credits, key=lambda sid: (skills[sid]['type'] != 'BASIC', sid)):
        definition = skills[sid]
        if definition.get('teacherId') and sid in owned: p['flags']['martial_taught_' + sid] = True
        available = cap if definition['type'] == 'BASIC' else min(cap, p['skills'].get(definition['requirements']['prerequisite']['skill'], 0))
        credit = credits[sid]
        if sid not in owned or not eligible(p, definition):
            potential += credit
            if sid in owned: refunds.append(sid)
            continue
        level, progress, refund = allocate(credit, available)
        p['skills'][sid], p['skillLearningProgress'][sid] = level, progress
        potential += refund; kept_credit += credit - refund
    assert potential + kept_credit == old_total, 'potential conservation failed'
    attributes['potential'] = integer(potential, 'migrated potential')
    p['items'] = p.get('items') or {}; p['equipment'] = p.get('equipment') or {}
    silver = 0; manuals = {}
    for iid, count in list(p['items'].items()):
        integer(count, 'item quantity')
        if iid not in legacy['books']:
            if iid not in items: raise ValueError(f'unknown item: {iid}')
            continue
        del p['items'][iid]
        book = legacy['books'][iid]; target = legacy['skills'][book['skill']]['target']
        if not target: silver += count * (book['value'] // 2); continue
        dest = skills[target]['manualId']
        if count > 0:
            if p['skills'].get(target, 0) > 0 or dest in manuals:
                silver += count * (book['value'] // 2)
            else:
                manuals[dest] = 1
                silver += (count - 1) * (book['value'] // 2)
    p['items'].update(manuals)
    p['money'] = integer(integer(p.get('money', 0), 'money') + silver, 'compensated money')
    moved = []
    for slot, iid in sorted(p['equipment'].items(), key=lambda pair: pair[0] == 'HIDDEN_WEAPON'):
        if not iid: del p['equipment'][slot]; continue
        if iid not in items: raise ValueError(f'unknown equipped item: {iid}')
        destination = items[iid].get('equipmentSlot')
        if destination == slot: continue
        del p['equipment'][slot]
        if slot == 'HIDDEN_WEAPON' and destination == 'WEAPON' and not p['equipment'].get('WEAPON'):
            p['equipment']['WEAPON'] = iid
        else: p['items'][iid] = p['items'].get(iid, 0) + 1
        moved.append(iid)
    candidates = {}
    for old_id in (p.get('equippedSkills') or {}).values():
        if old_id not in legacy['skills']: raise ValueError(f'unknown equipped skill: {old_id}')
        sid = legacy['skills'][old_id]['target']
        if not sid or p['skills'].get(sid, 0) < 1: continue
        definition = skills[sid]
        if definition['slot'] == 'combat' and definition['weaponType'] != weapon_type(p, items): continue
        candidates.setdefault(definition['slot'], set()).add(sid)
    p['activeArts'] = {slot: next(iter(ids)) for slot, ids in candidates.items() if len(ids) == 1}
    p.pop('equippedSkills', None)
    p['learningSkillId'], p['learningTeacherId'], p['learningTeacherLevel'] = '', '', 0
    p['stateTickCountdown'], p['targetId'], p['combatType'] = 0, '', ''
    if p.get('state') != 'DEAD': p['state'] = 'IDLE'
    if p.get('roomId', '').startswith('dg_'): p['roomId'] = 'ly_00'
    p['martialVersion'] = VERSION
    clamp_resources(p, skills, items)
    return p, {'id': p['id'], 'name': p.get('characterName'), 'status': 'migrated',
               'oldSkills': old_skills, 'newSkills': p['skills'], 'refundedUnavailable': refunds,
               'potentialBefore': player['attributes'].get('potential', 0), 'potentialAfter': potential,
               'compensatedCopper': silver, 'returnedEquipment': moved, 'activeArts': p['activeArts']}


def run(data_dir, resources, apply=False, server_stopped=False, backup_dir=None):
    data_dir = Path(data_dir).resolve(); resources = Path(resources).resolve()
    if not (data_dir / 'players').is_dir(): raise ValueError('explicit data directory must contain players/')
    if apply and not server_stopped: raise ValueError('--apply requires --server-stopped; stop and flush the service first')
    if (data_dir / '.martial-migration.lock').exists(): raise ValueError('migration lock exists; inspect interrupted run before retry')
    legacy = read(resources / 'migrations/martial_v1.json')
    skills = read(resources / 'skills.json'); items = read(resources / 'items.json')
    skills = {sid: dict(skill, id=sid) for sid, skill in skills.items()}
    changes = []; reports = []
    for path in sorted((data_dir / 'players').glob('*.json')):
        if path.is_symlink(): raise ValueError('symlinked player files are not supported')
        original = path.read_bytes(); player = read(path)
        if path.stem != player['id']: raise ValueError('player filename/id mismatch')
        updated, report = migrate(player, legacy, skills, items, read(resources / 'item_aliases.json'))
        reports.append(report)
        if report['status'] != 'already_v2':
            payload = (json.dumps(updated, ensure_ascii=False, indent=2) + '\n').encode('utf-8')
            changes.append((path, original, payload))
    result = {'version': VERSION, 'mode': 'apply' if apply else 'preview', 'players': reports, 'changed': len(changes)}
    if not apply or not changes: return result
    backup = Path(backup_dir).resolve() if backup_dir else data_dir.parent / ('martial-v2-backup-' + datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ') + '-' + uuid.uuid4().hex[:8])
    if backup == data_dir or data_dir in backup.parents or backup.exists(): raise ValueError('backup must be a new directory outside the data directory')
    for path in data_dir.rglob('*'):
        if path.is_symlink(): raise ValueError('data snapshot must not follow symlinks')
    lock = data_dir / '.martial-migration.lock'
    with lock.open('x', encoding='utf-8') as stream: stream.write(str(os.getpid()))
    originals = []
    try:
        shutil.copytree(data_dir, backup / 'data', ignore=shutil.ignore_patterns('.martial-migration.lock'))
        result['backup'] = str(backup)
        result['hashes'] = {p.name: {'before': hashlib.sha256(old).hexdigest(), 'after': hashlib.sha256(new).hexdigest()} for p, old, new in changes}
        (backup / 'report.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
        with tempfile.TemporaryDirectory(prefix='.martial-v2-stage-', dir=data_dir.parent) as temporary:
            staging = Path(temporary)
            for path, old, new in changes: (staging / path.name).write_bytes(new)
            try:
                for path, old, new in changes:
                    originals.append((path, old)); os.replace(staging / path.name, path)
            except BaseException:
                for path, old in originals:
                    staged = staging / path.name; staged.write_bytes(old); os.replace(staged, path)
                raise
    finally:
        lock.unlink(missing_ok=True)
    return result


def main():
    if hasattr(sys.stdout, 'reconfigure'): sys.stdout.reconfigure(encoding='utf-8')
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--data-dir', required=True, type=Path)
    parser.add_argument('--resources', type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument('--apply', action='store_true')
    parser.add_argument('--server-stopped', action='store_true')
    parser.add_argument('--backup-dir', type=Path)
    args = parser.parse_args()
    print(json.dumps(run(args.data_dir, args.resources, args.apply, args.server_stopped, args.backup_dir), ensure_ascii=False, indent=2))


if __name__ == '__main__': main()
