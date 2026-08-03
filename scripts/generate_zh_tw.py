#!/usr/bin/env python3
"""生成非默认语言的固定语言包和初始化翻译数据。"""

from __future__ import annotations

import argparse
from collections import Counter
import json
import os
import re
import sys
import time
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any, Callable


ROOT = Path(__file__).resolve().parents[1]
SQL_DIR = ROOT / "backend/migration/assets/v0.0.1/mysql"
JSON_SOURCES = [
    ROOT / "backend/internal/i18n/locales/zh-CN.json",
    ROOT / "frontend/admin/packages/core/src/locales/zh-CN.json",
    ROOT / "frontend/admin/packages/modules/system/src/locales/zh-CN.json",
    ROOT / "frontend/uni-app/packages/core/src/locales/zh-CN.json",
    ROOT / "frontend/uni-app/packages/modules/system/src/locales/zh-CN.json",
    ROOT / "frontend/taro-app/packages/core/src/locales/zh-CN.json",
    ROOT / "frontend/taro-app/packages/modules/system/src/locales/zh-CN.json",
]
DEFAULT_TARGET_LOCALES = ("zh-TW", "ko-KR", "fr-FR", "es-ES")
FALLBACK_TRANSLATIONS = {
    "es": {
        "Please enter": "Introduzca",
        "Please select": "Seleccione",
        "Please upload": "Cargue",
        "cannot be empty": "no puede estar vacío",
        "cannot exceed": "no puede superar",
        "must be greater than": "debe ser mayor que",
        "is required": "es obligatorio",
        "Required": "Obligatorio",
        "Success": "Éxito",
        "Failed": "Error",
        "Loading": "Cargando",
        "Search": "Buscar",
        "Reset": "Restablecer",
        "Cancel": "Cancelar",
        "Confirm": "Confirmar",
        "Save": "Guardar",
        "Delete": "Eliminar",
        "Edit": "Editar",
        "Create": "Crear",
        "Close": "Cerrar",
        "Back": "Volver",
        "Download": "Descargar",
        "Upload": "Cargar",
        "View": "Ver",
        "Enabled": "Habilitado",
        "Disabled": "Deshabilitado",
        "English": "Inglés",
        "Japanese": "Japonés",
        "Korean": "Coreano",
        "French": "Francés",
        "Spanish": "Español",
        "Traditional Chinese": "Chino tradicional",
        "Please": "Por favor",
        "password": "contraseña",
        "Password": "Contraseña",
        "username": "nombre de usuario",
        "Username": "Nombre de usuario",
        "user": "usuario",
        "User": "Usuario",
        "role": "rol",
        "Role": "Rol",
        "menu": "menú",
        "Menu": "Menú",
        "language": "idioma",
        "Language": "Idioma",
        "system": "sistema",
        "System": "Sistema",
        "configuration": "configuración",
        "Configuration": "Configuración",
        "data": "datos",
        "Data": "Datos",
        "file": "archivo",
        "File": "Archivo",
        "page": "página",
        "Page": "Página",
        "name": "nombre",
        "Name": "Nombre",
        "value": "valor",
        "Value": "Valor",
        "type": "tipo",
        "Type": "Tipo",
        "current": "actual",
        "Current": "Actual",
        "No ": "Sin ",
        " not ": " no ",
    },
}
PROTECTED_PATTERN = re.compile(
    r"(?s)```.*?```|`[^`]+`|\{\{[^{}]+\}\}|\$\{[^{}]+\}|\{[A-Za-z_][A-Za-z0-9_.-]*\}|%[sdv]|</?[^>]+>|https?://[^\s<>()]+|/(?:api|events|mcp|v[0-9]+)/[A-Za-z0-9_./:{}-]+"
)
MIGRATION_VERSION_PATTERN = re.compile(r"^v\d+\.\d+\.\d+$")
ENTRY_PATTERN = re.compile(r"__KRATOS_ENTRY_(\d{4})__")
TOKEN_PATTERN = re.compile(r"__KRATOS_TOKEN_(\d{4})__")
PLACEHOLDER_PATTERN = re.compile(r"\{\{[^{}]+\}\}|\$\{[^{}]+\}|\{[A-Za-z_][A-Za-z0-9_.-]*\}|%[sdv]")


