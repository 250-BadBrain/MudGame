"""Validate and activate one backend release; uses only Python's standard library."""

import argparse
import contextlib
import hashlib
import json
import os
import platform
import re
import shutil
import subprocess
import tarfile
import time
import urllib.request
from dataclasses import dataclass
from pathlib import Path, PurePosixPath


class DeployError(RuntimeError):
    pass


@dataclass
class Config:
    root: Path
    release_id: str
    service: str
    executable: str
    port: int
    archive_sha256: str

    @property
    def server(self):
        return self.root / "MudGame-server"

    @property
    def workspace(self):
        return self.root / ".deploy"

    @property
    def package(self):
        return self.workspace / "incoming" / (self.release_id + ".tar.gz")

    @property
    def staging(self):
        return self.workspace / "staging" / self.release_id

    @property
    def backup(self):
        return self.workspace / "backups" / self.release_id

    @property
    def failed(self):
        return self.workspace / "failed" / self.release_id

    def validate(self):
        if not self.root.is_absolute() or self.root.resolve() == self.root.resolve().parent:
            raise DeployError("root must be an absolute project directory")
        for value in (self.release_id, self.service, self.executable):
            if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.@-]*", value):
                raise DeployError("invalid release, service or executable name")
        if not 1 <= self.port <= 65535:
            raise DeployError("invalid health port")
        if self.server.is_symlink() or self.workspace.is_symlink():
            raise DeployError("server and deployment workspace must not be symlinks")


def run(args):
    result = subprocess.run(args, text=True, capture_output=True, timeout=90)
    if result.returncode:
        raise DeployError(f"command failed: {' '.join(args)}: {result.stderr.strip()}")
    return result.stdout.strip()


class SystemdService:
    def __init__(self, config):
        self.config = config

    def property(self, name):
        return run(["systemctl", "show", self.config.service, "--property=" + name, "--value"])

    def is_active(self):
        return self.property("ActiveState") == "active"

    def preflight(self, config):
        if platform.machine() not in ("x86_64", "amd64"):
            raise DeployError("this package requires Linux amd64")
        if not self.is_active():
            raise DeployError("expected an existing active systemd service")
        if Path(self.property("WorkingDirectory")).resolve() != config.server.resolve():
            raise DeployError("systemd WorkingDirectory does not match deployment directory")
        match = re.search(r"\bpath=([^ ;]+)", self.property("ExecStart"))
        if not match or Path(match.group(1)).resolve() != (config.server / config.executable).resolve():
            raise DeployError("systemd ExecStart does not match deployment executable")
        pid = int(self.property("MainPID"))
        # Only use the three deployment settings; never print the process environment.
        settings = {}
        for entry in (Path("/proc") / str(pid) / "environ").read_bytes().split(b"\0"):
            key, _, value = entry.partition(b"=")
            if key in (b"MUD_DATA_DIR", b"MUD_RESOURCES_DIR", b"PORT"):
                settings[key.decode()] = value.decode()

        def resolve(value):
            path = Path(value)
            return (path if path.is_absolute() else config.server / path).resolve()

        if resolve(settings.get("MUD_RESOURCES_DIR") or "resources") != (config.server / "resources").resolve():
            raise DeployError("service uses resources outside the deployment directory")
        if int(settings.get("PORT") or "4009") != config.port:
            raise DeployError("health port does not match service PORT")
        data = resolve(settings.get("MUD_DATA_DIR") or "../data/servers/server1")
        if data.is_relative_to(config.server.resolve()) or data.is_relative_to(config.workspace.resolve()):
            raise DeployError("save directory must be outside the program and deployment directories")
        run(["sudo", "-n", "true"])
        return data

    def stop(self):
        run(["sudo", "-n", "systemctl", "stop", self.config.service])
        if self.is_active() or self.property("MainPID") != "0":
            raise DeployError("service has not stopped; refusing to replace program")

    def start(self):
        run(["sudo", "-n", "systemctl", "start", self.config.service])

    def verify_running_binary(self):
        pid = int(self.property("MainPID"))
        if pid <= 0 or not os.path.samefile(Path("/proc") / str(pid) / "exe", self.config.server / self.config.executable):
            raise DeployError("systemd is not running the installed executable")


def check_health(config):
    deadline = time.monotonic() + 20
    error = "service did not become healthy"
    while time.monotonic() < deadline:
        try:
            with urllib.request.urlopen(f"http://127.0.0.1:{config.port}/health", timeout=2) as response:
                if json.load(response).get("ok") is True:
                    return
        except (OSError, ValueError) as exc:
            error = str(exc)
        time.sleep(0.5)
    raise DeployError("health check failed: " + error)


@contextlib.contextmanager
def deployment_lock(config):
    config.workspace.mkdir(parents=True, exist_ok=True)
    lock = config.workspace / "deploy.lock"
    try:
        stream = lock.open("x")
    except FileExistsError as exc:
        raise DeployError(f"deployment lock exists: {lock}; verify no deployment is running before removing it") from exc
    try:
        with stream:
            json.dump({"pid": os.getpid(), "release": config.release_id}, stream)
        yield
    finally:
        lock.unlink()


