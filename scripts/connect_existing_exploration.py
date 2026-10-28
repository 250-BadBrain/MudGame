"""Apply only the connected-exploration additions to existing resources.

Never calls the older content-depth editor or reads player saves. Replaces
only connection_* actions/options and the three journey metadata blocks.
"""
import copy
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RES = ROOT / 'MudGame-server/resources'


def read(path):
    return json.loads(path.read_text(encoding='utf-8'))


def write(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n', encoding='utf-8', newline='\n')


SKILLS = [
    ('datang_yunhe', 'dat_02', 'talang_jue', 'content_compare', 'checked', '以踏浪诀沿潮痕核更次',
     '你循踏浪诀辨水势的法门，把北散绳屑与闸盘水痕逐段相校，辨清折返相差的更次，无须再回码头核簿。闸盘细节与最终开闸方向仍须你亲自核验。', {'content_trace': 'known'}),
    ('dongming_chuanfen', 'cf_02', 'hunyuan_neigong', 'content_support', 'ready', '以混元功试梁稳住受力',
     '你先认清通气孔，再以混元功绵劲逐寸试出梁木的受力，把现有支点稳稳合拢，省去两步手工支护。压舱水怎样放，仍要结合舱梁痕迹决定。', {'content_detail': 'known'}),
    ('jiangnan_jianshou', 'jia_01', 'thunder_sword', 'content_trace', 'known', '以奔雷剑法辨枝口收锋',
     '你用奔雷剑法的发力与收势比对折枝，发现切口虽平，枝根却未受损。留下痕迹的人能发也能收，并未靠伤人立威；藏锋水台的两道剑痕仍待核验。', {}),
    ('datang_xb2', 'xb2_02', 'bloody_blade', 'content_detail', 'known', '以血战刀法辨绳势与桩力',
     '你按血战刀法辨刃口与磨耗的经验比对双桩绳纤：旧桩磨痕均匀，新桩的绳纤却被生拉成束，桩下浮土未实。你没有斩任何承重索；旧接头与支撑仍要另查。', {}),
    ('posui_zhongjuan', 'pos_02', 'wuzhen_pian', 'content_detail', 'known', '以悟真篇校读三相异文',
     '你依悟真篇校异文的方法，逐一比较图壁重复出现的笔势，人的轮廓始终正立，天地两形却互为倒像。留白的意义仍需结合残图判断，图壁也仍须按步骤操作。', {}),
    ('bashu_tangmen_trial', 'bas_01', 'tiebi_gong', 'content_trace', 'known', '以铁壁功护架试探回簧',
     '你以铁壁功的护架控制试探距离，只轻触空簧的传力处便撤手，辨出危险来自下一轮回弹而非毒色。你没有硬接暗器；牵丝机台的卸力次序仍须查清。', {}),
]

# Actor, chapter, title, (follow NPC, substitute action, room, label, response,
# scene message) for the two already-existing city conclusions.
CASES = [
    ('office_clerk', 'datang_yunhe', '密信与运河转运', [
        ('qingyi_spy', 'content_trace', 'dat_01', '凭匿名口信辨转运绳迹',
         '秦三娘把证人姓名隐去，只教你辨认转运队的旧绳结。去《运河追击》的断缆纤道，可凭口信核对前段痕迹；更次仍须另外查证。',
         '你依秦三娘的匿名口信对上北散绳屑：粮船曾逆流折返，证人身份始终没有暴露。你仍需核对更次与闸盘。'),
        ('office_clerk', 'content_compare', 'dat_02', '凭原卷更次核发船簿',
         '韩录事让你记下原卷中的更次旁注，不带走原件。去《运河追击》的双闸绞盘，先看纤道绳迹，再以旁注核更次，可省去回码头查簿。',
         '你将韩录事保留的更次旁注与北散绳屑相校，补全了发船簿的时间差，不必折返码头。闸盘刻号与最后开闸仍待判断。'),
    ]),
    ('bs_tang_steward', 'datang_xb2', '修路与绝壁悬栈', [
        ('bs_jiange_clerk', 'content_secure', 'xb2_02', '依驿丞修路方案分担承重',
         '剑阁驿丞把留窄路的分载法教给你：先辨哪根桩真正吃力，再把支点连回旧桩。去《绝壁悬栈》的承重双桩核验后，可以此方案直接稳住支护。',
         '你先验过双桩，再依驿丞的分载方案合上已有支点，让窄路保留回程。支护已稳，接索怎样处理仍须你结合旧接头判断。'),
        ('bs_wine_maker', 'content_trace', 'xb2_01', '凭封段见证辨旧接头',
         '巴蜀酒娘记得封段那天哪些赶路人被留在茶棚，也记得最早松动的旧接头。去《绝壁悬栈》的风剪悬栈，可用这份见证核前段绳迹；承重仍须现场处理。',
         '你照封段那日的见证比对旧接头，确认断索先磨损后崩开，并非被人挥刀斩断。双桩与支护仍待检查。'),
    ]),
    ('yz_yamen_clerk', 'yangzhou_neiluan', '三联旧账与盐帮内乱', [
        ('yz_boat_girl', 'content_detail', 'yan_02', '凭船娘水路证词核脚夫联',
         '船娘只替真实走过的水路作证，不交出脚夫姓名。去《盐帮内乱》的三联账房，可以搬盐的零碎尾数核脚夫联；破角盐引仍须另查。',
         '你以船娘记下的船次与尾数核账，脚夫联有实际搬运记录，官联商联却互相照抄。匿名证人被保全，盐引和最后取舍仍须现场判断。'),
        ('yz_salt_merchant', 'content_trace', 'yan_01', '凭三联封存记录查破角盐引',
         '程盐商交代封存时逐担验引的顺序，也承认原封记录不能只由他一人解释。去《盐帮内乱》的分盐旧秤，可以记录辨破角盐引；三联账仍须另核。',
         '你依三联封存时的验引顺序逐担比对，工人盐引有破角，堂口收到的却张张完好。三份账的责任仍须另外查证。'),
    ]),
]


def main():
    dialogues, npcs = read(RES / 'dialogues.json'), read(RES / 'npcs.json')
    chapters = {p.stem: read(p) for p in (RES / 'dungeons').glob('*.json')}
    before = copy.deepcopy(chapters)
    for d in dialogues.values():
        d['options'] = [o for o in d['options'] if not o['id'].startswith('connection_')]
    changed = set()

    def add(cid, aid, room, base, label, message, **extra):
        d = chapters[cid]
        a = copy.deepcopy(d['puzzleActions'][base])
        a.update(label=label, message=message, repeatMessage=message, rooms=[room], **extra)
        a['setFlags'] = extra.get('setFlags', {})
        d['puzzleActions'][aid] = a
        if aid not in d['rooms'][room]['actions']:
            d['rooms'][room]['actions'].append(aid)
        d['rooms'][room]['description'] = d['rooms'][room]['description'].split('【旧事与武学】')[0]
        changed.add(cid)
        return a

    for cid, room, skill, puzzle, state, label, message, pre in SKILLS:
        base = {'content_compare': 'content_compare', 'content_support': 'content_secure'}.get(puzzle, puzzle)
        a = add(cid, 'connection_skill', room, base, label, message,
                puzzleId=puzzle, fromStates=['unseen', 'bracing'] if puzzle == 'content_support' else ['unseen'],
                nextState=state, requiredSkills={skill: 1}, requiredPuzzleStates=pre,
                setFlags={'discovery_skill_route_' + cid: True})
        a['failureMessage'] = '须正式学会对应武学并满足原入门资格；先完成动作提示中的现场线索。也可继续普通调查路线。'

    for actor, cid, title, branches in CASES:
        d = dialogues[npcs[actor]['dialogueId']]
        left = next(o for o in d['options'] if o['id'] == 'content_case_left')
        witnesses = [k.removeprefix('discovery_').removesuffix('_specialty') for k, v in left['requires']['flags'].items() if v]
        journey = dict(title=title, dungeonId=cid, startFlag='discovery_' + actor + '_specialty',
                       witnesses=witnesses, itemId=next(iter(left['requires']['presentItems'])), branches=[])
        for side, (npc, base, room, label, response, scene) in zip(['left', 'right'], branches):
            case_flag = 'discovery_case_' + actor + '_' + side
            follow_flag = 'discovery_connection_follow_' + actor + '_' + side
            used_flag = 'discovery_connection_used_' + actor + '_' + side
            oid, aid = 'connection_follow_' + actor + '_' + side, 'connection_case_' + side
            target = dialogues[npcs[npc]['dialogueId']]
            target['options'].append(dict(id=oid, text='追问旧事：' + title + '的后续', response=response,
                                         explicit=True, requires={'flags': {case_flag: True}}, setFlags={follow_flag: True}))
            option = next(o for o in d['options'] if o['id'] == 'content_case_' + side)
            option['response'] = option['response'].split('【后续】')[0] + '【后续】可向' + npcs[npc]['name'] + '明确追问“' + title + '的后续”，取得可用于《' + chapters[cid]['name'] + '》的线索。'
            a = add(cid, aid, room, base, label, scene, requiredFlags={case_flag: True, follow_flag: True},
                    setFlags={used_flag: True})
            if base == 'content_secure':
                a.update(fromStates=['unseen', 'bracing'], requiredPuzzleStates={'content_detail': 'known'})
            a['failureMessage'] = '须先完成所选城市旧事，并向对应人物明确追问后续；现场前置调查也须完成。普通解法始终保留。'
            journey['branches'].append(dict(flag=case_flag, npcId=npc, optionId=oid, followFlag=follow_flag, actionId=aid, usedFlag=used_flag))
        d['journey'] = journey

    # Always explain optional routes in the physical scene, even before learned.
    for cid in changed:
        d = chapters[cid]
        if cid == 'datang_yunhe':
            # All three ways of verifying the watch records lead to the same
            # judgment; the label must not claim the player walked back.
            for a in d['puzzleActions'].values():
                a['label'] = a['label'].replace('回码头核更次后', '核清更次后')
                for choice in a.get('choices', []):
                    choice['label'] = choice['label'].replace('回码头核更次后', '核清更次后')
        for rid, room in d['rooms'].items():
            optional = [a for aid,a in d['puzzleActions'].items() if aid.startswith('connection_') and rid in a['rooms']]
            if optional:
                room['description'] = room['description'].split('【旧事与武学】')[0] + '【旧事与武学】此处也可尝试：' + '；'.join(a['label'] for a in optional) + '。需具备对应已学武学或亲自追问所得情报；普通调查仍可独立完成。'
        assert d['rewards'] == before[cid]['rewards'] and d.get('quickRewards') == before[cid].get('quickRewards')
        assert set(d['rooms']) == set(before[cid]['rooms'])
        write(RES / 'dungeons' / (cid + '.json'), d)
    write(RES / 'dialogues.json', dialogues)
    print('Connected 3 existing city cases (6 branches), 6 martial routes across 7 existing chapters; rewards unchanged.')


if __name__ == '__main__':
    main()
