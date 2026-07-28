#!/usr/bin/env python3

import json
import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("publish_packages.sh")


class PublishPackagesTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.bin_dir = self.root / "bin"
        self.bin_dir.mkdir()
        self.log_file = self.root / "publish.log"
        self.marker_file = self.root / "first.done"
        self.write_command(
            "npm",
            """
            if [[ "$1" == "view" && "$2" == "${PUBLISHED_SPEC:-}" ]]; then
              exit 0
            fi
            exit 1
            """,
        )
        self.write_command(
            "pnpm",
            """
            package_name="$(basename "$PWD")"
            if [[ "$package_name" == "admin" ]]; then
              sleep 0.1
              touch "$FIRST_DONE"
            elif [[ "${REQUIRE_FIRST_DONE:-false}" == "true" && ! -f "$FIRST_DONE" ]]; then
              echo "second publish started before first completed" >&2
              exit 1
            fi
            echo "$package_name|${CI:-unset}|$*" >> "$PUBLISH_LOG"
            """,
        )

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def write_command(self, name: str, body: str) -> None:
        path = self.bin_dir / name
        path.write_text(
            "#!/usr/bin/env bash\nset -euo pipefail\n" + textwrap.dedent(body),
            encoding="utf-8",
        )
        path.chmod(0o755)

    def write_package(self, name: str) -> Path:
        package_dir = self.root / name
        package_dir.mkdir()
        package = {"name": f"@test/{name}", "version": "1.2.3"}
        (package_dir / "package.json").write_text(json.dumps(package), encoding="utf-8")
        return package_dir

    def run_publish(
        self,
        *package_dirs: Path,
        published_spec: str = "",
        require_first_done: bool = False,
    ) -> subprocess.CompletedProcess[str]:
        env = os.environ.copy()
        env.update(
            {
                "FIRST_DONE": str(self.marker_file),
                "NPM_EXPECTED_VERSION": "1.2.3",
                "PATH": f"{self.bin_dir}:{env['PATH']}",
                "PUBLISHED_SPEC": published_spec,
                "PUBLISH_LOG": str(self.log_file),
                "REQUIRE_FIRST_DONE": str(require_first_done).lower(),
            }
        )
        env.pop("CI", None)
        relative_dirs = [os.path.relpath(path, SCRIPT.parent.parent) for path in package_dirs]
        return subprocess.run(
            ["bash", str(SCRIPT), *relative_dirs],
            capture_output=True,
            env=env,
            text=True,
        )

    def test_skips_an_already_published_version(self) -> None:
        admin = self.write_package("admin")
        app = self.write_package("app")

        result = self.run_publish(admin, app, published_spec="@test/admin@1.2.3")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("@test/admin@1.2.3", result.stdout)
        self.assertEqual(
            self.log_file.read_text(encoding="utf-8").splitlines(),
            ["app|unset|publish --registry https://registry.npmjs.org/ --access public --tag latest --no-git-checks"],
        )

    def test_waits_for_each_publish_before_starting_the_next(self) -> None:
        admin = self.write_package("admin")
        app = self.write_package("app")

        result = self.run_publish(admin, app, require_first_done=True)

        self.assertEqual(result.returncode, 0, result.stderr)
        lines = self.log_file.read_text(encoding="utf-8").splitlines()
        self.assertEqual([line.split("|", 1)[0] for line in lines], ["admin", "app"])


if __name__ == "__main__":
    unittest.main()