def load_opencc():
    try:
        from opencc import OpenCC
    except ImportError as exc:
        raise SystemExit(
            "生成 zh-TW 需要 OpenCC，请先执行：python3 -m pip install opencc-python-reimplemented"
        ) from exc
    return OpenCC("s2twp")


def convert_value(value: Any, converter) -> Any:
    if isinstance(value, str):
        return converter.convert(value)
    if isinstance(value, list):
        return [convert_value(item, converter) for item in value]
    if isinstance(value, dict):
        return {key: convert_value(item, converter) for key, item in value.items()}
    return value


def protect_text(text: str, index: int) -> tuple[str, dict[str, str]]:
    values: dict[str, str] = {}

    def replace(match: re.Match[str]) -> str:
        token = f"__KRATOS_TOKEN_{len(values):04d}__"
        values[token] = match.group(0)
        return token

    return PROTECTED_PATTERN.sub(replace, text), values


def restore_text(text: str, values: dict[str, str]) -> str:
    for token, value in values.items():
        text = text.replace(token, value)
    for match in TOKEN_PATTERN.finditer(text):
        text = text.replace(match.group(0), values.get(match.group(0), ""))
    for value in values.values():
        if value not in text:
            text = f"{text} {value}".strip()
    return text


def has_expected_placeholders(text: str, protected: dict[str, str]) -> bool:
    expected = Counter(
        placeholder
        for value in protected.values()
        for placeholder in PLACEHOLDER_PATTERN.findall(value)
    )
    return Counter(PLACEHOLDER_PATTERN.findall(text)) == expected


def fallback_translate(text: str, target: str) -> str:
    """使用内置术语表生成无网络环境下的可读语言草稿。"""
    protected, values = protect_text(text, 0)
    replacements = FALLBACK_TRANSLATIONS.get(target, {})
    for source, translated in sorted(replacements.items(), key=lambda item: len(item[0]), reverse=True):
        protected = re.sub(re.escape(source), translated, protected, flags=re.IGNORECASE)
    return restore_text(protected, values)


def google_translate(text: str, source: str, target: str) -> str:
    query = urllib.parse.urlencode(
        [("client", "gtx"), ("sl", source), ("tl", target), ("dt", "t"), ("q", text)]
    )
    endpoint = os.environ.get("I18N_TRANSLATE_ENDPOINT", "http://translate.googleapis.com/translate_a/single")
    request = urllib.request.Request(
        f"{endpoint}?{query}",
        headers={"User-Agent": "kratos-admin-i18n/1.0", "Connection": "close"},
    )
    last_error: Exception | None = None
    for attempt in range(5):
        try:
            with urllib.request.urlopen(request, timeout=20) as response:
                payload = json.loads(response.read().decode("utf-8"))
            return "".join(part[0] for part in payload[0] if part and part[0])
        except Exception as error:  # noqa: BLE001 - network provider failure is retried
            last_error = error
            time.sleep(1.0 * (attempt + 1))
    raise RuntimeError(f"Google V1 翻译失败（{source}->{target}）：{last_error}")


