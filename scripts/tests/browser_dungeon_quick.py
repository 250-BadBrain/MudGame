"""Run headless UI acceptance with an existing synthetic quick-complete fixture."""
import argparse
import os
from pathlib import Path
import socket
import subprocess
import time
import urllib.request

from smoke_content_review import start, stop

ROOT = Path(__file__).resolve().parents[2]


def free_port():
    with socket.socket() as probe:
        probe.bind(('127.0.0.1', 0))
        return probe.getsockname()[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--fixture-dir', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--scenario', choices=['quick', 'enhancement', 'content-depth'], default='quick')
    args = parser.parse_args()
    fixture, output = args.fixture_dir.resolve(), args.output.resolve()
    allowed = (ROOT / 'artifacts/quick-complete').resolve()
    assert fixture.is_relative_to(allowed) and (fixture / 'saves/users/quick_review_ui.json').exists()
    assert output.is_relative_to(allowed)
    output.mkdir(parents=True, exist_ok=True)
    backend_port, ui_port = free_port(), free_port()
    assert backend_port != 4009 and ui_port != backend_port
    backend = start(args.binary.resolve(), fixture, backend_port)
    ui = None
    try:
        env = dict(os.environ, VITE_WS_URL=f'ws://127.0.0.1:{backend_port}/ws', MUD_TEST_UI_URL=f'http://127.0.0.1:{ui_port}', MUD_TEST_UI_OUTPUT=str(output))
        flags = subprocess.CREATE_NEW_PROCESS_GROUP if os.name == 'nt' else 0
        with (output / 'vite.log').open('wb') as log:
            ui = subprocess.Popen(['node', str(ROOT / 'MudGame/node_modules/vite/bin/vite.js'), '--host', '127.0.0.1', '--port', str(ui_port), '--strictPort'], cwd=ROOT / 'MudGame', env=env, stdout=log, stderr=log, creationflags=flags)
        for _ in range(100):
            if ui.poll() is not None: raise AssertionError((output / 'vite.log').read_text(encoding='utf-8'))
            try:
                urllib.request.urlopen(env['MUD_TEST_UI_URL'], timeout=.2).close()
                break
            except OSError: time.sleep(.1)
        else: raise AssertionError('Vite did not start')
        script = {'quick':'browser_dungeon_quick.cjs', 'enhancement':'browser_enhancement.cjs', 'content-depth':'browser_content_depth.cjs'}[args.scenario]
        subprocess.run(['node', str(ROOT / 'scripts/tests' / script)], cwd=ROOT, env=env, check=True, timeout=90)
    finally:
        if ui and ui.poll() is None:
            ui.terminate()
            ui.wait(timeout=10)
        stop(backend)


if __name__ == '__main__':
    main()
