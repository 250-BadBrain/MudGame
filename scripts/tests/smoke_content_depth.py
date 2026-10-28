"""Content-depth protocol acceptance, exclusively with freshly generated saves.

Late combat uses a synthetic high-attribute role to test routes quickly; this
does not measure combat balance or demonstrate normal character progression.
"""
import argparse
import collections
import contextlib
import json
import socket
import tempfile
import time
from pathlib import Path

from smoke_backend import WebSocket
from smoke_content_review import start, stop
from smoke_dungeon_quick import USER, PASSWORD, connect
from smoke_martial_v2 import raw_call, public_data, walk

ROOT=Path(__file__).resolve().parents[2]
RES=ROOT/'MudGame-server/resources'

def room(client): return client.call('system','get_room')['room']

def clear_hostiles(client):
    while True:
        here=room(client)
        targets=[n for n in here['npcs'] if n.get('combatMode') in ['kill','defeat'] and n.get('state') not in ['DEAD','DEFEATED']]
        if not targets:return
        client.call('command','kill',targetId=targets[0]['id'])
        for _ in range(120):
            here=room(client)
            actor=next(p for p in here['playersInRoom'] if p.get('online'))
            assert actor['state']!='DEAD','synthetic role died'
            if actor['state']=='IDLE':break
            time.sleep(.1)
        else:raise AssertionError('combat did not end')

def walk_instance(client,dungeon,target):
    here=room(client)
    current=next(k for k in dungeon['rooms'] if here['id'].endswith('_'+k))
    todo=collections.deque([(current,[])]);seen={current}
    while todo:
        rid,path=todo.popleft()
        if rid==target:
            for direction in path:
                clear_hostiles(client)
                client.call('command','go',direction=direction)
            return
        for direction,destination in dungeon['rooms'][rid].get('exits',{}).items():
            if destination not in seen:seen.add(destination);todo.append((destination,path+[direction]))
    raise AssertionError('no instance path')