def translate_batch(texts: list[str], source: str, target: str, offline: bool = False) -> list[str]:
    if offline:
        return [fallback_translate(text, target) for text in texts]
    results: list[str] = []
    chunk: list[tuple[str, dict[str, str]]] = []
    chunk_size = 0
    provider_available = True

    def flush() -> None:
        nonlocal chunk, chunk_size, provider_available
        if not chunk:
            return
        if not provider_available:
            results.extend(fallback_translate(restore_text(value, protected), target) for value, protected in chunk)
            chunk = []
            chunk_size = 0
            return
        source_text = "\n".join(f"__KRATOS_ENTRY_{index:04d}__ {value}" for index, (value, _) in enumerate(chunk))
        try:
            translated = google_translate(source_text, source, target)
        except RuntimeError:
            provider_available = False
            results.extend(fallback_translate(restore_text(value, protected), target) for value, protected in chunk)
            chunk = []
            chunk_size = 0
            return
        matches = list(ENTRY_PATTERN.finditer(translated))
        translated_by_index: dict[int, str] = {}
        for position, match in enumerate(matches):
            end = matches[position + 1].start() if position + 1 < len(matches) else len(translated)
            translated_by_index[int(match.group(1))] = translated[match.end() : end].strip()
        for index, (_, protected) in enumerate(chunk):
            result = translated_by_index.get(index)
            if result is None:
                try:
                    result = google_translate(chunk[index][0], source, target)
                except RuntimeError:
                    provider_available = False
                    result = fallback_translate(restore_text(chunk[index][0], protected), target)
            result = restore_text(result, protected)
            if not has_expected_placeholders(result, protected) and provider_available:
                try:
                    result = restore_text(google_translate(chunk[index][0], source, target), protected)
                except RuntimeError:
                    provider_available = False
            if not has_expected_placeholders(result, protected):
                result = (
                    fallback_translate(restore_text(chunk[index][0], protected), target)
                    if not provider_available
                    else restore_text(chunk[index][0], protected)
                )
            results.append(result)
        chunk = []
        chunk_size = 0

    for index, text in enumerate(texts):
        protected, values = protect_text(text, index)
        if chunk and chunk_size + len(protected) > 1200:
            flush()
        chunk.append((protected, values))
        chunk_size += len(protected)
    flush()
    return results


def collect_strings(value: Any, values: list[str]) -> None:
    if isinstance(value, str):
        values.append(value)
    elif isinstance(value, list):
        for item in value:
            collect_strings(item, values)
    elif isinstance(value, dict):
        for item in value.values():
            collect_strings(item, values)


def replace_strings(value: Any, replacement: Callable[[], str]) -> Any:
    if isinstance(value, str):
        return replacement()
    if isinstance(value, list):
        return [replace_strings(item, replacement) for item in value]
    if isinstance(value, dict):
        return {key: replace_strings(item, replacement) for key, item in value.items()}
    return value


