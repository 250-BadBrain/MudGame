"""Connected choices/learning acceptance on fresh synthetic saves and ports."""
import argparse
import contextlib
import json
import socket
import tempfile
from pathlib import Path

from smoke_backend import WebSocket
from smoke_content_depth import room, clear_hostiles, walk_instance
from smoke_content_review import start, stop
from smoke_dungeon_quick import USER, PASSWORD, connect
from smoke_martial_v2 import public_data, raw_call, walk

ROOT = Path(__file__).resolve().parents[2]
RES = ROOT / 'MudGame-server/resources'


def load(path):
    return json.loads(path.read_text(encoding='utf-8'))


def journal(client):
    return client.call('command', 'get_quests')['journal']


def run(binary, directory):
    checks = []
    rooms, npcs = public_data()
    room_world = {rid:w['id'] for p in (RES/'worlds').glob('*.json') for w in [load(p)] for rid in w['rooms']}
    dialogues = load(RES / 'dialogues.json')
    skills = load(RES / 'skills.json')
    chapters = {p.stem: load(p) for p in (RES / 'dungeons').glob('*.json')}
    journeys = [d for d in dialogues.values() if d.get('journey')]
    routes = [(d,a) for d in chapters.values() for a in d['puzzleActions'].values() if a.get('requiredSkills')]
    assert len(journeys) == 3 and len(routes) == 6
    with socket.socket() as probe:
        probe.bind(('127.0.0.1', 0)); port = probe.getsockname()[1]
    assert port != 4009
    process, client = start(binary, directory, port), None
    try:
        client = WebSocket(port)
        client.call('system','register',username=USER,password=PASSWORD)
        client.call('system','login',username=USER,password=PASSWORD)
        roles = [client.call('system','create_character',characterName=n,strength=16,constitution=16,agility=16,perception=16,willpower=16)['characterId'] for n in ['衔接左线侠','衔接右线侠','衔接页面侠']]
    finally:
        if client: client.close()
        stop(process)
    # Only newly-created, stopped synthetic saves are edited. High attributes
    # exercise real movement/combat, not normal-player combat balance.
    for role in roles:
        path=directory/'saves/players'/(role+'.json'); p=load(path)
        p['items'].update(sealed_letter=1,reed_root=1,iron_pickaxe=1)
        p['attributes']['base'].update(strength=100000,constitution=100000,agility=100000,perception=100000,willpower=100000)
        p['attributes']['resources'].update(hp=400000,maxHp=400000,mp=80)
        for cid in chapters: p['flags']['dungeon_cleared_'+cid]=True
        for sid,defn in skills.items():
            if defn['type']=='BASIC': p['skills'][sid]=20
        if role==roles[2]:
            p['flags'].pop('dungeon_cleared_datang_yunhe')
            p['skills']['talang_jue']=1; p['flags']['martial_taught_talang_jue']=True
        path.write_text(json.dumps(p,ensure_ascii=False),encoding='utf-8')

    def visit(npc):
        rid=next(rid for rid,r in rooms.items() if npc in r.get('npcs',[]))
        current=room(client)['id']
        if room_world[current] != room_world[rid]:
            client.call('command','enter_world',worldId=room_world[rid])
        walk(client,rid,rooms)

    process, client = start(binary,directory,port),None
    try:
        for index,side in enumerate(['left','right']):
            client=connect(port,roles[index]); assert journal(client)==[]
            for d in journeys:
                j=d['journey']; b=j['branches'][index]; cid=j['dungeonId']
                visit(b['npcId'])
                assert not raw_call(client,'command','choose_dialogue',npcId=b['npcId'],optionId=b['optionId'])['flag']
                for npc in j['witnesses']:
                    visit(npc); client.call('command','choose_dialogue',npcId=npc,optionId='content_specialty')
                visit(d['npcId'])
                client.call('command','choose_dialogue',npcId=d['npcId'],optionId='content_case_'+side)
                assert not raw_call(client,'command','choose_dialogue',npcId=d['npcId'],optionId='content_case_'+('right' if side=='left' else 'left'))['flag']
                visit(b['npcId'])
                for _ in range(2): client.call('command','talk',npcId=b['npcId'])
                assert not any(e['id']==b['followFlag'] for e in journal(client))
                client.call('command','choose_dialogue',npcId=b['npcId'],optionId=b['optionId'])
                client.call('command','enter_dungeon',dungeonId=cid)
                a=chapters[cid]['puzzleActions'][b['actionId']]
                for puzzle,state in a.get('requiredPuzzleStates',{}).items():
                    observe=next((aid, act) for aid,act in chapters[cid]['puzzleActions'].items() if not aid.startswith(('connection_','content_aid_')) and act['puzzleId']==puzzle and act.get('nextState')==state)
                    walk_instance(client,chapters[cid],observe[1]['rooms'][0]); client.call('command','do_action',action=observe[0])
                walk_instance(client,chapters[cid],a['rooms'][0])
                result=client.call('command','do_action',action=b['actionId'])
                assert result['room']['dungeon']['puzzleStates'][a['puzzleId']]==a['nextState'], (cid,side,result)
                card=next(e for e in journal(client) if e['id']=='journey_'+d['npcId'])
                assert card['state']=='已在现场互证' and card['progress']=='4/4'
                assert any(e['id']==b['usedFlag'] for e in journal(client))
                client.call('command','leave_dungeon')
                checks.append(d['npcId']+'/'+side+': real witnesses, exclusive choice, explicit follow-up and scene consequence')
            client.close(); client=None

        client=connect(port,roles[0])
        for d,a in routes:
            sid=next(iter(a['requiredSkills'])); cid=d['id']
            client.call('command','enter_dungeon',dungeonId=cid)
            walk_instance(client,d,a['rooms'][0]); clear_hostiles(client)
            rejected=client.call('command','do_action',action='connection_skill')
            assert rejected['room']['dungeon']['puzzleStates'][a['puzzleId']] != a['nextState'] and '须正式学会' in rejected['message'], (cid,rejected)
            assert 'connection_skill' not in room(client)['actions']
            client.call('command','leave_dungeon')
            teacher=skills[sid]['teacherId']; visit(teacher)
            client.call('command','learn',npcId=teacher,skillId=sid)
            client.call('command','enter_dungeon',dungeonId=cid)
            for puzzle,state in a.get('requiredPuzzleStates',{}).items():
                aid,obs=next((aid,act) for aid,act in d['puzzleActions'].items() if aid.startswith('content_') and not aid.startswith('content_aid_') and act['puzzleId']==puzzle and act.get('nextState')==state)
                walk_instance(client,d,obs['rooms'][0]); client.call('command','do_action',action=aid)
            walk_instance(client,d,a['rooms'][0]); clear_hostiles(client)
            before=client.call('command','get_attributes')['attributes']
            money=client.call('command','get_backpack')['money']
            result=client.call('command','do_action',action='connection_skill')
            after=client.call('command','get_attributes')['attributes']
            assert result['room']['dungeon']['puzzleStates'][a['puzzleId']]==a['nextState'], (cid,result)
            assert after['currentMp']==before['currentMp'] and client.call('command','get_backpack')['money']==money
            client.call('command','leave_dungeon')
            checks.append(cid+': unlearned denied, actually taught art enables alternate scene step without charge')
        before_restart=journal(client)
    finally:
        if client: client.close()
        stop(process)
    process,client=start(binary,directory,port),None
    try:
        client=connect(port,roles[0]); assert journal(client)==before_restart
        client.call('system','select_character',characterId=roles[2])
        client.call('command','enter_world',worldId='luoyang'); walk(client,'ly_old_alley',rooms)
        assert journal(client)==[]
        checks.append('saved histories and derived next steps survive restart; fresh browser role prepared')
    finally:
        if client: client.close()
        stop(process)
    return dict(status='PASS',checks=checks,syntheticCombat=True,uiPlayer=roles[2])


if __name__=='__main__':
    parser=argparse.ArgumentParser(); parser.add_argument('--binary',type=Path,required=True); parser.add_argument('--report',type=Path,required=True); parser.add_argument('--keep-fixture',type=Path); args=parser.parse_args()
    if args.keep_fixture:
        directory=args.keep_fixture.resolve(); allowed=(ROOT/'artifacts/quick-complete').resolve()
        assert directory.is_relative_to(allowed) and directory!=allowed and not directory.exists()
        directory.mkdir(parents=True); context=contextlib.nullcontext(str(directory))
    else: context=tempfile.TemporaryDirectory(prefix='mud-connected-exploration-')
    with context as temporary:
        result=run(args.binary.resolve(),Path(temporary)); args.report.parent.mkdir(parents=True,exist_ok=True)
        args.report.write_text(json.dumps(result,ensure_ascii=False,indent=2),encoding='utf-8'); print(json.dumps(result,ensure_ascii=False))
