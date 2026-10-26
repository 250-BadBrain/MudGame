"""Run a built backend with temporary saves and verify titles over real WebSocket messages."""

import argparse
import base64
import json
import os
import signal
import socket
import struct
import subprocess
import tempfile
import time
import urllib.request
from pathlib import Path


class WebSocket:
    def __init__(self, port):
        self.socket = socket.create_connection(("127.0.0.1", port), timeout=8)
        self.stream = self.socket.makefile("rb")
        key = base64.b64encode(os.urandom(16)).decode()
        self.socket.sendall((f"GET /ws HTTP/1.1\r\nHost: 127.0.0.1:{port}\r\nUpgrade: websocket\r\n"
                             f"Connection: Upgrade\r\nSec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n").encode())
        assert b"101" in self.stream.readline(), "WebSocket upgrade failed"
        while True:
            line = self.stream.readline()
            assert line, "WebSocket handshake closed unexpectedly"
            if line == b"\r\n":
                break

    def close(self):
        self.stream.close()
        self.socket.close()

    def send(self, payload, opcode=1):
        mask = os.urandom(4)
        length = len(payload)
        header = bytes([0x80 | opcode])
        if length < 126:
            header += bytes([0x80 | length])
        elif length < 65536:
            header += bytes([0xFE]) + struct.pack("!H", length)
        else:
            header += bytes([0xFF]) + struct.pack("!Q", length)
        masked = bytes(value ^ mask[index % 4] for index, value in enumerate(payload))
        self.socket.sendall(header + mask + masked)

    def read(self, length):
        value = self.stream.read(length)
        assert len(value) == length, "WebSocket closed unexpectedly"
        return value

    def receive(self):
        message = bytearray()
        while True:
            first, second = self.read(2)
            length = second & 0x7F
            if length == 126:
                length = struct.unpack("!H", self.read(2))[0]
            elif length == 127:
                length = struct.unpack("!Q", self.read(8))[0]
            assert not second & 0x80, "server frames must be unmasked"
            payload = self.read(length)
            opcode = first & 0x0F
            assert opcode != 8, "WebSocket close frame"
            if opcode == 9:
                self.send(payload, 10)
                continue
            if opcode == 10:
                continue
            assert opcode in (0, 1), "unexpected frame type"
            message.extend(payload)
            if first & 0x80:
                return json.loads(message)

    def call(self, kind, subtype, **args):
        self.send(json.dumps({"type": kind, "subtype": subtype, "args": args}).encode())
        while True:
            result = self.receive()
            if result.get("type") == "event":
                continue
            assert result.get("type") == kind and result.get("subtype") == subtype, result
            assert result.get("flag") is True, result
            return result.get("results", {})


def start(binary, resources, data, cwd, port):
    environment = dict(os.environ, PORT=str(port), MUD_DATA_DIR=str(data), MUD_RESOURCES_DIR=str(resources))
    process = subprocess.Popen([str(binary)], cwd=cwd, env=environment, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        for _ in range(100):
            if process.poll() is not None:
                raise AssertionError("backend failed to start: " + (data / "server.log").read_text(encoding="utf-8"))
            try:
                with urllib.request.urlopen(f"http://127.0.0.1:{port}/health", timeout=0.2) as response:
                    assert json.load(response)["ok"] is True
                return process
            except OSError:
                time.sleep(0.05)
        raise AssertionError("backend startup timed out")
    except BaseException:
        process.kill()
        process.wait()
        raise


def stop(process):
    process.send_signal(signal.SIGINT)
    try:
        assert process.wait(timeout=10) == 0, "backend shutdown failed"
    finally:
        if process.poll() is None:
            process.kill()
            process.wait()


def smoke(binary, resources):
    with tempfile.TemporaryDirectory(prefix="mudgame-smoke-") as directory:
        cwd = Path(directory)
        data = cwd / "saves"
        with socket.socket() as probe:
            probe.bind(("127.0.0.1", 0))
            port = probe.getsockname()[1]
        process = start(binary, resources, data, cwd, port)
        client = None
        try:
            client = WebSocket(port)
            client.call("system", "register", username="smoke", password="temporary-smoke-password")
            client.call("system", "login", username="smoke", password="temporary-smoke-password")
            player = client.call("system", "create_character", characterName="冒烟侠客", strength=16,
                                 constitution=16, agility=16, perception=16, willpower=16)["characterId"]
            client.call("system", "select_character", characterId=player)
            assert client.call("system", "get_room")["room"]["id"] == "ly_00"
            assert not client.call("command", "get_attributes")["attributes"].get("titles")
        finally:
            if client:
                client.close()
            stop(process)

        # Modify only this script's synthetic save, with the backend stopped.
        path = data / "players" / (player + ".json")
        saved = json.loads(path.read_text(encoding="utf-8"))
        saved.setdefault("flags", {})["dungeon_cleared_abandoned_temple_rain"] = True
        path.write_text(json.dumps(saved, ensure_ascii=False), encoding="utf-8")
        process = start(binary, resources, data, cwd, port)
        client = None
        try:
            client = WebSocket(port)
            client.call("system", "login", username="smoke", password="temporary-smoke-password")
            client.call("system", "select_character", characterId=player)
            attributes = client.call("command", "get_attributes")["attributes"]
            assert "荒庙义士" in attributes["titles"], attributes
            logs = (data / "server.log").read_text(encoding="utf-8")
            assert "[titles] loaded 33 titles" in logs and "titles disabled" not in logs, logs
            print("PASS: register/login/create/select/room/attributes/save reload/runtime titles; temporary data only.", flush=True)
        finally:
            if client:
                client.close()
            stop(process)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--resources", type=Path, required=True)
    args = parser.parse_args()
    smoke(args.binary.resolve(), args.resources.resolve())
