"""Isolated real-protocol acceptance. Creates and edits only synthetic test saves."""
import argparse
import contextlib
import json
import os
import signal
import socket
import subprocess
import tempfile
import time
import urllib.request
from pathlib import Path

from smoke_backend import WebSocket
from smoke_martial_v2 import public_data, raw_call, walk

ROOT = Path(__file__).resolve().parents[2]
RESOURCES = ROOT / 'MudGame-server/resources'
USERNAME = 'content_review_ui'
PASSWORD = 'temporary-content-pass'


def start(binary, directory, port):
    env = dict(os.environ, PORT=str(port), MUD_DATA_DIR=str(directory / 'saves'), MUD_RESOURCES_DIR=str(RESOURCES))
    flags = subprocess.CREATE_NEW_PROCESS_GROUP if os.name == 'nt' else 0
    with (directory / 'process.log').open('ab') as log:
        process = subprocess.Popen([str(binary)], cwd=directory, env=env, stdout=log, stderr=log, creationflags=flags)
    try:
        for _ in range(100):
            if process.poll() is not None: raise AssertionError((directory / 'process.log').read_text(encoding='utf-8'))
            try:
                with urllib.request.urlopen(f'http://127.0.0.1:{port}/health', timeout=.2) as response:
                    assert json.load(response)['ok']; return process
            except OSError: time.sleep(.05)
        raise AssertionError('startup timed out')
    except BaseException:
        process.kill(); process.wait(); raise


def stop(process):
    if process.poll() is not None: return
    process.send_signal(signal.CTRL_BREAK_EVENT if os.name == 'nt' else signal.SIGINT)
    try: assert process.wait(timeout=10) == 0, 'backend did not flush and shut down cleanly'
    finally:
        if process.poll() is None: process.kill(); process.wait()


def connect(port, player=None):
    client = WebSocket(port)
    client.call('system', 'login', username=USERNAME, password=PASSWORD)
    if player: client.call('system', 'select_character', characterId=player)
    return client


