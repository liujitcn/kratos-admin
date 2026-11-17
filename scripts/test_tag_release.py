import tempfile
import unittest
from pathlib import Path

from scripts import tag_release


class TagReleaseTests(unittest.TestCase):
    def test_release_tags_share_version_and_start_with_api(self) -> None:
        self.assertEqual(
            tag_release.release_tags((0, 1, 19)),
            (
                "backend/api/v0.1.19",
                "backend/client/v0.1.19",
                "backend/v0.1.19",
                "v0.1.19",
                "npm/v0.1.19",
            ),
        )

    def test_update_api_module_version_rewrites_dependency(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            go_mod = Path(directory) / "go.mod"
            go_mod.write_text(
                "module example.com/app\n\n"
                "require github.com/liujitcn/kratos-admin/backend/api v0.0.36\n",
                encoding="utf-8",
            )

            changed = tag_release.update_api_module_version(go_mod, (0, 1, 19))

            self.assertTrue(changed)
            self.assertIn(
                "github.com/liujitcn/kratos-admin/backend/api v0.1.19",
                go_mod.read_text(encoding="utf-8"),
            )


if __name__ == "__main__":
    unittest.main()
