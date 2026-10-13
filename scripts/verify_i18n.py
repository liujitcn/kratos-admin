#!/usr/bin/env python3
"""发布前只读校验国际化语言包及其生成产物。"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
from collections.abc import Iterator
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT = SCRIPT_DIR.parent
sys.path.insert(0, str(SCRIPT_DIR))

from generate_locale_drafts import parse_primary_i18n_sources, parse_sql_values  # noqa: E402
from sync_locales import (  # noqa: E402
    DEFAULT_LOCALE,
    FRONTEND_GENERATED_FILES,
    LOCALE_DIRECTORIES,
    locale_files,
    render_frontend_generated,
    validate_locale_sets,
)


LOCALE_PATTERN = re.compile(r"^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$")
TARGET_KEY_PATTERN = re.compile(r"^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$")
SQL_FILE_PATTERN = re.compile(r"^i18n\.(?P<locale>[^.]+)\.up\.sql$")
SQL_INSERT_PATTERN = re.compile(r'INSERT(?: IGNORE)? INTO [`"]base_i18n[`"]')
GO_ERROR_CALL_PATTERN = re.compile(r'errorsx\.(?P<method>\w+)\s*\(')
SOURCE_TOKEN_PATTERN = re.compile(r'"(?:\\.|[^"\\])*"|\x27(?:\\.|[^\x27\\])*\x27|`[^`]*`|//[^\n]*|/\*[\s\S]*?\*/|[()\[\]{},]')
PROTO_MESSAGE_KEY_PATTERN = re.compile(r'\bid:\s*"([^"\n]+)"')
FRONTEND_MESSAGE_KEY_PATTERN = re.compile(r"[\"'`]((?:common|core|system|rag)\.[A-Za-z0-9_.-]+)[\"'`]")
SSE_EVENT_NAMES = {"system.admin.codegen", "system.admin.runtime-console", "system.admin.ops-monitoring"}
PERSISTED_MESSAGE_KEY_PATTERN = re.compile(r'(?:EncodeMessage|WrapMessageError)\(\s*"([^"\n]+)"|(?:NameKey|DescriptionKey):\s*"([^"\n]+)"')
GO_ERROR_KEY_PREFIX = "legacy.error."
OPENAPI_LOCALIZED_FIELD_PATTERN = re.compile(
    r"^(?P<indent>\s*)(?P<field>description|summary|title):(?:\s.*)?$"
)
OPENAPI_REQUIRED_MARKERS = ("openapi:", "info:", "paths:", "components:", "tags:")
OPENAPI_CJK_PATTERN = re.compile(r"[一-龥]")
OPENAPI_JA_UNTRANSLATED_TERMS = ("出库", "入库")
SOURCE_LOCALE = DEFAULT_LOCALE


class VerificationError(ValueError):
    """表示一个国际化校验项失败。"""


def parse_locales(value: str) -> list[str]:
    """解析并去重命令行传入的目标语言。"""
    locales: list[str] = []
    for item in value.split(","):
        locale = item.strip()
        if not locale:
            continue
        if not LOCALE_PATTERN.fullmatch(locale):
            raise VerificationError(f"语言代码不是有效的 BCP 47 代码: {locale}")
        if locale not in locales:
            locales.append(locale)
    if not locales:
        raise VerificationError("目标语言列表不能为空")
    return locales


def parse_sql_records(path: Path) -> list[tuple[str, int, str, str]]:
    """读取一个 i18n SQL 文件中的翻译记录。"""
    records: list[tuple[str, int, str, str]] = []
    for line_number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
        if not SQL_INSERT_PATTERN.search(line):
            continue
        values = parse_sql_values(line)
        if values is None or len(values) != 4:
            raise VerificationError(f"{path}:{line_number} 的 base_i18n INSERT 格式无效")
        try:
            target_key = str(values[0] or "")
            target_id = int(values[1] or "")
        except ValueError as error:
            raise VerificationError(f"{path}:{line_number} 的翻译目标编号无效") from error
        if not TARGET_KEY_PATTERN.fullmatch(target_key):
            raise VerificationError(f"{path}:{line_number} 的翻译目标键格式无效")
        locale = str(values[2] or "")
        name = str(values[3] or "")
        if not locale:
            raise VerificationError(f"{path}:{line_number} 的翻译语言为空")
        records.append((target_key, target_id, locale, name))
    if not records:
        raise VerificationError(f"SQL 文件没有 base_i18n 翻译记录: {path}")
    return records


def verify_language_bundles() -> list[str]:
    """校验语言集合、语言键、占位符和前端注册生成物。"""
    file_sets = {name: locale_files(directory) for name, directory in LOCALE_DIRECTORIES.items()}
    locales = validate_locale_sets(file_sets)
    for name, path in FRONTEND_GENERATED_FILES.items():
        expected = render_frontend_generated(locales, name == "admin-core", name == "admin-core")
        if not path.exists() or path.read_text(encoding="utf-8") != expected:
            raise VerificationError(f"前端语言注册生成物过期，请执行 make i18n: {path}")
    return locales


def verify_sql(locales: list[str], source_locale: str) -> None:
    """校验两种数据库的译文集合、默认源记录和跨数据库一致性。"""
    expected_locales = set(locales) - {source_locale}
    migration_root = ROOT / "backend/migration/assets"
    groups: dict[Path, dict[str, Path]] = {}
    for database in ("mysql", "postgres"):
        for path in migration_root.glob(f"*/{database}/i18n.*.up.sql"):
            match = SQL_FILE_PATTERN.fullmatch(path.name)
            if match is None or not LOCALE_PATTERN.fullmatch(match.group("locale")):
                raise VerificationError(f"SQL 文件名不是 i18n.<locale>.up.sql: {path}")
            groups.setdefault(path.parent, {})[match.group("locale")] = path
    if not groups:
        raise VerificationError(f"未找到国际化 SQL 文件: {migration_root}")

    database_records: dict[tuple[Path, str], set[tuple[str, int, str, str]]] = {}
    for directory, files in sorted(groups.items(), key=lambda item: str(item[0])):
        if set(files) != expected_locales:
            raise VerificationError(
                f"{directory} 的 SQL 语言集合不一致: "
                f"缺少 {sorted(expected_locales - set(files))}，多出 {sorted(set(files) - expected_locales)}"
            )
        sources = parse_primary_i18n_sources(directory / "default_data.up.sql")
        required_keys = {
            key for key, value in sources.items()
            if key[0] != "base_config.value" and value
        }
        reference_keys: set[tuple[str, int]] | None = None
        for locale, path in sorted(files.items()):
            records = parse_sql_records(path)
            keys = {(item[0], item[1]) for item in records}
            if len(keys) != len(records):
                raise VerificationError(f"SQL 翻译记录重复: {path}")
            if reference_keys is not None and keys != reference_keys:
                raise VerificationError(f"{path} 的翻译目标集合与其他语言不一致")
            reference_keys = keys
            if any(item[2] != locale for item in records):
                raise VerificationError(f"{path} 包含与文件名不一致的 locale")
            missing = sorted(required_keys - keys)
            orphaned = sorted(keys - set(sources))
            if missing or orphaned:
                raise VerificationError(f"{path} 缺少默认译文 {missing}，无默认源记录的译文 {orphaned}")
            database_records[(directory, locale)] = set(records)

    for (directory, locale), records in database_records.items():
        if directory.name != "mysql":
            continue
        postgres_directory = directory.parent / "postgres"
        if postgres_directory.exists() and database_records.get((postgres_directory, locale)) != records:
            raise VerificationError(f"{directory.parent} 的 MySQL/PostgreSQL {locale} 译文不一致")


def normalized_openapi(text: str) -> tuple[str, ...]:
    """屏蔽可翻译字段值，保留 OpenAPI 契约结构和非翻译内容。"""
    normalized: list[str] = []
    for line in text.splitlines():
        match = OPENAPI_LOCALIZED_FIELD_PATTERN.fullmatch(line)
        if match:
            normalized.append(f"{match.group('indent')}{match.group('field')}: <localized>")
        else:
            normalized.append(line)
    return tuple(normalized)


def verify_openapi(target_locales: list[str]) -> None:
    """校验 OpenAPI 源文档和各语言文档的契约结构一致。"""
    output_dir = ROOT / "backend/internal/openapi/assets"
    source_path = output_dir / "openapi.yaml"
    source_text = source_path.read_text(encoding="utf-8")
    if any(marker not in source_text for marker in OPENAPI_REQUIRED_MARKERS):
        raise VerificationError(f"OpenAPI 源文档缺少必要结构: {source_path}")
    source_shape = normalized_openapi(source_text)
    for locale in target_locales:
        path = output_dir / f"openapi.{locale}.yaml"
        if not path.exists():
            raise VerificationError(f"缺少 OpenAPI {locale} 文档: {path}")
        text = path.read_text(encoding="utf-8")
        if any(marker not in text for marker in OPENAPI_REQUIRED_MARKERS):
            raise VerificationError(f"OpenAPI {locale} 文档缺少必要结构: {path}")
        if normalized_openapi(text) != source_shape:
            raise VerificationError(
                f"OpenAPI {locale} 文档与源文档结构不一致，请执行 make i18n: {path}"
            )
        localized_fields = [
            line
            for line in text.splitlines()
            if OPENAPI_LOCALIZED_FIELD_PATTERN.fullmatch(line)
        ]
        if locale == "en-US":
            residual = [line.strip() for line in localized_fields if OPENAPI_CJK_PATTERN.search(line)]
            if residual:
                raise VerificationError(
                    f"OpenAPI {locale} 文案仍包含中文，请补充本地化术语: {path}: {residual[:3]}"
                )
        if locale == "ja-JP":
            residual = [
                line.strip()
                for line in localized_fields
                if any(term in line for term in OPENAPI_JA_UNTRANSLATED_TERMS)
            ]
            if residual:
                raise VerificationError(
                    f"OpenAPI {locale} 文案仍包含简体中文术语，请补充本地化术语: {path}: {residual[:3]}"
                )


def go_error_messages(source: str) -> Iterator[tuple[int, str]]:
    """提取 errorsx 静态文案，正确跳过错误实参内的嵌套调用、逗号和字符串。"""
    for call in GO_ERROR_CALL_PATTERN.finditer(source):
        arguments: list[str] = []
        start = call.end()
        depth = 0
        for token in SOURCE_TOKEN_PATTERN.finditer(source, start):
            value = token.group()
            if value in ("(", "[", "{"):
                depth += 1
            elif value == ")" and depth == 0:
                arguments.append(source[start:token.start()].strip())
                break
            elif value in (")", "]", "}"):
                depth -= 1
            elif value == "," and depth == 0:
                arguments.append(source[start:token.start()].strip())
                start = token.end()
        index = 1 if call.group("method") == "WrapInternal" else 0
        if len(arguments) <= index:
            continue
        argument = arguments[index]
        if re.fullmatch(r'"(?:\\.|[^"\\])*"', argument):
            yield call.start(), json.loads(argument)
        elif re.fullmatch(r'`[^`]*`', argument):
            yield call.start(), argument[1:-1]


def verify_go_error_messages() -> None:
    """校验 errorsx 构造文案、WrapInternal 文案及 Proto CEL 消息键。"""
    bundles = {
        locale: json.loads(path.read_text(encoding="utf-8"))
        for locale, path in locale_files(LOCALE_DIRECTORIES["backend"]).items()
    }
    missing: list[str] = []
    go_root = ROOT / "backend"
    for path in go_root.rglob("*.go"):
        if path.name.endswith("_test.go") or "gen" in path.parts or "migration" in path.parts:
            continue
        source = path.read_text(encoding="utf-8")
        for offset, message in go_error_messages(source):
            key = GO_ERROR_KEY_PREFIX + hashlib.sha256(message.encode()).hexdigest()[:16]
            if any(key not in bundle for bundle in bundles.values()):
                line = source.count("\n", 0, offset) + 1
                missing.append(f"{path.relative_to(ROOT)}:{line}: {key} ({message})")
    for path in (go_root / "api/proto").rglob("*.proto"):
        source = path.read_text(encoding="utf-8")
        for match in PROTO_MESSAGE_KEY_PATTERN.finditer(source):
            key = match.group(1)
            if any(key not in bundle for bundle in bundles.values()):
                line = source.count("\n", 0, match.start()) + 1
                missing.append(f"{path.relative_to(ROOT)}:{line}: {key}")
    if missing:
        raise VerificationError("后端消息缺少语言包词条: " + "；".join(missing))


def verify_frontend_message_keys() -> None:
    """校验界面命名空间中的静态 key 和后端下发给前端的消息键。"""
    bundles: dict[str, dict[str, set[str]]] = {}
    for name, directory in LOCALE_DIRECTORIES.items():
        if name == "backend":
            continue
        terminal = name.split("-", 1)[0]
        for locale, path in locale_files(directory).items():
            bundles.setdefault(terminal, {}).setdefault(locale, set()).update(
                json.loads(path.read_text(encoding="utf-8"))
            )
    missing: list[str] = []
    for terminal, workspace in (("admin", "admin"), ("uni", "uni-app"), ("taro", "taro-app")):
        for directory, children, files in os.walk(ROOT / "frontend" / workspace):
            children[:] = [name for name in children if name not in {
                "node_modules", "dist", "rpc", "locales", "tests", "test"
            }]
            for filename in files:
                path = Path(directory) / filename
                if path.suffix not in {".ts", ".tsx", ".vue", ".mjs"} or re.search(r"\.(?:test|spec)\.", filename):
                    continue
                source = path.read_text(encoding="utf-8")
                for match in FRONTEND_MESSAGE_KEY_PATTERN.finditer(source):
                    key = match.group(1)
                    # SSE 名称属于事件协议；结尾点号是动态查找前缀。
                    if key.endswith(".") or key in SSE_EVENT_NAMES:
                        continue
                    if any(key not in keys for keys in bundles[terminal].values()):
                        line = source.count("\n", 0, match.start()) + 1
                        missing.append(f"{path.relative_to(ROOT)}:{line}: {key}")
    for path in (ROOT / "backend").rglob("*.go"):
        if path.name.endswith("_test.go") or "gen" in path.parts:
            continue
        source = path.read_text(encoding="utf-8")
        for match in PERSISTED_MESSAGE_KEY_PATTERN.finditer(source):
            key = match.group(1) or match.group(2)
            if any(key not in keys for keys in bundles["admin"].values()):
                line = source.count("\n", 0, match.start()) + 1
                missing.append(f"{path.relative_to(ROOT)}:{line}: {key}")
    if missing:
        raise VerificationError("前端引用未定义的语言 key: " + "；".join(missing))


def main() -> int:
    """执行全部国际化发布前校验。"""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-locale", default=SOURCE_LOCALE, help="语言包主语言")
    parser.add_argument("--locales", default="en-US,zh-TW,ja-JP", help="OpenAPI 目标语言")
    args = parser.parse_args()

    try:
        target_locales = parse_locales(args.locales)
        target_locales = [locale for locale in target_locales if locale != args.source_locale]
    except VerificationError as error:
        print(f"国际化发布校验失败: {error}", file=sys.stderr)
        return 1

    errors: list[str] = []
    locales: list[str] | None = None
    try:
        locales = verify_language_bundles()
    except (OSError, json.JSONDecodeError, VerificationError, ValueError) as error:
        errors.append(f"语言包: {error}")

    if locales is not None:
        if args.source_locale not in locales:
            errors.append(f"语言包: 缺少主语言 {args.source_locale}")
        try:
            verify_sql(locales, SOURCE_LOCALE)
        except (OSError, json.JSONDecodeError, VerificationError, ValueError) as error:
            errors.append(f"SQL: {error}")

    try:
        verify_openapi(target_locales)
    except (OSError, json.JSONDecodeError, VerificationError, ValueError) as error:
        errors.append(f"OpenAPI: {error}")

    try:
        verify_go_error_messages()
    except (OSError, json.JSONDecodeError, VerificationError, ValueError) as error:
        errors.append(f"后端消息: {error}")

    try:
        verify_frontend_message_keys()
    except (OSError, json.JSONDecodeError, VerificationError, ValueError) as error:
        errors.append(f"前端引用: {error}")

    if errors:
        print("国际化发布校验失败:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    assert locales is not None

    print(
        "国际化发布校验通过："
        f"语言包({len(locales)})、前端引用、后端错误/Proto、SQL({len(set(locales) - {SOURCE_LOCALE})}种目标语言)、"
        f"OpenAPI({len(target_locales)}种目标语言)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