def run(binary, directory):
    checks = []; rooms, _ = public_data()
    with socket.socket() as probe:
        probe.bind(('127.0.0.1', 0)); port = probe.getsockname()[1]
    assert port != 4009
    process = start(binary, directory, port); client = None
    try:
        client = WebSocket(port)
        client.call('system', 'register', username=USERNAME, password=PASSWORD)
        client.call('system', 'login', username=USERNAME, password=PASSWORD)
        player = client.call('system', 'create_character', characterName='内容验收侠', strength=16, constitution=16, agility=16, perception=16, willpower=16)['characterId']
        client.call('system', 'select_character', characterId=player)
        checks.append('register/login/create/select')
    finally:
        if client: client.close()
        stop(process)
    # Offline fixture injection is confined to this script's freshly created character.
    path = directory / 'saves/players' / (player + '.json')
    saved = json.loads(path.read_text(encoding='utf-8'))
    saved['items'].update(herb=20, dry_ration=10, bronze_fragment=5, iron_sword=1, smoke_powder=2)
    saved['attributes']['resources'].update(hp=40, mp=30)
    path.write_text(json.dumps(saved, ensure_ascii=False), encoding='utf-8')
    process = start(binary, directory, port); client = None
    try:
        client = connect(port, player)
        quests = client.call('command', 'get_quests')['quests']
        assert len(quests) == 38 and any(q['locked'] for q in quests)
        checks.append('38 unique quests with honest locked state')
        initial = client.call('command', 'get_attributes')['attributes']
        for _ in range(3):
            client.call('command', 'meditate'); client.call('command', 'stop')
        assert client.call('command', 'get_attributes')['attributes']['currentMp'] == initial['currentMp']
        checks.append('start/stop cannot farm immediate MP')
        client.call('command', 'meditate'); time.sleep(10.3); client.call('command', 'stop')
        assert client.call('command', 'get_attributes')['attributes']['currentMp'] == initial['currentMp'] + 15
        checks.append('meditation pays after natural 10 seconds')
        client.call('command', 'heal')
        assert client.call('command', 'get_attributes')['attributes']['currentHp'] == initial['currentHp']
        time.sleep(10.3); client.call('command', 'stop')
        assert client.call('command', 'get_attributes')['attributes']['currentHp'] == initial['currentHp'] + 20
        checks.append('healing pays after natural interval')
        walk(client, 'ly_south_gate', rooms)
        client.call('command', 'choose_dialogue', npcId='escort_bureau_chief', optionId='ask_work')
        walk(client, 'ly_inn', rooms)
        client.call('command', 'talk', npcId='storyteller'); client.call('command', 'talk', npcId='storyteller')
        quest = next(q for q in client.call('command', 'get_quests')['quests'] if q['id'] == 'escort_rumor_run')
        assert quest['canTurnIn'] and quest['progress'] == 100
        walk(client, 'ly_south_gate', rooms)
        client.call('command', 'choose_dialogue', npcId='escort_bureau_chief', optionId='turn_in_rumor')
        assert not raw_call(client, 'command', 'choose_dialogue', npcId='escort_bureau_chief', optionId='turn_in_rumor')['flag']
        checks.append('world quest actual clue/turn-in/duplicate prevention')
        walk(client, 'ly_refugee_camp', rooms)
        client.call('command', 'choose_dialogue', npcId='old_refugee', optionId='start_rumor_case')
        client.call('command', 'choose_dialogue', npcId='mute_refugee_woman', optionId='gather_rumor')
        client.call('command', 'choose_dialogue', npcId='old_refugee', optionId='turn_in_rumor')
        checks.append('independent mute-woman quest route')
        walk(client, 'ly_herbal_hall', rooms)
        before = client.call('command', 'get_backpack')['items']
        herb_before = next(i['count'] for i in before if i['id'] == 'herb')
        for _ in range(3): client.call('command', 'talk', npcId='herbalist_sun')
        after = client.call('command', 'get_backpack')['items']
        assert next(i['count'] for i in after if i['id'] == 'herb') == herb_before
        for _ in range(2): client.call('command', 'choose_dialogue', npcId='herbalist_sun', optionId='brew_with_herbs')
        after = client.call('command', 'get_backpack')['items']
        assert next(i['count'] for i in after if i['id'] == 'herb') == herb_before - 6
        checks.append('talk never spends; explicit recipes repeat and charge')
        assert not raw_call(client, 'command', 'enhance_item', itemId='training_spear')['flag']
        client.call('command', 'enhance_item', itemId='iron_sword')
        sword = next(i for i in client.call('command', 'get_backpack')['items'] if i['id'] == 'iron_sword')
        assert sword['enhancement']['level'] == 1 and sword['enhancement']['nextMoney'] == 400
        checks.append('enhancement ownership/cost/next-level preview')
        walk(client, 'ly_weapon_school', rooms)
        assert not raw_call(client, 'command', 'kill', targetId='weapon_school_master')['flag']
        checks.append('teacher protected from lethal attack')
        client.call('command', 'enter_world', worldId='luoyang')
        entry = client.call('system', 'get_room')['room']['id']
        client.call('command', 'enter_dungeon', dungeonId='abandoned_temple_rain')
        room = client.call('system', 'get_room')['room']
        target = next(n for n in room['npcs'] if n.get('hostile'))
        client.call('command', 'kill', targetId=target['id'])
        assert not raw_call(client, 'command', 'leave_dungeon')['flag']
        checks.append('combat cannot freely leave')
        before_hp = client.call('command', 'get_attributes')['attributes']['currentHp']
        result = client.call('command', 'use_item', itemId='smoke_powder')
        assert result['room']['id'] == entry
        assert client.call('command', 'get_attributes')['attributes']['currentHp'] <= before_hp
        locked = next(d for d in client.call('command', 'get_dungeons')['dungeons'] if d['id'] == 'datang_changshengjue')
        assert locked['locked']
        checks.append('smoke moves/partially settles without healing or unlocking')
    finally:
        if client: client.close()
        stop(process)
    saved = json.loads(path.read_text(encoding='utf-8'))
    saved['state'] = 'DEAD'; saved['attributes']['resources'].update(hp=0, mp=0, trainedMaxMp=4000)
    path.write_text(json.dumps(saved, ensure_ascii=False), encoding='utf-8')
    process = start(binary, directory, port); client = None
    try:
        client = connect(port, player)
        client.call('command', 'stop')
        me = next(p for p in client.call('system', 'get_room')['room']['playersInRoom'] if p['id'] == player)
        assert me['state'] == 'DEAD'
        assert not raw_call(client, 'command', 'leave_dungeon')['flag']
        checks.append('death survives stop and invalid exit')
        client.call('command', 'relive')
        a = client.call('command', 'get_attributes')['attributes']
        assert a['currentMp'] <= a['trainedMaxMp'] and a['resources']['trainedMaxMp'] == 4000 and a['weakRemainingSeconds'] > 0
        checks.append('revival preserves permanent MP, caps current MP, applies weakness')
    finally:
        if client: client.close()
        stop(process)
    return {'result': 'PASS', 'checks': checks, 'count': len(checks), 'directory': str(directory), 'player': player}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--keep-fixture', type=Path)
    parser.add_argument('--report', type=Path)
    args = parser.parse_args()
    if args.keep_fixture:
        directory = args.keep_fixture.resolve()
        if not directory.is_relative_to((ROOT / 'artifacts/content-review').resolve()) or directory.exists():
            raise ValueError('kept fixture must be a new directory inside artifacts/content-review')
        directory.mkdir(parents=True)
        context = contextlib.nullcontext(str(directory))
    else: context = tempfile.TemporaryDirectory(prefix='mud-content-acceptance-')
    with context as temporary:
        result = run(args.binary.resolve(), Path(temporary))
        if args.report:
            args.report.parent.mkdir(parents=True, exist_ok=True)
            args.report.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding='utf-8')
        print(json.dumps(result, ensure_ascii=False), flush=True)
