"""验证国际化检查能够拦截真实的缺译和 SQL 失效记录。"""

import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import sync_locales
import verify_i18n


class I18nVerificationTests(unittest.TestCase):
    """使用独立目录验证检查规则，不修改仓库资源。"""

    def setUp(self) -> None:
        """创建包含三个前端和后端语言目录的临时仓库。"""
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.directories = {
            "backend": self.root / "backend/internal/i18n/assets",
            "admin-core": self.root / "frontend/admin/packages/core/src/locales",
            "uni-core": self.root / "frontend/uni-app/packages/core/src/locales",
            "taro-core": self.root / "frontend/taro-app/packages/core/src/locales",
        }
        for directory in self.directories.values():
            for locale in ("zh-CN", "en-US", "zh-TW", "ja-JP"):
                self.write(directory / f"{locale}.json", "{}")
        self.addCleanup(patch.stopall)
        patch.object(verify_i18n, "ROOT", self.root).start()
        patch.object(verify_i18n, "LOCALE_DIRECTORIES", self.directories).start()

    def write(self, path: Path, content: str) -> None:
        """写入检查需要的临时源码或语言资源。"""
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")

    def test_discovers_mobile_system_bundles(self) -> None:
        """移动端的真实 workspace 路径必须参与语言发现。"""
        for workspace in ("uni-app", "taro-app"):
            (self.root / f"frontend/{workspace}/packages/modules/system/src/locales").mkdir(parents=True)
        with patch.object(sync_locales, "ROOT", self.root):
            directories = sync_locales.discover_locale_directories()
        self.assertIn("uni-system", directories)
        self.assertIn("taro-system", directories)

    def test_wrap_internal_nested_arguments_and_escaped_messages(self) -> None:
        """嵌套错误表达式和跨行调用不能让 WrapInternal 文案漏检。"""
        source = 'errorsx.WrapInternal(errors.Join(err, other),\n "查询\\\"模型\\\"失败")'
        self.write(self.root / "backend/internal/service.go", source)
        with self.assertRaisesRegex(verify_i18n.VerificationError, "legacy.error"):
            verify_i18n.verify_go_error_messages()
        message = '查询"模型"失败'
        key = "legacy.error." + hashlib.sha256(message.encode()).hexdigest()[:16]
        for path in self.directories["backend"].glob("*.json"):
            self.write(path, json.dumps({key: {"other": message}}, ensure_ascii=False))
        verify_i18n.verify_go_error_messages()

    def test_shared_openapi_message_must_have_identical_translation(self) -> None:
        """OpenAPI 合并资源前应发现后端与管理端的同名译文冲突。"""
        for path in self.directories["backend"].glob("*.json"):
            self.write(path, json.dumps({
                "system.code.gen.test": {"other": "Test"},
                "common.status.disabled": {"other": "Disabled"},
            }))
        for path in self.directories["admin-core"].glob("*.json"):
            self.write(path, '{"common.status.disabled": "Disabled"}')
        files = {name: sync_locales.locale_files(directory) for name, directory in self.directories.items()}
        sync_locales.validate_locale_sets(files)
        self.write(self.directories["admin-core"] / "ja-JP.json", '{"common.status.disabled": "Different"}')
        with self.assertRaisesRegex(ValueError, "共享词条内容不一致"):
            sync_locales.validate_locale_sets(files)

    def test_missing_proto_cel_key(self) -> None:
        """缺少 CEL RuleId 的翻译时必须失败。"""
        self.write(self.root / "backend/api/proto/login.proto", 'id: "base.login.tenant_code.required"')
        with self.assertRaisesRegex(verify_i18n.VerificationError, "base.login.tenant_code.required"):
            verify_i18n.verify_go_error_messages()

    def test_frontend_keys_include_indirect_mapping_and_all_locales(self) -> None:
        """按钮 key 映射与其他语言漏项均应被前端引用检查拦截。"""
        self.write(
            self.root / "frontend/uni-app/packages/core/src/view.vue",
            'const actions = [{ labelKey: "common.action.back" }]; t(actions[0].labelKey)',
        )
        with self.assertRaisesRegex(verify_i18n.VerificationError, "common.action.back"):
            verify_i18n.verify_frontend_message_keys()
        for path in self.directories["uni-core"].glob("*.json"):
            self.write(path, '{"common.action.back": "Back"}')
        verify_i18n.verify_frontend_message_keys()
        self.write(self.directories["uni-core"] / "ja-JP.json", "{}")
        with self.assertRaisesRegex(verify_i18n.VerificationError, "common.action.back"):
            verify_i18n.verify_frontend_message_keys()

    def test_backend_persisted_message_needs_frontend_translation(self) -> None:
        """后端日志标记必须能在管理端语言包中渲染。"""
        self.write(self.root / "backend/internal/task.go", 'i18n.EncodeMessage("system.base.job.result.done", nil)')
        with self.assertRaisesRegex(verify_i18n.VerificationError, "system.base.job.result.done"):
            verify_i18n.verify_frontend_message_keys()

    def test_sql_orphans_and_postgres_translation_drift(self) -> None:
        """同时验证无源译文和 PostgreSQL 单独修改译文的情况。"""
        locales = ["zh-CN", "en-US", "zh-TW", "ja-JP"]
        migration = self.root / "backend/migration/assets/v0.0.1"
        for database in ("mysql", "postgres"):
            prefix = "INSERT IGNORE INTO `" if database == "mysql" else 'INSERT INTO "'
            quote = "`" if database == "mysql" else '"'
            suffix = ";" if database == "mysql" else " ON CONFLICT DO NOTHING;"
            self.write(migration / database / "default_data.up.sql", f"{prefix}base_dict{quote} VALUES (1000, 'kind', '类型'){suffix}")
            for locale in locales[1:]:
                self.write(migration / database / f"i18n.{locale}.up.sql", f"{prefix}base_i18n{quote} VALUES ('base_dict.name', 1000, '{locale}', 'Type'){suffix}")
        verify_i18n.verify_sql(locales, "zh-CN")
        path = migration / "postgres/i18n.en-US.up.sql"
        content = path.read_text()
        self.write(path, content.replace("'Type'", "'Different'"))
        with self.assertRaisesRegex(verify_i18n.VerificationError, "MySQL/PostgreSQL"):
            verify_i18n.verify_sql(locales, "zh-CN")
        self.write(path, content + "\n" + content.replace("1000", "9999"))
        with self.assertRaisesRegex(verify_i18n.VerificationError, "无默认源记录"):
            verify_i18n.verify_sql(locales, "zh-CN")


if __name__ == "__main__":
    unittest.main()
