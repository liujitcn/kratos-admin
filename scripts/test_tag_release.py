import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

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

    def test_final_release_tags_exclude_already_published_api(self) -> None:
        api_tag, *final_tags = tag_release.release_tags((0, 1, 19))
        with (
            patch.object(tag_release, "tag_exists_locally", side_effect=lambda tag: tag == api_tag),
            patch.object(tag_release, "tag_exists_remotely", return_value=False),
        ):
            self.assertEqual(tag_release.resolve_release_tags((0, 1, 19)), tuple(final_tags))

    def test_ensure_api_tag_reuses_published_module_with_unchanged_contents(self) -> None:
        with (
            patch.object(tag_release, "tag_commit", return_value="api-commit"),
            patch.object(tag_release, "tag_exists_remotely", return_value=True),
            patch.object(tag_release, "run", return_value="same-tree\nsame-tree") as run,
        ):
            tag_release.ensure_api_tag((0, 1, 19), "release-commit")

        run.assert_called_once_with(
            ["git", "rev-parse", "api-commit:backend/api", "release-commit:backend/api"]
        )

    def test_ensure_api_tag_rejects_published_module_with_changed_contents(self) -> None:
        with (
            patch.object(tag_release, "tag_commit", return_value="api-commit"),
            patch.object(tag_release, "run", return_value="old-tree\nnew-tree") as run,
            self.assertRaisesRegex(RuntimeError, "内容不同.*VERSION"),
        ):
            tag_release.ensure_api_tag((0, 1, 19), "release-commit")

        self.assertEqual(run.call_count, 1)

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
