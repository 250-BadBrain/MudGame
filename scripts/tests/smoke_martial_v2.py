"""Real protocol smoke against an already running, isolated local V2 server."""
import argparse
import collections
import json
import time
import uuid
from pathlib import Path

from smoke_backend import WebSocket

ROOT = Path(__file__).resolve().parents[2]
RESOURCES = ROOT / 'MudGame-server/resources'


def raw_call(client, kind, subtype, **args):
    client.send(json.dumps({'type': kind, 'subtype': subtype, 'args': args}).encode())
    while True:
        result = client.receive()
        if result.get('type') == 'event': continue
        assert result.get('type') == kind and result.get('subtype') == subtype, result
        return result


def public_data():
    rooms = {}
    for path in (RESOURCES / 'worlds').glob('*.json'):
        rooms.update(json.loads(path.read_text(encoding='utf-8'))['rooms'])
    npcs = json.loads((RESOURCES / 'npcs.json').read_text(encoding='utf-8'))
    return rooms, npcs


def walk(client, target, rooms):
    current = client.call('system', 'get_room')['room']['id']
    pending = collections.deque([(current, [])]); seen = {current}
    while pending:
        room, path = pending.popleft()
        if room == target:
            for direction in path: client.call('command', 'go', direction=direction)
            assert client.call('system', 'get_room')['room']['id'] == target
            return
        for direction, destination in rooms[room].get('exits', {}).items():
            if not isinstance(destination, str): destination = destination['targetRoomId']
            if destination not in seen:
                seen.add(destination); pending.append((destination, path + [direction]))
    raise AssertionError('No public route to ' + target)


def smoke(port):
    assert port != 4009, 'Do not point this test at the default production port'
    rooms, npcs = public_data()
    suffix = uuid.uuid4().hex[:6]; username = 'v2smoke' + suffix
    password = 'temporary-smoke-pass'
    client = WebSocket(port)
    player = None
    try:
        client.call('system', 'register', username=username, password=password)
        client.call('system', 'login', username=username, password=password)
        player = client.call('system', 'create_character', characterName='验收'+suffix,
                             strength=16, constitution=16, agility=16, perception=16, willpower=16)['characterId']
        client.call('system', 'select_character', characterId=player)
        catalog = client.call('command', 'get_skills')['skills']
        assert len(catalog) == 38
        changsheng = next(skill for skill in catalog if skill['id'] == 'csj_changshengjue')
        assert changsheng['rarity'] == 5 and '47' in changsheng['source']
        client.call('command', 'practice', skillId='basic_unarmed')
        time.sleep(20.3)
        client.call('command', 'stop')
        catalog = client.call('command', 'get_skills')['skills']
        assert next(skill for skill in catalog if skill['id'] == 'basic_unarmed')['level'] == 20
        assert client.call('command', 'get_attributes')['attributes']['potential'] == 481
        walk(client, 'ly_weapon_school', rooms)
        client.call('command', 'learn', npcId='weapon_school_master', skillId='fuhu_jin')
        client.call('command', 'practice', skillId='fuhu_jin')
        time.sleep(10.3)
        client.call('command', 'stop')
        client.call('command', 'equip_skill', skillId='fuhu_jin')
        art = next(skill for skill in client.call('command', 'get_skills')['skills'] if skill['id'] == 'fuhu_jin')
        assert art['level'] >= 11 and art['equipped'] and art['active'], art
        for item in ['simple_medicine', 'dry_ration']:
            npc = next(nid for nid, n in npcs.items() if item in n.get('shopItems', []) and any(nid in room.get('npcs', []) for room in rooms.values()))
            room = next(rid for rid, r in rooms.items() if npc in r.get('npcs', []))
            walk(client, room, rooms)
            client.call('command', 'buy', npcId=npc, itemId=item, amount=2)
        client.call('command', 'use_item', itemId='simple_medicine')
        ready = client.call('command', 'get_attributes')['attributes']['recoveryItemReadyAt']
        client.call('command', 'unequip_skill', slot='combat')
        client.call('command', 'equip_skill', skillId='fuhu_jin')
        failed = raw_call(client, 'command', 'use_item', itemId='dry_ration')
        assert failed['flag'] is False and '冷却' in failed['results']['error'], failed
        # Disconnect pauses practice without discarding progress or clearing recovery cooldown.
        client.call('command', 'practice', skillId='fuhu_jin')
        client.close(); client = None
        time.sleep(.25)
        client = WebSocket(port)
        client.call('system', 'login', username=username, password=password)
        client.call('system', 'select_character', characterId=player)
        room = client.call('system', 'get_room')['room']
        assert next(p for p in room['playersInRoom'] if p['id'] == player)['state'] == 'IDLE'
        assert client.call('command', 'get_attributes')['attributes']['recoveryItemReadyAt'] == ready
        backpack = client.call('command', 'get_backpack')['items']
        assert next(item for item in backpack if item['id'] == 'dry_ration')['count'] == 2
        # A genuine chapter encounter exercises the same combat/room stream as the UI.
        entry_room = client.call('system', 'get_room')['room']['id']
        client.call('command', 'enter_dungeon', dungeonId='abandoned_temple_rain')
        room = client.call('system', 'get_room')['room']
        target = next(n for n in room['npcs'] if n['name'] == npcs['temple_bandit']['name'])
        client.call('command', 'kill', targetId=target['id'])
        for _ in range(25):
            time.sleep(1)
            room = client.call('system', 'get_room')['room']
            me = next(p for p in room['playersInRoom'] if p['id'] == player)
            if me['state'] != 'COMBAT':
                assert me['state'] != 'DEAD'
                break
        else: raise AssertionError('first chapter encounter timed out')
        client.call('command', 'leave_dungeon')
        assert client.call('system', 'get_room')['room']['id'] == entry_room
        print(json.dumps({'result': 'PASS', 'player': player, 'checks': ['38-entry catalog', 'natural budget training',
                          'teacher entry', 'self practice', 'active combat art', 'unequip/equip', 'shared recovery cooldown',
                          'failure preserves food', 'disconnect pauses practice', 'reconnect preserves cooldown', 'enter chapter', 'real NPC combat', 'partial settlement exit']}, ensure_ascii=False))
    finally:
        if client: client.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--port', type=int, required=True)
    smoke(parser.parse_args().port)
