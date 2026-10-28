"""Real WebSocket acceptance using only freshly created synthetic player saves."""
import argparse
import json
import socket
import tempfile
from pathlib import Path

from smoke_backend import WebSocket
from smoke_content_review import start, stop
from smoke_martial_v2 import raw_call

ROOT = Path(__file__).resolve().parents[2]
USER, PASSWORD = 'quick_review_ui', 'temporary-quick-pass'


def connect(port, player):
    client = WebSocket(port)
    client.call('system', 'login', username=USER, password=PASSWORD)
    client.call('system', 'select_character', characterId=player)
    return client


def run(binary, directory):
    checks = []
    with socket.socket() as probe:
        probe.bind(('127.0.0.1', 0))
        port = probe.getsockname()[1]
    assert port != 4009
    process, client = start(binary, directory, port), None
    try:
        client = WebSocket(port)
        client.call('system', 'register', username=USER, password=PASSWORD)
        client.call('system', 'login', username=USER, password=PASSWORD)
        player = client.call('system', 'create_character', characterName='快速验收侠', strength=16, constitution=16, agility=16, perception=16, willpower=16)['characterId']
        client.call('system', 'select_character', characterId=player)
        assert not raw_call(client, 'command', 'quick_complete_dungeon', dungeonId='datang_changshengjue', expectedSequence=0)['flag']
        assert not next(d for d in client.call('command', 'get_dungeons')['dungeons'] if d['id'] == 'datang_changshengjue')['canQuickComplete']
        checks.append('uncleared rejected; directory button disabled')
    finally:
        if client: client.close()
        stop(process)
    path = directory / 'saves/players' / (player + '.json')
    saved = json.loads(path.read_text(encoding='utf-8'))
    saved['flags']['dungeon_cleared_datang_changshengjue'] = True
    saved['flags']['dungeon_cleared_datang_changan'] = True
    saved['attributes']['resources'].update(hp=30, mp=20)
    path.write_text(json.dumps(saved, ensure_ascii=False), encoding='utf-8')
    process, client = start(binary, directory, port), None
    try:
        client = connect(port, player)
        dungeon = next(d for d in client.call('command', 'get_dungeons')['dungeons'] if d['id'] == 'datang_changshengjue')
        assert dungeon['canQuickComplete'] and dungeon['quickCompleteSequence'] == 0
        before = client.call('command', 'get_attributes')['attributes']
        result = client.call('command', 'quick_complete_dungeon', dungeonId=dungeon['id'], expectedSequence=0)['settlement']
        assert result['progress'] == 100 and result['rewardItems']['herb'] == 4 and result['rewardItems']['dry_ration'] >= 2
        assert result['rewardItems']['bronze_fragment'] == 2 and result['rewardMoney'] == 80
        assert result['rewardExperience'] == 260 and result['rewardPotential'] == 180
        after = client.call('command', 'get_attributes')['attributes']
        assert after['currentHp'] == before['currentHp'] and after['currentMp'] == before['currentMp']
        assert not raw_call(client, 'command', 'quick_complete_dungeon', dungeonId=dungeon['id'], expectedSequence=0)['flag']
        assert client.call('command', 'get_attributes')['attributes'] == after
        checks.append('full rewards, fixed process loot, bronze, no recovery and replay rejected')
        second = client.call('command', 'quick_complete_dungeon', dungeonId='datang_changan', expectedSequence=1)['settlement']
        assert second['rewardItems']['bronze_fragment'] == 2 and second['rewardItems']['xiaohuandan'] == 2
        assert second['rewardMoney'] == 820  # full 420 + process 400
        checks.append('empty normal item table still grants bronze and fixed process items/money')
        client.call('command', 'meditate')
        assert not raw_call(client, 'command', 'quick_complete_dungeon', dungeonId=dungeon['id'], expectedSequence=2)['flag']
        client.call('command', 'stop')
        assert all(d['quickCompleteSequence'] == 2 for d in client.call('command', 'get_dungeons')['dungeons'])
        checks.append('busy rejected without consuming sequence')
    finally:
        if client: client.close()
        stop(process)
    process, client = start(binary, directory, port), None
    try:
        client = connect(port, player)
        assert all(d['quickCompleteSequence'] == 2 for d in client.call('command', 'get_dungeons')['dungeons'])
        assert not raw_call(client, 'command', 'quick_complete_dungeon', dungeonId='datang_changan', expectedSequence=1)['flag']
        client.call('command', 'quick_complete_dungeon', dungeonId='datang_changan', expectedSequence=2)
        checks.append('sequence persists across restart; old replay rejected and next click succeeds')
    finally:
        if client: client.close()
        stop(process)
    return dict(status='PASS', checks=checks, player=player, fixture=str(directory))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', required=True, type=Path)
    parser.add_argument('--report', type=Path)
    parser.add_argument('--fixture-dir', type=Path, help='New synthetic fixture folder strictly inside artifacts/quick-complete')
    args = parser.parse_args()
    if args.fixture_dir:
        fixture = args.fixture_dir.resolve()
        allowed = (ROOT / 'artifacts/quick-complete').resolve()
        assert fixture.is_relative_to(allowed) and fixture != allowed and not fixture.exists()
        fixture.mkdir(parents=True)
        report = run(args.binary.resolve(), fixture)
    else:
        with tempfile.TemporaryDirectory(prefix='mud-quick-acceptance-') as temporary:
            report = run(args.binary.resolve(), Path(temporary))
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding='utf-8')
    print(json.dumps(report, ensure_ascii=False))


if __name__ == '__main__':
    main()
