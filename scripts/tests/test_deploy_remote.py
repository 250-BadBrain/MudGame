import hashlib
import io
import json
import sys
import tarfile
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from deploy_remote import Config, DeployError, deploy


class FakeService:
    def __init__(self, config, fail_start=False):
        self.config = config
        self.events = []
        self.active = True
        self.fail_start = fail_start

    def preflight(self, config):
        return config.root / "data" / "servers" / "server1"

    def is_active(self):
        return self.active

    def verify_running_binary(self):
        pass

    def stop(self):
        self.events.append("stop")
        self.active = False
        (self.preflight(self.config) / "player.json").write_text("flushed-save")

    def start(self):
        self.events.append("start")
        self.active = True
        if self.fail_start and (self.config.server / "mud").read_bytes() != b"old-program":
            raise DeployError("new service start failed")


class DeployTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.release = "20261026-test"
        self.config = Config(self.root, self.release, "mud", "mud", 4009, "")
        self.config.server.mkdir()
        (self.config.server / "mud").write_bytes(b"old-program")
        (self.config.server / "resources").mkdir()
        (self.config.server / "resources" / "old.json").write_text("{}")
        data = self.root / "data" / "servers" / "server1"
        data.mkdir(parents=True)
        (data / "player.json").write_text("original-save")
        self.package()

    def package(self, extra=None, corrupt_hash=False):
        elf = bytearray(64)
        elf[:6] = b"\x7fELF\x02\x01"
        elf[18] = 62
        files = {"mud": bytes(elf)}
        for name in ("items", "skills", "npcs", "titles"):
            files[f"resources/{name}.json"] = b"{}"
        files["resources/worlds/test.json"] = b"{}"
        manifest = {
            "releaseId": self.release,
            "target": "linux/amd64",
            "files": {name: hashlib.sha256(value).hexdigest() for name, value in files.items()},
        }
        if corrupt_hash:
            manifest["files"]["mud"] = "0" * 64
        files["release.json"] = json.dumps(manifest).encode()
        if extra:
            files.update(extra)
        self.config.package.parent.mkdir(parents=True, exist_ok=True)
        with tarfile.open(self.config.package, "w:gz") as archive:
            for name, value in files.items():
                entry = tarfile.TarInfo(name)
                entry.size = len(value)
                archive.addfile(entry, io.BytesIO(value))
        self.config.archive_sha256 = hashlib.sha256(self.config.package.read_bytes()).hexdigest()

    def test_success_preserves_data_and_backs_up_after_flush(self):
        service = FakeService(self.config)
        deploy(self.config, service, lambda _: None)
        self.assertEqual(service.events, ["stop", "start"])
        self.assertTrue(service.active)
        self.assertEqual((self.config.backup / "MudGame-server" / "mud").read_bytes(), b"old-program")
        self.assertEqual((self.config.backup / "data-snapshot" / "player.json").read_text(), "flushed-save")
        self.assertEqual((service.preflight(self.config) / "player.json").read_text(), "flushed-save")
        self.assertEqual((self.config.server / "mud").read_bytes()[:4], b"\x7fELF")
        self.assertFalse(self.config.package.exists())
        self.assertFalse((self.root / ".deploy" / "deploy.lock").exists())

    def test_stage_only_does_not_stop_service_or_change_program(self):
        service = FakeService(self.config)
        deploy(self.config, service, lambda _: self.fail("must not probe new service"), stage_only=True)
        self.assertEqual(service.events, [])
        self.assertEqual((self.config.server / "mud").read_bytes(), b"old-program")
        self.assertEqual((service.preflight(self.config) / "player.json").read_text(), "original-save")
        self.assertFalse(self.config.staging.exists())
        self.assertFalse(self.config.package.exists())

    def test_unhealthy_release_stops_new_service_then_restores_old_program(self):
        service = FakeService(self.config)

        def health(_):
            if (self.config.server / "mud").read_bytes() != b"old-program":
                raise DeployError("health check failed")

        with self.assertRaisesRegex(DeployError, "rolled back"):
            deploy(self.config, service, health)
        self.assertEqual(service.events, ["stop", "start", "stop", "start"])
        self.assertEqual((self.config.server / "mud").read_bytes(), b"old-program")
        self.assertTrue(service.active)
        self.assertTrue((self.config.failed / "mud").exists())

    def test_partially_started_service_is_stopped_before_rollback(self):
        service = FakeService(self.config, fail_start=True)
        with self.assertRaisesRegex(DeployError, "rolled back"):
            deploy(self.config, service, lambda _: None)
        self.assertEqual(service.events, ["stop", "start", "stop", "start"])
        self.assertEqual((self.config.server / "mud").read_bytes(), b"old-program")

    def test_hash_mismatch_does_not_stop_production(self):
        self.package(corrupt_hash=True)
        service = FakeService(self.config)
        with self.assertRaisesRegex(DeployError, "checksum"):
            deploy(self.config, service, lambda _: None)
        self.assertEqual(service.events, [])

    def test_archive_checksum_mismatch_does_not_stop_production(self):
        self.config.archive_sha256 = "0" * 64
        service = FakeService(self.config)
        with self.assertRaisesRegex(DeployError, "checksum"):
            deploy(self.config, service, lambda _: None)
        self.assertEqual(service.events, [])

    def test_package_cannot_include_saves_or_escape_staging(self):
        for extra in ("data/player.json", "../escape", "/absolute", "resources/../../escape"):
            with self.subTest(extra=extra):
                self.package(extra={extra: b"{}"})
                service = FakeService(self.config)
                with self.assertRaises(DeployError):
                    deploy(self.config, service, lambda _: None)
                self.assertEqual(service.events, [])

    def test_concurrent_deployment_is_rejected(self):
        lock = self.root / ".deploy" / "deploy.lock"
        lock.write_text("another deployment")
        service = FakeService(self.config)
        with self.assertRaisesRegex(DeployError, "lock"):
            deploy(self.config, service, lambda _: None)
        self.assertEqual(service.events, [])
        self.assertEqual(lock.read_text(), "another deployment")

    def test_save_directory_cannot_contain_its_own_backup(self):
        service = FakeService(self.config)
        service.preflight = lambda _: self.root
        with self.assertRaisesRegex(DeployError, "inside the save directory"):
            deploy(self.config, service, lambda _: None)
        self.assertEqual(service.events, [])


if __name__ == "__main__":
    unittest.main()
