"""Isolated real-protocol acceptance for unlimited smoke and legacy enhancement reset."""
import argparse
import contextlib
import json
import math
import socket
import tempfile
from pathlib import Path

from smoke_backend import WebSocket
from smoke_content_review import start, stop
from smoke_dungeon_quick import connect, USER, PASSWORD
from smoke_martial_v2 import raw_call

ROOT = Path(__file__).resolve().parents[2]


def gear(client, item):
    return next(i for i in client.call('command', 'get_backpack')['items'] if i['id'] == item)


def run(binary, directory):
    checks = []
    with socket.socket() as probe:
        probe.bind(('127.0.0.1', 0)); port = probe.getsockname()[1]
    assert port != 4009
    process, client = start(binary, directory, port), None
    try:
        client = WebSocket(port)
        client.call('system', 'register', username=USER, password=PASSWORD)
        client.call('system', 'login', username=USER, password=PASSWORD)
        roles = [client.call('system', 'create_character', characterName=name, strength=16, constitution=16, agility=16, perception=16, willpower=16)['characterId'] for name in ['强化验收侠', '新强化侠']]
    finally:
        if client: client.close()
        stop(process)
    legacy, fresh = roles
    # Edit only brand-new synthetic roles after clean shutdown.
    for role in roles:
        path = directory / 'saves/players' / (role + '.json')
        p = json.loads(path.read_text(encoding='utf-8'))
        p['roomId'] = 'ly_west_market'
        if role == legacy:
            p['money'] = 100000
            p['items'].update(iron_sword=1, tangmen_baojian=1, cangming_juan=1, langfan_jian=1, langfan_j=1, bronze_fragment=20)
            p['enhancedItems'] = dict(iron_sword=2, tangmen_baojian=3, cangming_juan=5, langfan_jian=2, langfan_j=3)
            p['flags'].pop('enhancement_coefficients_v2', None)
        path.write_text(json.dumps(p, ensure_ascii=False), encoding='utf-8')
    process, client = start(binary, directory, port), None
    try:
        client = connect(port, legacy)
        assert gear(client, 'tangmen_baojian')['enhancement']['canReset']
        assert client.call('command', 'get_backpack')['money'] == 100600
        checks.append('legacy levels preserved; alias duplicate refund precedes reset eligibility')
        for role in roles:
            client.call('system', 'select_character', characterId=role)
            before = client.call('command', 'get_backpack')['money']
            client.call('command', 'buy', npcId='black_market_broker', itemId='smoke_powder', amount=6)
            assert client.call('command', 'get_backpack')['money'] == before - 1500
            assert 'limit' not in next(i for i in client.call('command', 'get_shop', npcId='black_market_broker')['shopItems'] if i['id'] == 'smoke_powder')
        checks.append('two characters each buy more than old global stock; pay 250 per pack; supply stays unlimited')
        assert not raw_call(client, 'command', 'reset_enhancement', itemId='iron_sword')['flag']
        client.call('system', 'select_character', characterId=legacy)
        before = client.call('command', 'get_attributes')['attributes']
        client.call('command', 'equip', itemId='tangmen_baojian')
        attrs = client.call('command', 'get_attributes')['attributes']
        assert attrs['attack'] == before['attack'] + 6
        assert math.isclose(attrs['critRate'], before['critRate'] + 5.3)
        equipment = client.call('command', 'get_equipment')['equipment']['RING']
        assert equipment['enhancement']['nextBonus']['critRate'] == .1
        assert equipment['enhancement']['nextBonus']['attack'] == 1
        client.call('command', 'enhance_item', itemId='tangmen_baojian')
        improved = client.call('command', 'get_attributes')['attributes']
        assert improved['attack'] == attrs['attack'] + 1 and math.isclose(improved['critRate'], attrs['critRate'] + .1)
        checks.append('equipped multi-affix gear uses numeric coefficients; actual fractional crit matches preview')
        balance = client.call('command', 'get_backpack')['money']
        client.call('command', 'reset_enhancement', itemId='tangmen_baojian')
        assert client.call('command', 'get_backpack')['money'] == balance + 2000
        reset = client.call('command', 'get_attributes')['attributes']
        assert reset['attack'] == before['attack'] + 3 and reset['critRate'] == before['critRate'] + 5
        assert reset['currentHp'] == improved['currentHp']
        assert not raw_call(client, 'command', 'reset_enhancement', itemId='tangmen_baojian')['flag']
        client.call('command', 'enhance_item', itemId='tangmen_baojian')
        client.call('command', 'unequip', slot='RING')
        checks.append('reset refunds entire current investment, removes gain without healing; replay rejected')
        client.call('command', 'equip', itemId='cangming_juan')
        capped = client.call('command', 'get_attributes')['attributes']
        client.call('command', 'reset_enhancement', itemId='cangming_juan')
        lower = client.call('command', 'get_attributes')['attributes']
        assert capped['maxHp'] - lower['maxHp'] == 25 and capped['maxMp'] - lower['maxMp'] == 25
        assert capped['currentHp'] == lower['currentHp'] and capped['trainedMaxMp'] == lower['trainedMaxMp']
        client.call('command', 'unequip', slot='NECKLACE')
        checks.append('HP/MP effects gain 5 per level; reset preserves current resources and permanent training')
        balance = client.call('command', 'get_backpack')['money']
        client.call('command', 'reset_enhancement', itemId='langfan_jian')
        assert client.call('command', 'get_backpack')['money'] == balance + 1200
        assert gear(client, 'langfan_j')['count'] == 2
        checks.append('old alias command resolves canonically; only retained maximum-level investment refunded')
        balance = client.call('command', 'get_backpack')['money']
    finally:
        if client: client.close()
        stop(process)
    process, client = start(binary, directory, port), None
    try:
        client = connect(port, legacy)
        assert client.call('command', 'get_backpack')['money'] == balance
        assert not gear(client, 'tangmen_baojian')['enhancement']['canReset']
        assert gear(client, 'tangmen_baojian')['enhancement']['level'] == 1
        assert gear(client, 'iron_sword')['enhancement']['canReset']
        assert not raw_call(client, 'command', 'reset_enhancement', itemId='tangmen_baojian')['flag']
        checks.append('restart preserves used/unused eligibility; re-enhancing cannot renew a consumed reset')
    finally:
        if client: client.close()
        stop(process)
    return dict(status='PASS', checks=checks, player=legacy)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--report', type=Path, required=True)
    parser.add_argument('--keep-fixture', type=Path)
    args = parser.parse_args()
    if args.keep_fixture:
        directory = args.keep_fixture.resolve()
        allowed = (ROOT / 'artifacts/quick-complete').resolve()
        assert directory.is_relative_to(allowed) and directory != allowed and not directory.exists()
        directory.mkdir(parents=True)
        context = contextlib.nullcontext(str(directory))
    else:
        context = tempfile.TemporaryDirectory(prefix='mud-enhancement-')
    with context as temporary:
        report = run(args.binary.resolve(), Path(temporary))
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding='utf-8')
        print(json.dumps(report, ensure_ascii=False))
