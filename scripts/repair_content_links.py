"""Historical 2026-10-27 content repair, not a current resource rebuild.

Do not rerun against current resources: its recipe values predate
docs/design/audit_repairs_2026_10_28.md. Maintain current resources directly.
"""
import copy
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RES = ROOT / 'MudGame-server/resources'


def read(path):
    return json.loads(path.read_text(encoding='utf-8'))


def write(path, data):
    newline = '\r\n' if path.name == 'npcs.json' else '\n'
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + '\n', encoding='utf-8', newline=newline)


def main():
    quests = read(RES / 'quests.json'); npcs = read(RES / 'npcs.json'); dialogues = read(RES / 'dialogues.json')
    dungeons = {p: read(p) for p in (RES / 'dungeons').glob('*.json')}
    aliases = {'xinggong_after': 'xinggong_aftermath', 'kongming_dao': 'kongming_daozhen', 'longgong_yi': 'longgong_yiji'}
    for alias, target in aliases.items(): quests[alias]['aliasOf'] = target
    for k in ['zhongjuan_daozhen', 'kongming_daozhen']:
        quests[k]['giverNpcId'] = quests[k]['turnInNpcId'] = 'ca2_monk'
    quests['luoyang_bounty']['giverNpcId'] = quests['luoyang_bounty']['turnInNpcId'] = 'office_clerk'
    quests['luoyang_bounty']['description'] = '北城官署悬赏三两银，请你在《洛阳粮道》挫败瓦岗偏师，满通关结算后回来复命。'
    quests['zhongjuan_daozhen'].update(name='遗境道真', description='将《战神遗境》的机关见闻带给长安化度寺知客僧辨读。')
    quests['kongming_daozhen']['description'] = '完成《破碎空明》的探索，将所见带回长安和春亭，向化度寺知客僧复命。'
    quests['yangzhou_bounty'].update(name='枯井缉邪', description='盐运司书办请你追查上游暗桩，在《枯井之下》满通关结算后回盐运使署报告。')
    quests['yunhe_houyi']['description'] = '将《运河追击》的战况和水路见闻带回扬州盐运使署，供书办安排巡防。'
    quests['saibei_treasure'].update(name='狼山商路', description='回鹘马商请你打通《狼山口》，满通关结算后回边关马市领取开路酬谢。')
    targets = ['datang_xb2', 'fuyu_ht2', 'posui_sh2', 'bianhuang_dz2', 'luoshui_tg2', 'yangzhou_ym2', 'datang_tx2', 'fuyu_yy2', 'posui_th2', 'bianhuang_bf2']
    for index, target in enumerate(targets): quests[f'quest_r204_{index}']['objectives'][0]['targetId'] = 'dungeon_cleared_' + target
    source_flags = {'zhandao_case_done': 'datang_zhandao', 'langshan_case_done': 'saibei_langshan', 'jianchi_case_done': 'jiangnan_jianchi', 'tianjin_case_done': 'dongdu_tianjin', 'haisi_case_done': 'dongming_haisi'}
    by_id = {d['id']: d for d in dungeons.values()}
    for qid, quest in quests.items():
        if qid in aliases: continue
        for objective in quest['objectives']:
            flag = objective['targetId']
            target = flag.removeprefix('dungeon_cleared_') if flag.startswith('dungeon_cleared_') else source_flags.get(flag)
            if target:
                quest['sourceDungeon'] = target
                objective['targetId'] = 'dungeon_cleared_' + target
                objective['description'] = f'完成《{by_id[target]["name"]}》的调查目标并满通关离开结算。'
        if quest.get('sourceDungeon') and (quest['description'] == quest['name'] or qid.startswith('quest_r204_')):
            name = by_id[quest['sourceDungeon']]['name']
            quest['description'] = f'完成《{name}》并满通关离开后，将见闻带回{npcs[quest["turnInNpcId"]]["name"]}处，领取一次性酬谢。'
    # The existing marsh clue remains valid; now the promised witness route is real.
    quests['caoyin_case']['objectives'] = [
        {'type': 'flag', 'targetId': 'caoyin_boat_witness', 'description': '向二十四桥船娘询问银箱的去向。', 'required': 1},
        {'type': 'flag', 'targetId': 'caoyin_office_witness', 'description': '向盐运司书办核对封条，取得搜查方向。', 'required': 1},
        {'type': 'flag', 'targetId': 'found_caoyin_clue', 'description': '在苇荡搜索空银箱及官印拓痕。', 'required': 1},
    ]
    quests['caoyin_case']['description'] = '程老爷请你询问二十四桥船娘、核对盐运司书办的记录，再去苇荡搜索银箱凭据，回来报告漕银下落。'
    witness_lines = [
        ('dlg_yz_boat', 'caoyin_boat_witness', '船娘指向苇荡：“那夜卸下的空箱子是官船制式，吃水却比来时浅。去问书办封条上的号。”'),
        ('dlg_yz_clerk', 'caoyin_office_witness', '书办核对封条：“箱号对得上。若要实证，去苇荡搜索泥滩，光凭传闻我不能销账。”'),
    ]
    for did, flag, line in witness_lines:
        options = dialogues[did]['options']; options[:] = [o for o in options if o['id'] != 'caoyin_witness']
        options.insert(0, {'id': 'caoyin_witness', 'text': '询问漕银失窃的线索', 'response': line, 'setFlags': {flag: True}, 'requires': {'questId': 'caoyin_case', 'questStatus': 'active'}})
    # Materialize the person previously impersonated inside the old refugee's dialogue.
    old_options = dialogues['dlg_old_refugee']['options']
    clue = next((o for o in old_options if o['id'] == 'gather_rumor'), None)
    if clue is None:
        clue = next((o for o in dialogues.get('dlg_mute_refugee', {}).get('options', []) if o['id'] == 'gather_rumor'), None)
    if clue in old_options: old_options.remove(clue)
    npc = copy.deepcopy(npcs['old_refugee']); npc.update(name='哑婆', title='营地南角', description='她蜷在营地南角的窝棚旁，手边放着一截炭条。', dialogueId='dlg_mute_refugee', questIds=['refugee_rumor_case'])
    npc.pop('id', None); npcs['mute_refugee_woman'] = npc
    dialogues['dlg_mute_refugee'] = {'id': 'dlg_mute_refugee', 'npcId': 'mute_refugee_woman', 'title': '炭痕里的线索', 'opening': '哑婆抬头看了你一眼，捡起地上的炭条。', 'options': [clue or {'id': 'gather_rumor', 'text': '向哑婆打听白衣娘娘', 'response': '她在地上画出一口井、一条河和一个倒写的“隋”字，随即指向老瘸子的窝棚。', 'setFlags': {'heard_refugee_whisper': True}, 'requires': {'questId': 'refugee_rumor_case', 'questStatus': 'active'}}]}
    world = read(RES / 'worlds/luoyang.json'); room = world['rooms']['ly_refugee_camp']
    if 'mute_refugee_woman' not in room['npcs']: room['npcs'].append('mute_refugee_woman')
    write(RES / 'worlds/luoyang.json', world)
    search_points = {
        'yz_reed_marsh': ('found_caoyin_clue', '你在苇荡深处的泥滩上找到空银箱，箱底官印拓痕与盐运司的封条相符，留下了漕银被搬走的证据。'),
        'ly_salt_wharf': ('found_smuggling_evidence', '你伏在盐垛后，记下把官盐搬上无籍小船的船号，船帮上刻着歪斜的鹞子记号。'),
        'tg_jingou_beacon': ('found_beacon_signal', '你在未点燃的整齐柴堆旁，捡到一枚被扯掉的戍卒号牌：断讯并非无人值守。'),
    }
    for path in (RES / 'worlds').glob('*.json'):
        world = read(path)
        for rid, room in world['rooms'].items():
            if rid in search_points: room['searchFlag'], room['searchMessage'] = search_points[rid]
            for enabled, action in [(bool(room.get('rumors')), 'listen_rumor'), (bool(room.get('searchFlag')), 'search')]:
                if enabled and action not in room.setdefault('actions', []): room['actions'].append(action)
        write(path, world)
    # Explicit start/turn-in options, preserving the four authored quest conversations.
    for qid, quest in quests.items():
        if qid in aliases: continue
        for phase, key, field in [('start', 'giverNpcId', 'startQuestId'), ('complete', 'turnInNpcId', 'completeQuestId')]:
            nid = quest[key]; npc = npcs[nid]
            if not npc.get('dialogueId'):
                npc['dialogueId'] = f'dlg_{nid}_commissions'
                dialogues[npc['dialogueId']] = {'id': npc['dialogueId'], 'npcId': nid, 'title': '委托与复命', 'opening': f'{npc["name"]}将手边的记录翻开：“有事便说，办妥的事也可来此复命。”', 'options': []}
            dialogue = dialogues[npc['dialogueId']]
            npc.setdefault('questIds', [])
            if qid not in npc['questIds']: npc['questIds'].append(qid)
            if any(o.get(field) == qid for o in dialogue['options']): continue
            response = quest['description'] if phase == 'start' else f'{npc["name"]}核对了你的见闻：“《{quest["name"]}》这件事办妥了，酬谢收好。”'
            dialogue['options'].append({'id': f'quest_{phase}_{qid}', 'text': ('接取：' if phase == 'start' else '交付：') + quest['name'], 'response': response, field: qid, 'requires': {'questId': qid, 'questStatus': 'not_started' if phase == 'start' else 'active'}})
    for did in ['dlg_herbalist_sun', 'dlg_heluo_blacksmith']:
        for option in dialogues[did]['options']:
            if option.get('takeItems') or option.get('takeMoney'): option['repeatable'] = True
    recipes = {
        'brew_herb_pill': ({'herb': 5, 'dry_ration': 2}, 0, '回春丹'),
        'brew_dahuandan': ({'herb': 8, 'dry_ration': 4}, 150, '大还丹'),
        'brew_te_huan': ({'dahuandan': 2, 'dry_ration': 5}, 300, '特级大还丹'),
        'reforge_sword': ({'iron_sword': 1}, 100, '钢刀'),
        'reforge_staff': ({'iron_staff': 1}, 150, '钢刀'),
        'forge_ring': ({'bronze_fragment': 2}, 200, '玄铁战指'),
    }
    items = read(RES / 'items.json')
    for dialogue in dialogues.values():
        for option in dialogue['options']:
            if option['id'] in recipes:
                ingredients, money, product = recipes[option['id']]
                option['takeItems'] = ingredients; option['requires'] = {'items': ingredients}; option['takeMoney'] = money
                costs = '、'.join(f'{items[k]["name"]}×{v}' for k, v in ingredients.items()) + (f'、{money}文钱' if money else '')
                option['text'] = f'制作{product}（{costs}）'; option['response'] = f'你交付{costs}，制成{product}。'
            elif option.get('takeMoney'):
                option['text'] = option['text'].replace('100两', '100文钱')
    linked = 0
    for path, dungeon in dungeons.items():
        for aid, action in dungeon.get('puzzleActions', {}).items():
            if not aid.startswith('listen_brief') or action.get('dialogueNpcId'): continue
            for rid in action.get('rooms', []):
                actions = dungeon['rooms'][rid].setdefault('actions', [])
                if aid not in actions: actions.append(aid); linked += 1
        if dungeon['id'] == 'youming_guishi':
            action = dungeon['puzzleActions']['listen_brief_yg']
            action['message'] = '引路人竖起三根手指：“商路、侯路、武路，各有守关人。三路都可探索，也可以完成其中一路后离开；想取收藏，就得先过各自的守关人。”'
        if dungeon['id'] == 'tianbei_beidou':
            dungeon['description'] = '天北峰的北斗古阵，七步依次串联，全部走通后星盘现世。越序触碰不生效，已完成步骤保留。'
            room = dungeon['rooms']['bd_01']
            room['description'] = '北斗阵殿七面石壁各嵌星石，须依次触发天枢、天璇、天玑、天权、玉衡、开阳、摇光。越序触碰不生效，已完成步骤保留。'
            for aid, star in zip(['tianshu', 'jumen', 'lucun', 'wenqu', 'lianzhen', 'wuqu', 'pojun'], ['天枢', '天璇', '天玑', '天权', '玉衡', '开阳', '摇光']):
                action = dungeon['puzzleActions'][aid]; action['label'] = f'触动{star}星位'; action['message'] = f'{star}星位亮起，星光在阵中连成一道新的轨迹。'
        write(path, dungeon)
    write(RES / 'quests.json', quests); write(RES / 'dialogues.json', dialogues); write(RES / 'npcs.json', npcs)
    print(f'Connected {len(quests)-len(aliases)} unique quests; retained {len(aliases)} aliases; bound {linked} briefings.')


if __name__ == '__main__': main()