def run(binary,directory):
    checks=[];rooms,npcs=public_data()
    with socket.socket() as probe:probe.bind(('127.0.0.1',0));port=probe.getsockname()[1]
    assert port!=4009
    process,client=start(binary,directory,port),None
    try:
        client=WebSocket(port);client.call('system','register',username=USER,password=PASSWORD);client.call('system','login',username=USER,password=PASSWORD)
        roles=[client.call('system','create_character',characterName=name,strength=16,constitution=16,agility=16,perception=16,willpower=16)['characterId'] for name in ['见闻验收侠','见闻页面侠']]
    finally:
        if client:client.close()
        stop(process)
    for role in roles:
        path=directory/'saves/players'/(role+'.json');p=json.loads(path.read_text(encoding='utf-8'))
        p['items'].update(iron_sword=1,common_clothes=1,sealed_letter=1,dry_ration=5,smoke_powder=10)
        for file in (RES/'dungeons').glob('*.json'):p['flags']['dungeon_cleared_'+json.loads(file.read_text(encoding='utf-8'))['id']]=True
        if role==roles[0]:
            p['attributes']['base'].update(strength=100000,constitution=100000,agility=100000,perception=100000,willpower=100000)
            p['attributes']['resources'].update(hp=400000,maxHp=400000,mp=80)
        path.write_text(json.dumps(p,ensure_ascii=False),encoding='utf-8')
    process,client=start(binary,directory,port),None
    try:
        client=connect(port,roles[0]);catalog=client.call('command','get_dungeons')['dungeons']
        assert len(catalog)==78 and sum(d.get('contentStatus')=='enriched' for d in catalog)==40
        journal=client.call('command','get_quests')['journal'];assert journal==[]
        checks.append('existing directory retained; 40 expanded statuses; no unseen discoveries in journal')
        for npc in ['qingyi_spy','luoyang_guard']:
            target=next(rid for rid,r in rooms.items() if npc in r.get('npcs',[]));walk(client,target,rooms)
            for _ in range(3):
                spoken=client.call('command','talk',npcId=npc)
                assert any(o['id']=='content_specialty' for o in spoken['dialogue']['options']), 'wire response lost explicit choices'
            assert all(e['id']!='discovery_'+npc+'_specialty' for e in client.call('command','get_quests')['journal'])
            client.call('command','choose_dialogue',npcId=npc,optionId='content_specialty')
        walk(client,'ly_north_office',rooms)
        client.call('command','choose_dialogue',npcId='office_clerk',optionId='content_case_left')
        assert not raw_call(client,'command','choose_dialogue',npcId='office_clerk',optionId='content_case_right')['flag']
        walk(client,'ly_refugee_camp',rooms);client.call('command','choose_dialogue',npcId='old_refugee',optionId='content_after_office_clerk_left')
        checks.append('distinct witnesses must be explicitly consulted; one case conclusion excludes its opposite; aftermath available')
        walk(client,'ly_east_market',rooms);client.call('command','equip',itemId='common_clothes')
        client.call('command','choose_dialogue',npcId='east_market_keeper',optionId='content_show_common_clothes')
        assert client.call('command','get_equipment')['equipment']['BODY']['id']=='common_clothes'
        assert not raw_call(client,'command','choose_dialogue',npcId='east_market_keeper',optionId='content_show_shenhai_zhenzhu')['flag']
        checks.append('equipped item is valid nonconsuming evidence; forged unowned/unoffered evidence denied')
        for cid in ['datang_yunhe','fuyu_mishi','dongming_chuanfen','yangzhou_neiluan']:
            dungeon=json.loads((RES/'dungeons'/(cid+'.json')).read_text(encoding='utf-8'))
            client.call('command','enter_dungeon',dungeonId=cid)
            assert not raw_call(client,'command','do_action',action='content_decide_good')['flag']
            for step in dungeon['contentFeature']['solution']:
                walk_instance(client,dungeon,step['roomId']);clear_hostiles(client)
                if step['actionId']=='content_decide_good':
                    client.call('command','do_action',action='content_decide_other')
                    assert room(client)['dungeon']['puzzleStates']['gate']=='closed'
                    client.call('command','do_action',action='open_gate')
                client.call('command','do_action',action=step['actionId'])
            assert room(client)['dungeon']['puzzleStates']['gate']=='open'
            assert any(e['id']=='discovery_chapter_'+cid for e in client.call('command','get_quests')['journal'])
            client.call('command','leave_dungeon')
            checks.append(cid+': actual movement/combat, observations, wrong-choice recovery and final decision passed')
        client.call('command','enter_world',worldId='luoyang')
        walk(client,'ly_old_alley',rooms)
        journal_before=client.call('command','get_quests')['journal']
    finally:
        if client:client.close()
        stop(process)
    process,client=start(binary,directory,port),None
    try:
        client=connect(port,roles[0]);assert client.call('command','get_quests')['journal']==journal_before
        client.call('system','select_character',characterId=roles[1]);walk(client,'ly_old_alley',rooms)
        checks.append('journal survives clean restart; separate untouched UI role prepared with no discovered clues')
    finally:
        if client:client.close()
        stop(process)
    return dict(status='PASS',checks=checks,syntheticCombat=True,player=roles[0],uiPlayer=roles[1])

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--binary',type=Path,required=True);parser.add_argument('--report',type=Path,required=True);parser.add_argument('--keep-fixture',type=Path);args=parser.parse_args()
    if args.keep_fixture:
        directory=args.keep_fixture.resolve();allowed=(ROOT/'artifacts/quick-complete').resolve()
        assert directory.is_relative_to(allowed) and directory!=allowed and not directory.exists();directory.mkdir(parents=True);context=contextlib.nullcontext(str(directory))
    else:context=tempfile.TemporaryDirectory(prefix='mud-content-depth-')
    with context as temporary:
        report=run(args.binary.resolve(),Path(temporary));args.report.parent.mkdir(parents=True,exist_ok=True);args.report.write_text(json.dumps(report,ensure_ascii=False,indent=2),encoding='utf-8');print(json.dumps(report,ensure_ascii=False))