def generate_json(source: Path, locale: str, converter, machine: bool, offline: bool, write: bool) -> None:
    source_data = json.loads(source.read_text(encoding="utf-8"))
    if locale == "zh-TW":
        target_data = convert_value(source_data, converter)
    else:
        english_source = source.with_name("en-US.json")
        english_data = json.loads(english_source.read_text(encoding="utf-8"))
        if offline:
            source_data = english_data
        source_values: list[str] = []
        fallback_values: list[str] = []
        collect_strings(source_data, source_values)
        collect_strings(english_data, fallback_values)
        translated = (
            translate_batch(source_values, "en" if offline else "zh-CN", locale.split("-")[0], offline)
            if machine or offline
            else source_values
        )
        translated = [value or fallback_values[index] for index, value in enumerate(translated)]
        iterator = iter(translated)
        target_data = replace_strings(source_data, lambda: next(iterator))
    target = source.with_name(f"{locale}.json")
    if write:
        target.write_text(json.dumps(target_data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def parse_menu_titles(default_data: Path) -> dict[int, str]:
    titles: dict[int, str] = {}
    pattern = re.compile(r"INSERT IGNORE INTO `base_menu` .*?VALUES \((\d+),.*?, '(\{.*\})', '\[.*\]'.*\);$")
    for line in default_data.read_text(encoding="utf-8").splitlines():
        match = pattern.match(line)
        if not match:
            continue
        try:
            metadata = json.loads(match.group(2).replace('\\"', '"'))
        except json.JSONDecodeError:
            continue
        if isinstance(metadata.get("title"), str):
            titles[int(match.group(1))] = metadata["title"]
    return titles


def source_text(line: str, menu_titles: dict[int, str]) -> str:
    menu_id = re.search(r"WHERE `id` = (\d+)", line)
    if menu_id:
        return menu_titles.get(int(menu_id.group(1)), "")
    literal = re.search(r"SHA2\('((?:[^'\\]|\\.)*)', 256\)", line)
    return literal.group(1).replace("\\'", "'") if literal else ""


def extract_translation(line: str, locale: str = "en-US") -> str:
    marker = f", '{locale}', "
    if marker not in line:
        return ""
    remainder = line.split(marker, 1)[1]
    suffix_index = remainder.find("', 'reviewed'")
    if suffix_index < 0:
        return ""
    if "base_config_translation" in line:
        field_end = remainder.find(", '")
        return remainder[field_end + 3 : suffix_index] if field_end >= 0 else ""
    return remainder[1:suffix_index]


def replace_translation(line: str, locale: str, translated: str) -> str:
    marker = ", 'en-US', "
    if marker not in line:
        return line.replace("en-US", locale)
    prefix, remainder = line.split(marker, 1)
    suffix_index = remainder.find("', 'reviewed'")
    if suffix_index < 0:
        return line
    escaped = translated.replace("'", "\\'")
    if "base_config_translation" in line:
        field_end = remainder.find(", '")
        if field_end < 0:
            return line
        return f"{prefix}, '{locale}', {remainder[:field_end]}, '{escaped}'{remainder[suffix_index + 1:]}"
    return f"{prefix}, '{locale}', '{escaped}'{remainder[suffix_index + 1:]}"


def generate_sql(locale: str, converter, machine: bool, offline: bool, write: bool, sql_directory: Path) -> None:
    source = SQL_DIR / "translation.en-US.up.sql"
    target = sql_directory / f"translation.{locale}.up.sql"
    menu_titles = parse_menu_titles(SQL_DIR / "default_data.up.sql")
    lines = source.read_text(encoding="utf-8").splitlines()
    values = [source_text(line, menu_titles) if locale == "zh-TW" else extract_translation(line) for line in lines]
    translated = (
        values
        if locale == "zh-TW"
        else translate_batch(values, "en", locale.split("-")[0], offline)
        if machine or offline
        else values
    )
    generated = [
        replace_translation(line, locale, converter.convert(translated[index]) if locale == "zh-TW" else translated[index])
        if "INSERT IGNORE INTO" in line
        else line.replace("en-US", locale)
        for index, line in enumerate(lines)
    ]
    if write:
        target.write_text("\n".join(generated) + "\n", encoding="utf-8")


def render_translation_description(locale: str) -> str:
    return (
        f"由 `scripts/generate_zh_tw.py` 生成的 {locale} 动态资源翻译草稿。\n\n"
        "提交前应完成人工审核；迁移只使用已审核的固定文本，不在运行时调用翻译服务。\n"
    )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write", action="store_true", help="写入所有新增语言文件")
    parser.add_argument("--machine", action="store_true", help="使用 Google V1 生成韩法西草稿")
    parser.add_argument("--offline", action="store_true", help="使用内置术语表离线生成机器翻译草稿")
    parser.add_argument("--locale", dest="locales", action="append", help="只生成指定语言，可重复传入")
    parser.add_argument("--migration-version", help="将翻译 SQL 写入指定版本目录，例如 v0.0.3")
    args = parser.parse_args()
    locales = tuple(args.locales or DEFAULT_TARGET_LOCALES)
    sql_directory = SQL_DIR
    if args.migration_version:
        if not MIGRATION_VERSION_PATTERN.fullmatch(args.migration_version):
            raise SystemExit("迁移版本必须是 vX.Y.Z 格式")
        sql_directory = ROOT / "backend/migration/assets" / args.migration_version / "mysql"
        if args.write:
            sql_directory.mkdir(parents=True, exist_ok=True)
    converter = load_opencc() if "zh-TW" in locales else None
    for locale in locales:
        if locale != "zh-TW" and not args.machine and not args.offline:
            raise SystemExit("生成韩语、法语和西班牙语需要显式传入 --machine 或 --offline")
        for source in JSON_SOURCES:
            generate_json(source, locale, converter, args.machine, args.offline, args.write)
        generate_sql(locale, converter, args.machine, args.offline, args.write, sql_directory)
        if args.migration_version and args.write:
            (sql_directory / f"translation.{locale}.description.md").write_text(
                render_translation_description(locale), encoding="utf-8"
            )
    action = "已生成" if args.write else "可生成"
    print(f"{action} {', '.join(locales)} 语言包和迁移数据")
    return 0


if __name__ == "__main__":
    sys.exit(main())
