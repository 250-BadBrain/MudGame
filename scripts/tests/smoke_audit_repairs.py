"""Accept audit fixes over real WebSocket messages, with synthetic saves only."""
import argparse
import contextlib
import json
import socket
import tempfile
from pathlib import Path

from smoke_content_review import start, stop
from smoke_dungeon_quick import run as quick_run, connect
from smoke_martial_v2 import raw_call, public_data, walk

ROOT = Path(__file__).resolve().parents[2]


def inventory_state(backpack):
    return backpack['money'], sorted((item['id'], item['count'], (item.get('enhancement') or {}).get('level', 0)) for item in backpack['items'])


def run(binary, directory):
    quick = quick_run(binary, directory)
    player = quick['player']
    checks = list(quick['checks'])
    rooms, _ = public_data()
    with socket.socket() as probe:
        probe.bind(('127.0.0.1', 0)); port = probe.getsockname()[1]
    process, client = start(binary, directory, port), None
    try:
        client = connect(port, player)
        assert not raw_call(client, 'system', 'create_character', characterName='越界侠', strength=1000, constitution=1000, agility=1000, perception=1000, willpower=-3920)['flag']
        assert not raw_call(client, 'system', 'create_character', characterName='小数侠', strength=16.1, constitution=16.9, agility=16, perception=16, willpower=16)['flag']
        dead = client.call('system', 'create_character', characterName='江湖无名小卒', strength=16, constitution=16, agility=16, perception=16, willpower=16)['characterId']
        victim = client.call('system', 'create_character', characterName='离线验证者', strength=16, constitution=16, agility=16, perception=16, willpower=16)['characterId']
        checks.append('creation enforces integer ranges; six-character Chinese name accepted')
    finally:
        if client: client.close()
        stop(process)
    # Seed only these newly generated role files, with the backend stopped.
    path = directory / 'saves/players' / (player + '.json')
    saved = json.loads(path.read_text(encoding='utf-8'))
    before_money = saved['money']
    saved['items'].update(langfan_jian=2, langfan_j=1, smoke_powder=3)
    saved['enhancedItems'] = dict(langfan_jian=2, langfan_j=3)
    saved['attributes']['resources']['hp'] = 148
    path.write_text(json.dumps(saved, ensure_ascii=False), encoding='utf-8')
    dead_path = directory / 'saves/players' / (dead + '.json')
    saved_dead = json.loads(dead_path.read_text(encoding='utf-8'))
    saved_dead.update(roomId='dg_isolated_lost_instance', state='DEAD')
    saved_dead['attributes']['resources']['hp'] = 0
    dead_path.write_text(json.dumps(saved_dead, ensure_ascii=False), encoding='utf-8')
    process, client = start(binary, directory, port), None
    try:
        client = connect(port, player)
        backpack = client.call('command', 'get_backpack')
        assert backpack['money'] == before_money + 600
        gear = next(i for i in backpack['items'] if i['id'] == 'langfan_j')
        assert gear['count'] == 3 and gear['enhancement']['level'] == 3
        assert all(i['id'] != 'langfan_jian' for i in backpack['items'])
        assert next(i for i in backpack['items'] if i['id'] == 'bronze_fragment')['sellPrice'] == 200
        checks.append('legacy item aliases merge quantities, preserve max enhancement and refund duplicate investment')
        catalog = client.call('command', 'get_dungeons')['dungeons']
        assert len(catalog) == 78 and sum(d.get('contentStatus') == 'enriched' for d in catalog) == 40
        checks.append('all dungeons retained; 40 late chapters show their expanded interaction status')
        client.call('command', 'enter_world', worldId='bashu'); walk(client, 'bs_market', rooms)
        before = inventory_state(client.call('command', 'get_backpack'))
        for amount in [-1, 0, 10001, 1 << 60, 1.5, '2']:
            assert not raw_call(client, 'command', 'buy', npcId='bs_herbalist', itemId='herb', amount=amount)['flag']
        assert inventory_state(client.call('command', 'get_backpack')) == before
        client.call('command', 'buy', npcId='bs_herbalist', itemId='herb', amount=150)
        client.call('command', 'buy', npcId='bs_herbalist', itemId='dry_ration', amount=40)
        client.call('command', 'enter_world', worldId='luoyang'); walk(client, 'ly_herbal_hall', rooms)
        for _ in range(10): client.call('command', 'choose_dialogue', npcId='herbalist_sun', optionId='brew_herb_pill')
        assert not raw_call(client, 'command', 'sell', itemId='herb_pill', amount=10)['flag']
        walk(client, 'ly_east_market', rooms)
        before_sale = client.call('command', 'get_backpack')['money']
        client.call('command', 'sell', itemId='herb_pill', amount=10)
        assert client.call('command', 'get_backpack')['money'] == before_sale + 600
        checks.append('oversized/fractional trades rejected atomically; crafted pills sell at 20%, only at a live shop')
        client.call('command', 'enter_world', worldId='luoyang')
        assert not raw_call(client, 'command', 'kill', targetId=victim)['flag']
        assert not raw_call(client, 'command', 'spar', targetId=victim)['flag']
        client.call('system', 'select_character', characterId=victim)
        walk(client, 'ly_outskirts', rooms)
        client.call('system', 'select_character', characterId=player)
        walk(client, 'ly_outskirts', rooms)
        client.call('command', 'kill', targetId=victim)
        client.call('command', 'use_item', itemId='smoke_powder')
        checks.append('safe-zone player attacks blocked; an offline player outside safety can be attacked')
        client.call('system', 'select_character', characterId=dead)
        actor = next(p for p in client.call('system', 'get_room')['room']['playersInRoom'] if p['id'] == dead)
        assert actor['state'] == 'DEAD'
        client.call('command', 'stop')
        assert client.call('command', 'get_attributes')['attributes']['currentHp'] == 0
        client.call('command', 'relive')
        assert client.call('command', 'get_attributes')['attributes']['weakRemainingSeconds'] > 0
        checks.append('lost dungeon instance does not revive a dead player; actual revival applies weakness')
        balance = client.call('command', 'get_backpack')['money']
    finally:
        if client: client.close()
        stop(process)
    process, client = start(binary, directory, port), None
    try:
        client = connect(port, dead)
        assert client.call('command', 'get_backpack')['money'] == balance
        client.call('system', 'select_character', characterId=player)
        gear = next(i for i in client.call('command', 'get_backpack')['items'] if i['id'] == 'langfan_j')
        assert gear['count'] == 3 and gear['enhancement']['level'] == 3
        persisted = json.loads(path.read_text(encoding='utf-8'))
        assert 'langfan_jian' not in persisted['items'] and 'langfan_jian' not in persisted['enhancedItems']
        checks.append('normalized item ownership and refund removal survive restart')
    finally:
        if client: client.close()
        stop(process)
    return dict(status='PASS', checks=checks, player=player, fixture=str(directory))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--report', type=Path, required=True)
    parser.add_argument('--keep-fixture', type=Path)
    args = parser.parse_args()
    if args.keep_fixture:
        directory = args.keep_fixture.resolve()
        assert directory.is_relative_to((ROOT / 'artifacts/quick-complete').resolve()) and not directory.exists()
        directory.mkdir(parents=True)
        context = contextlib.nullcontext(str(directory))
    else: context = tempfile.TemporaryDirectory(prefix='mud-audit-repairs-')
    with context as temporary:
        result = run(args.binary.resolve(), Path(temporary))
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding='utf-8')
        print(json.dumps(result, ensure_ascii=False))