def stage(config):
    if hashlib.sha256(config.package.read_bytes()).hexdigest() != config.archive_sha256.lower():
        raise DeployError("archive checksum mismatch")
    if config.staging.exists() or config.backup.exists() or config.failed.exists():
        raise DeployError("release ID has already been used")
    with tarfile.open(config.package, "r:gz") as archive:
        files = {}
        for entry in archive.getmembers():
            path = PurePosixPath(entry.name)
            if path.is_absolute() or ".." in path.parts or "\\" in entry.name or not (entry.isfile() or entry.isdir()):
                raise DeployError("unsafe archive entry: " + entry.name)
            name = str(path)
            if entry.isfile():
                if name in files:
                    raise DeployError("duplicate archive file: " + name)
                files[name] = entry
        if "release.json" not in files:
            raise DeployError("missing release manifest")
        manifest = json.load(archive.extractfile(files["release.json"]))
        if manifest.get("releaseId") != config.release_id or manifest.get("target") != "linux/amd64":
            raise DeployError("release manifest does not match deployment")
        expected = manifest.get("files", {})
        if not expected or set(files) != set(expected) | {"release.json"}:
            raise DeployError("archive contents do not match manifest")
        for name in expected:
            if name != config.executable and not name.startswith("resources/"):
                raise DeployError("package may contain only the executable and resources")
        required = {config.executable} | {"resources/" + name + ".json" for name in ("items", "skills", "npcs", "titles")}
        if not required.issubset(expected) or not any(name.startswith("resources/worlds/") for name in expected):
            raise DeployError("release is missing required resources")
        config.staging.mkdir(parents=True)
        for name, entry in files.items():
            value = archive.extractfile(entry).read()
            if name != "release.json" and hashlib.sha256(value).hexdigest() != expected[name].lower():
                raise DeployError("file checksum mismatch: " + name)
            if name.endswith(".json"):
                json.loads(value)
            target = config.staging / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(value)
    binary = config.staging / config.executable
    header = binary.read_bytes()[:20]
    if len(header) < 20 or header[:6] != b"\x7fELF\x02\x01" or int.from_bytes(header[18:20], "little") != 62:
        raise DeployError("executable is not Linux amd64 ELF")
    binary.chmod(0o755)


def deploy(config, service=None, health=check_health, stage_only=False):
    config.validate()
    service = service or SystemdService(config)
    with deployment_lock(config):
        stage(config)
        data = service.preflight(config)
        if config.backup.resolve().is_relative_to(data.resolve()):
            raise DeployError("save snapshot destination is inside the save directory")
        if not config.server.is_dir() or not (config.server / config.executable).is_file():
            raise DeployError("existing backend executable is missing")
        if stage_only:
            shutil.rmtree(config.staging)
            config.package.unlink()
            print("Release verified; production was not stopped. Validation package cleaned up.", flush=True)
            return

        old_server = config.backup / "MudGame-server"
        stopped = False
        installed = False
        try:
            print("Stopping service and backing up flushed saves...", flush=True)
            # Set before calling stop: even a failed stop can leave an inactive service.
            stopped = True
            service.stop()
            config.backup.mkdir(parents=True)
            if data.exists():
                shutil.copytree(data, config.backup / "data-snapshot", symlinks=True)
            (config.backup / "backup.json").write_text(
                json.dumps({"dataDirectory": str(data), "release": config.release_id}), encoding="utf-8"
            )
            config.server.rename(old_server)
            config.staging.rename(config.server)
            installed = True
            print("Starting new backend and checking health...", flush=True)
            service.start()
            health(config)
            if not service.is_active():
                raise DeployError("service became inactive after health check")
            service.verify_running_binary()
        except Exception as exc:
            try:
                if installed:
                    # A failed start may still leave a process writing files or holding the port.
                    service.stop()
                    config.failed.parent.mkdir(parents=True, exist_ok=True)
                    config.server.rename(config.failed)
                if old_server.exists():
                    old_server.rename(config.server)
                if stopped:
                    service.start()
                    health(config)
                    if not service.is_active():
                        raise DeployError("previous service is not active")
                    service.verify_running_binary()
            except Exception as recovery:
                raise DeployError(f"deployment failed ({exc}); rollback failed ({recovery}); backup: {config.backup}") from recovery
            raise DeployError(f"deployment failed ({exc}); rolled back to previous backend; saves kept in place; backup: {config.backup}") from exc

        config.package.unlink()
        print(f"Deployment complete: {config.release_id}; backup: {config.backup}; saves kept in place.", flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--release", required=True)
    parser.add_argument("--service", default="mud")
    parser.add_argument("--executable", default="mud")
    parser.add_argument("--port", type=int, default=4009)
    parser.add_argument("--sha256", required=True)
    parser.add_argument("--stage-only", action="store_true")
    args = parser.parse_args()
    config = Config(args.root, args.release, args.service, args.executable, args.port, args.sha256)
    try:
        deploy(config, stage_only=args.stage_only)
    except Exception as exc:
        parser.exit(1, f"Deployment error: {exc}\n")


if __name__ == "__main__":
    main()
