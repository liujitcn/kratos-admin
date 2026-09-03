#!/usr/bin/env python3
"""收集项目 Markdown 文档，并生成独立的多语言目录文件。"""

from __future__ import annotations

import argparse
import copy
import html
import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath
from typing import Any, Iterator


DEFAULT_SOURCE_LOCALE = "zh-CN"
DEFAULT_OUTPUT_PATH = "internal/projectdocs"
BACKEND_OUTPUT_PATH = "backend/internal/docs"
I18N_ENDPOINT = "https://translate.googleapis.com/translate_a/single"
MYMEMORY_ENDPOINT = "https://api.mymemory.translated.net/get"
GOOGLE_CLIENTS = ("gtx", "dict-chrome-ex", "chrome", "at")
DEFAULT_I18N_BATCH_CHARS = 400
DEFAULT_I18N_REQUEST_DELAY = 0.2
DEFAULT_I18N_REQUEST_TIMEOUT = 8.0
MAX_SOURCE_PATH_DEPTH = 3
MAX_DOCUMENT_BYTES = 2 << 20
EXCLUDED_DIRECTORIES = {
    ".git",
    ".idea",
    ".turbo",
    ".vscode",
    "build",
    "data",
    "dist",
    "node_modules",
    "vendor",
}
LOCALE_PATTERN = re.compile(r"^[A-Za-z]{2,3}(?:[-_][A-Za-z0-9]{2,8})*$")
GO_IDENTIFIER_PATTERN = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")
ENTRY_PATTERN = re.compile(
    r"(?:__\s*)?KRATOS[_ ]?ENTRY[_ ]?(\d{4})(?:\s*__)?\s*:?\s*", re.IGNORECASE
)
TOKEN_PATTERN = re.compile(
    r"(?:__\s*)?KRATOS[_ ]?TOKEN[_ ]?(\d{4})(?:\s*__)?", re.IGNORECASE
)
MARKDOWN_LINK_PATTERN = re.compile(r"\[([^\]\r\n]*)\]\(([^)\r\n]+)\)")
TRANSLATION_MARKER_ARTIFACT_PATTERN = re.compile(
    r"(?<![A-Za-z])(?:KRATOSENTRY|ZZZMARKER)\s*(?:\s*(?:0*\d{1,4}|[:：.\-])){1,6}\s*[.。]?\s*",
    re.IGNORECASE,
)
TRANSLATION_RATE_LIMITED = False
LAST_I18N_REQUEST_AT = 0.0
PROTECTED_PATTERN = re.compile(
    r"(?s)```.*?```|`[^`]+`|\]\([^)]+\)|\{\{[^{}]+\}\}|\$\{[^{}]+\}|"
    r"\{[A-Za-z_][A-Za-z0-9_.-]*\}|%[sdv]|</?[^>]+>|"
    r"https?://[^\s<>()]+|(?<=\]\()[^)]+(?=\))|"
    r"/(?:api|events|mcp|v[0-9]+)/[A-Za-z0-9_./:{}-]+"
)
FENCE_PATTERN = re.compile(r"^\s*(```|~~~)")
DOCUMENT_NAME_I18N = {
    "AI助手设计.md": {"en-US": "AI Assistant Design.md", "zh-TW": "AI 助手設計.md", "ja-JP": "AI アシスタント設計.md"},
    "前端组件清单.md": {"en-US": "Frontend Component Catalog.md", "zh-TW": "前端元件清單.md", "ja-JP": "フロントエンドコンポーネント一覧.md"},
    "国际化最终方案.md": {"en-US": "Final Internationalization Design.md", "zh-TW": "國際化最終方案.md", "ja-JP": "国際化最終設計.md"},
    "国际化语言扩展指南.md": {"en-US": "Internationalization Language Extension Guide.md", "zh-TW": "國際化語言擴充指南.md", "ja-JP": "国際化言語拡張ガイド.md"},
    "接口参数校验设计.md": {"en-US": "API Parameter Validation Design.md", "zh-TW": "API 參數驗證設計.md", "ja-JP": "API パラメータ検証設計.md"},
    "数据库与初始化数据设计.md": {"en-US": "Database and Seed Data Design.md", "zh-TW": "資料庫與初始化資料設計.md", "ja-JP": "データベースと初期データ設計.md"},
    "服务接入指南.md": {"en-US": "Service Integration Guide.md", "zh-TW": "服務接入指南.md", "ja-JP": "サービス統合ガイド.md"},
    "登录与密码加密流程.md": {"en-US": "Login and Password Encryption Flow.md", "zh-TW": "登入與密碼加密流程.md", "ja-JP": "ログインとパスワード暗号化フロー.md"},
    "站内信设计.md": {"en-US": "Internal Message Design.md", "zh-TW": "站內信設計.md", "ja-JP": "站内信設計.md"},
    "系统总体设计.md": {"en-US": "System Architecture.md", "zh-TW": "系統總體設計.md", "ja-JP": "システム全体設計.md"},
    "后端服务设计.md": {"en-US": "Backend Service Design.md", "zh-TW": "後端服務設計.md", "ja-JP": "バックエンドサービス設計.md"},
    "商城业务范围.md": {"en-US": "Shop Business Scope.md", "zh-TW": "商城業務範圍.md", "ja-JP": "ショップ業務範囲.md"},
    "商城端设计.md": {"en-US": "Storefront Design.md", "zh-TW": "商城端設計.md", "ja-JP": "ストアフロント設計.md"},
    "推荐系统设计.md": {"en-US": "Recommendation System Design.md", "zh-TW": "推薦系統設計.md", "ja-JP": "レコメンドシステム設計.md"},
    "租户与门店体系设计.md": {"en-US": "Tenant and Store Model Design.md", "zh-TW": "租戶與門店體系設計.md", "ja-JP": "テナント・店舗体系設計.md"},
    "管理后台设计.md": {"en-US": "Admin Console Design.md", "zh-TW": "管理後臺設計.md", "ja-JP": "管理画面設計.md"},
    "统计数据流转设计.md": {"en-US": "Analytics Data Flow Design.md", "zh-TW": "統計資料流轉設計.md", "ja-JP": "統計データフロー設計.md"},
    "订单数据流转设计.md": {"en-US": "Order Data Flow Design.md", "zh-TW": "訂單資料流轉設計.md", "ja-JP": "注文データフロー設計.md"},
    "评价与审核数据流转设计.md": {"en-US": "Review and Moderation Data Flow Design.md", "zh-TW": "評價與審核資料流轉設計.md", "ja-JP": "レビュー・審査データフロー設計.md"},
}
CONTENT_LITERAL_I18N = {
    '"语言=路径"': {"en-US": '"locale=path"', "zh-TW": '"語言=路徑"', "ja-JP": '"言語=パス"'},
}


@dataclass(frozen=True)
class SourceDocument:
    """保存扫描到的源 Markdown 文档。"""

    stable_path: str
    source_path: Path
    content: str
    updated_at: str
    locale: str = ""


def normalize_locale(value: str) -> str:
    """规范化语言代码，用于比较而不改变输出键的大小写。"""
    return value.strip().replace("_", "-").lower()


def canonical_locale(value: str) -> str:
    """将语言代码规范为适合文件名的 BCP 47 常用大小写。"""
    parts = value.replace("_", "-").split("-")
    if not parts or not parts[0]:
        return value
    result = [parts[0].lower()]
    for part in parts[1:]:
        if len(part) == 2 and part.isalpha():
            result.append(part.upper())
        elif len(part) == 4 and part.isalpha():
            result.append(part.title())
        else:
            result.append(part)
    return "-".join(result)


def parse_locales(value: str) -> list[str]:
    """解析逗号分隔的 BCP 47 语言代码列表。"""
    result: list[str] = []
    seen: set[str] = set()
    for item in value.split(","):
        locale = canonical_locale(item.strip())
        if not locale:
            continue
        if not LOCALE_PATTERN.fullmatch(locale):
            raise ValueError(f"目标语言不是有效的 BCP 47 代码: {locale}")
        normalized = normalize_locale(locale)
        if normalized not in seen:
            seen.add(normalized)
            result.append(locale)
    return sorted(result, key=normalize_locale)


def split_locale_suffix(document_path: str) -> tuple[str, str]:
    """从 Markdown 文件名提取语言后缀并返回稳定文档路径。"""
    path = PurePosixPath(document_path)
    if path.suffix.lower() != ".md":
        return document_path, ""
    stem = path.stem
    separator = stem.rfind(".")
    if separator <= 0:
        return document_path, ""
    locale = stem[separator + 1 :].replace("_", "-")
    if not LOCALE_PATTERN.fullmatch(locale):
        return document_path, ""
    return str(path.with_name(f"{stem[:separator]}.md")), canonical_locale(locale)


def should_collect(document_path: str) -> bool:
    """判断三段路径范围内的文件是否为 README 或 docs Markdown。"""
    path = PurePosixPath(document_path)
    if len(path.parts) > MAX_SOURCE_PATH_DEPTH:
        return False
    stable_path, _ = split_locale_suffix(document_path)
    if PurePosixPath(stable_path).name == "README.md":
        return True
    return path.suffix.lower() == ".md" and "docs" in path.parts[:-1]


def format_updated_at(timestamp: float) -> str:
    """将文件修改时间格式化为秒级 UTC RFC3339。"""
    value = datetime.fromtimestamp(timestamp, timezone.utc).replace(microsecond=0)
    return value.isoformat().replace("+00:00", "Z")


def read_source_document(root: Path, source_path: Path) -> SourceDocument:
    """读取并校验一篇默认或本地化 Markdown 源文件。"""
    relative_path = source_path.relative_to(root).as_posix()
    stable_path, locale = split_locale_suffix(relative_path)
    stat = source_path.stat()
    if stat.st_size > MAX_DOCUMENT_BYTES:
        raise ValueError(f"文档超过 2 MiB: {relative_path}")
    try:
        content = source_path.read_text(encoding="utf-8")
    except UnicodeDecodeError as error:
        raise ValueError(f"文档不是有效 UTF-8: {relative_path}") from error
    return SourceDocument(
        stable_path=stable_path,
        source_path=source_path,
        content=content,
        updated_at=format_updated_at(stat.st_mtime),
        locale=locale,
    )


def scan_source(
    root: Path,
) -> tuple[list[SourceDocument], dict[str, dict[str, SourceDocument]]]:
    """按 project-docs 约定扫描默认文档和语言后缀文档。"""
    if not root.is_dir():
        raise ValueError(f"项目根目录不是目录: {root}")
    documents: dict[str, SourceDocument] = {}
    localized: dict[str, dict[str, SourceDocument]] = {}
    for current_root, directory_names, file_names in os.walk(root):
        current_path = Path(current_root)
        relative_directory = current_path.relative_to(root)
        depth = 0 if relative_directory == Path(".") else len(relative_directory.parts)
        directory_names[:] = sorted(
            name
            for name in directory_names
            if name not in EXCLUDED_DIRECTORIES and depth + 1 < MAX_SOURCE_PATH_DEPTH
        )
        for file_name in sorted(file_names):
            source_path = current_path / file_name
            relative_path = source_path.relative_to(root).as_posix()
            if not should_collect(relative_path):
                continue
            document = read_source_document(root, source_path)
            if document.locale:
                locale_key = normalize_locale(document.locale)
                by_locale = localized.setdefault(document.stable_path, {})
                if locale_key in by_locale:
                    raise ValueError(
                        f"项目文档语言源文件重复: {document.stable_path} ({document.locale})"
                    )
                by_locale[locale_key] = document
                continue
            if document.stable_path in documents:
                raise ValueError(f"项目文档路径重复: {document.stable_path}")
            documents[document.stable_path] = document
    return [documents[path] for path in sorted(documents)], localized


def build_catalog(documents: list[SourceDocument]) -> dict[str, Any]:
    """按稳定路径把默认文档构造成递归目录树。"""
    root: dict[str, Any] = {"documents": [], "directories": {}}
    for document in documents:
        path = PurePosixPath(document.stable_path)
        current = root
        directory_path = PurePosixPath()
        for directory_name in path.parts[:-1]:
            directory_path /= directory_name
            directories = current["directories"]
            current = directories.setdefault(
                directory_name,
                {
                    "name": directory_name,
                    "path": str(directory_path),
                    "documents": [],
                    "directories": {},
                },
            )
        current["documents"].append(
            {
                "path": document.stable_path,
                "name": path.name,
                "content": document.content,
                "updated_at": document.updated_at,
            }
        )

    def finalize(node: dict[str, Any]) -> dict[str, Any]:
        directories = node["directories"]
        node["directories"] = [finalize(directories[name]) for name in sorted(directories)]
        return node

    return finalize(root)


def protect_text(value: str) -> tuple[str, dict[str, str]]:
    """保护 Markdown 代码、链接和占位符，避免机器翻译破坏格式。"""
    values: dict[str, str] = {}

    def replace(match: re.Match[str]) -> str:
        token = f"KRATOSTOKEN{len(values):04d}"
        values[token] = match.group(0)
        return token

    return PROTECTED_PATTERN.sub(replace, value), values


def restore_text(value: str, protected: dict[str, str]) -> str:
    """恢复 Markdown 中被保护的原始片段。"""
    for token, original in protected.items():
        value = value.replace(token, original)
        match = TOKEN_PATTERN.fullmatch(token)
        if match:
            loose_token = re.compile(
                rf"(?:__)?\s*KRATOS[_ ]?TOKEN\s*{match.group(1)}\s*(?:__)?", re.IGNORECASE
            )
            value = loose_token.sub(original, value)
    return re.sub(r"\]\s+\(", "](", value)


def remove_translation_marker_artifacts(value: str) -> str:
    """移除翻译服务意外保留的内部批次行标记。"""
    return TRANSLATION_MARKER_ARTIFACT_PATTERN.sub("", value)


def repair_markdown_links(source: str, translated: str) -> str:
    """恢复翻译服务改写的 Markdown 链接结构并保留稳定目标路径。"""
    source_links = MARKDOWN_LINK_PATTERN.findall(source)
    if not source_links:
        return translated

    repaired = translated
    search_start = 0
    for source_label, destination in source_links:
        destination_start = repaired.find(destination, search_start)
        if destination_start < 0:
            continue
        suffix_start = repaired.rfind("](", search_start, destination_start)
        line_start = repaired.rfind("\n", 0, destination_start) + 1
        link_start = repaired.rfind("[", line_start, suffix_start)
        destination_end = destination_start + len(destination)
        while destination_end < len(repaired) and repaired[destination_end] in ")]":
            destination_end += 1
        if suffix_start >= line_start and link_start >= line_start:
            label = repaired[link_start + 1 : suffix_start]
            replacement = f"[{label}]({destination})"
            repaired = repaired[:link_start] + replacement + repaired[destination_end:]
            search_start = link_start + len(replacement)
            continue
        if suffix_start >= line_start:
            replacement = f"[{source_label}]({destination})"
            repaired = repaired[:suffix_start] + replacement + repaired[destination_end:]
            search_start = suffix_start + len(replacement)
            continue
        replacement = f"[{source_label}]({destination})"
        repaired = repaired[:destination_start] + replacement + repaired[destination_end:]
        search_start = destination_start + len(replacement)
    for source_label, destination in source_links:
        if destination in repaired:
            continue
        line_ending = ""
        if repaired.endswith("\r\n"):
            line_ending = "\r\n"
            repaired = repaired[:-2]
        elif repaired.endswith("\n"):
            line_ending = "\n"
            repaired = repaired[:-1]
        repaired += f" [{source_label}]({destination})" + line_ending
    return re.sub(r"\]\(([^)\r\n]+)\)\]", r"](\1)", repaired)


def repair_markdown_document(source: str, translated: str) -> str:
    """按源文档逐行恢复链接结构，避免跨段误匹配。"""
    source_lines = source.splitlines(keepends=True)
    translated_lines = translated.splitlines(keepends=True)
    if len(source_lines) != len(translated_lines):
        return repair_markdown_links(source, translated)
    return "".join(
        repair_markdown_links(source_line, translated_line)
        for source_line, translated_line in zip(source_lines, translated_lines)
    )


def i18n_request_delay() -> float:
    """读取翻译请求之间的最小间隔。"""
    try:
        return max(0.0, float(os.environ.get("I18N_REQUEST_DELAY", DEFAULT_I18N_REQUEST_DELAY)))
    except ValueError:
        return DEFAULT_I18N_REQUEST_DELAY


def i18n_request_timeout() -> float:
    """读取单次翻译请求的超时时间。"""
    try:
        return max(1.0, float(os.environ.get("I18N_REQUEST_TIMEOUT", DEFAULT_I18N_REQUEST_TIMEOUT)))
    except ValueError:
        return DEFAULT_I18N_REQUEST_TIMEOUT


def wait_for_i18n_request() -> None:
    """限制连续请求频率，降低公共翻译接口触发限流的概率。"""
    global LAST_I18N_REQUEST_AT
    now = time.monotonic()
    wait = i18n_request_delay() - (now - LAST_I18N_REQUEST_AT)
    if wait > 0:
        time.sleep(wait)
    LAST_I18N_REQUEST_AT = time.monotonic()


def request_google_i18n(value: str, source: str, target: str) -> str:
    """调用 Google V1 接口翻译文档自然语言。"""
    endpoint = os.environ.get("I18N_ENDPOINT", I18N_ENDPOINT)
    last_error: Exception | None = None
    clients = tuple(
        item.strip()
        for item in os.environ.get("I18N_GOOGLE_CLIENTS", ",".join(GOOGLE_CLIENTS)).split(",")
        if item.strip()
    )
    for client in clients:
        query = urllib.parse.urlencode(
            [
                ("client", client),
                ("sl", source.split("-", 1)[0]),
                ("tl", target.split("-", 1)[0]),
                ("dt", "t"),
                ("q", value),
            ]
        )
        for attempt in range(2):
            try:
                wait_for_i18n_request()
                request = urllib.request.Request(
                    f"{endpoint}{'&' if '?' in endpoint else '?'}{query}",
                    headers={"User-Agent": "kratos-admin-i18n/1.0", "Connection": "close"},
                )
                with urllib.request.urlopen(request, timeout=i18n_request_timeout()) as response:
                    payload = json.loads(response.read().decode("utf-8"))
                return "".join(item[0] for item in payload[0] if item and item[0])
            except urllib.error.HTTPError as error:
                last_error = error
                if error.code in (403, 429):
                    break
            except Exception as error:  # noqa: BLE001 - 翻译服务失败需要统一重试
                last_error = error
            if attempt < 1:
                time.sleep(0.5)
    raise RuntimeError(f"Google V1 翻译失败: {last_error}")


def request_mymemory_i18n(value: str, source: str, target: str) -> str:
    """调用 MyMemory 公共接口翻译较小的文档分片。"""
    query = urllib.parse.urlencode(
        {
            "q": value,
            "langpair": f"{source.split('-', 1)[0]}|{target.split('-', 1)[0]}",
            "mt": "1",
        }
    )
    endpoint = os.environ.get("I18N_FALLBACK_ENDPOINT", MYMEMORY_ENDPOINT)
    last_error: Exception | None = None
    for attempt in range(3):
        try:
            wait_for_i18n_request()
            request = urllib.request.Request(
                f"{endpoint}{'&' if '?' in endpoint else '?'}{query}",
                headers={"User-Agent": "kratos-admin-i18n/1.0", "Connection": "close"},
            )
            with urllib.request.urlopen(request, timeout=i18n_request_timeout()) as response:
                payload = json.loads(response.read().decode("utf-8"))
            status = payload.get("responseStatus")
            translated = payload.get("responseData", {}).get("translatedText")
            if status != 200 or not isinstance(translated, str) or not translated:
                details = payload.get("responseDetails") or f"status={status}"
                raise RuntimeError(f"MyMemory 翻译失败: {details}")
            return html.unescape(translated)
        except urllib.error.HTTPError as error:
            if error.code == 429:
                raise RuntimeError("MyMemory 翻译接口返回 HTTP 429 Too Many Requests") from error
            last_error = error
            if attempt < 2:
                time.sleep(0.5 * (attempt + 1))
        except Exception as error:  # noqa: BLE001 - 翻译服务失败需要统一重试
            last_error = error
            if attempt < 2:
                time.sleep(0.5 * (attempt + 1))
    raise RuntimeError(f"MyMemory 翻译失败: {last_error}")


def request_i18n(value: str, source: str, target: str) -> str:
    """优先调用 Google，触发限流后切换 MyMemory。"""
    global TRANSLATION_RATE_LIMITED
    provider = os.environ.get("I18N_TRANSLATOR", "auto").lower()
    if provider == "mymemory" or TRANSLATION_RATE_LIMITED:
        return request_mymemory_i18n(value, source, target)
    try:
        return request_google_i18n(value, source, target)
    except RuntimeError as error:
        TRANSLATION_RATE_LIMITED = True
        print(f"Google V1 不可用，后续分片切换 MyMemory 翻译: {error}", file=sys.stderr)
        return request_mymemory_i18n(value, source, target)


def load_opencc() -> Any:
    """按需加载繁体中文转换器；未安装时返回 None。"""
    try:
        from opencc import OpenCC
    except ImportError:
        return None
    return OpenCC("s2twp")


def i18n_markdown_with_status(
    value: str,
    source: str,
    target: str,
    offline: bool,
    batch_chars: int = DEFAULT_I18N_BATCH_CHARS,
) -> tuple[str, bool]:
    """翻译 Markdown 自然语言，并保留代码块、链接和占位符。"""
    if normalize_locale(target) == "zh-tw":
        converter = load_opencc()
        if converter is not None:
            protected, protected_values = protect_text(value)
            return restore_text(converter.convert(protected), protected_values), True
    if offline:
        return value, True

    lines = value.splitlines(keepends=True)
    line_order: list[int] = []
    direct_lines: dict[int, str] = {}
    translated_parts: dict[int, list[str]] = {}
    line_endings: dict[int, str] = {}
    pending: list[tuple[int, str, str, str, dict[str, str]]] = []
    pending_size = 0
    marker_prefix_size = len("KRATOSENTRY0000: ")
    content_batch_chars = max(1, batch_chars - marker_prefix_size)
    in_fence = False
    fence_character = ""
    i18n_succeeded = True

    def split_protected_text(text: str) -> list[str]:
        """按批次上限拆分文本，并避免切断受保护的 Markdown 令牌。"""
        if len(text) <= content_batch_chars:
            return [text]
        parts: list[str] = []
        start = 0
        while start < len(text):
            end = min(start + content_batch_chars, len(text))
            if end < len(text):
                token = next(
                    (
                        match
                        for match in TOKEN_PATTERN.finditer(text, max(start, end - 32))
                        if match.start() < end < match.end()
                    ),
                    None,
                )
                if token is not None:
                    end = token.start()
                if end <= start:
                    end = min(start + content_batch_chars, len(text))
                else:
                    separator = text.rfind(" ", start + content_batch_chars // 2, end)
                    if separator > start:
                        end = separator + 1
            parts.append(text[start:end])
            start = end
        return parts

    def flush() -> None:
        nonlocal pending, pending_size, i18n_succeeded
        if not pending:
            return
        source_text = "\n".join(
            f"KRATOSENTRY{index:04d}: {protected}"
            for index, (_, _, _, protected, _) in enumerate(pending)
        )
        translated = ""
        try:
            translated = request_i18n(source_text, source, target)
        except RuntimeError as error:
            if i18n_succeeded:
                print(
                    f"项目文档翻译批次失败，保留原文并等待下次重试: {error}",
                    file=sys.stderr,
                )
            i18n_succeeded = False
        translated_by_index: dict[int, str] = {}
        matches = list(ENTRY_PATTERN.finditer(translated))
        for position, match in enumerate(matches):
            end = matches[position + 1].start() if position + 1 < len(matches) else len(translated)
            translated_by_index[int(match.group(1))] = translated[match.end() : end].strip()
        expected_indexes = set(range(len(pending)))
        if set(translated_by_index) != expected_indexes:
            if i18n_succeeded:
                print(
                    "项目文档翻译批次缺少行标记，保留原文并等待下次重试",
                    file=sys.stderr,
                )
            i18n_succeeded = False
        for marker_index, (line_index, original, ending, _, protected_values) in enumerate(pending):
            translated_line = translated_by_index.get(marker_index, "")
            translated_parts.setdefault(line_index, []).append(
                repair_markdown_links(
                    original,
                    restore_text(translated_line, protected_values) if translated_line else original,
                )
            )
            line_endings[line_index] = ending
        pending = []
        pending_size = 0

    for line_index, line in enumerate(lines):
        line_order.append(line_index)
        body = line.rstrip("\r\n")
        ending = line[len(body) :]
        stripped = body.strip()
        fence_match = FENCE_PATTERN.match(stripped)
        if in_fence:
            direct_lines[line_index] = line
            if fence_match and fence_match.group(1)[0] == fence_character:
                in_fence = False
                fence_character = ""
            continue
        if fence_match:
            flush()
            direct_lines[line_index] = line
            in_fence = True
            fence_character = fence_match.group(1)[0]
            continue
        if not any(character.isalpha() or character.isdigit() for character in stripped):
            flush()
            direct_lines[line_index] = line
            continue
        protected, protected_values = protect_text(body)
        parts = split_protected_text(protected)
        for part_index, part in enumerate(parts):
            entry_size = len(part) + marker_prefix_size + (1 if pending else 0)
            if pending and pending_size + entry_size > batch_chars:
                flush()
            part_original = restore_text(part, protected_values)
            part_ending = ending if part_index == len(parts) - 1 else ""
            pending.append((line_index, part_original, part_ending, part, protected_values))
            pending_size += entry_size
    flush()
    return "".join(
        direct_lines.get(
            line_index,
            "".join(translated_parts.get(line_index, [])) + line_endings.get(line_index, ""),
        )
        for line_index in line_order
    ), i18n_succeeded


def i18n_document_name_with_status(
    value: str,
    source: str,
    target: str,
    offline: bool,
    batch_chars: int = DEFAULT_I18N_BATCH_CHARS,
) -> tuple[str, bool]:
    """翻译非 README 文档显示名，并保留 Markdown 扩展名。"""
    if value == "README.md":
        return value, True
    path = PurePosixPath(value)
    translated, succeeded = i18n_markdown_with_status(path.stem, source, target, offline, batch_chars)
    return f"{translated.strip() or path.stem}{path.suffix}", succeeded


def localized_document_name(value: str, target_locale: str) -> str:
    """返回稳定词典中的本地化文档名。"""
    return DOCUMENT_NAME_I18N.get(value, {}).get(canonical_locale(target_locale), "")


def localize_content_literals(value: str, target_locale: str) -> str:
    """替换机器翻译刻意保护、但需要随语言变化的示例字面量。"""
    locale = canonical_locale(target_locale)
    for source, translations in CONTENT_LITERAL_I18N.items():
        target = translations.get(locale)
        if target:
            value = value.replace(source, target)
    return value


def iter_documents(node: dict[str, Any]) -> Iterator[dict[str, Any]]:
    """递归遍历目录中的全部文档节点。"""
    documents = node.get("documents", [])
    if not isinstance(documents, list):
        raise ValueError("项目文档目录的 documents 必须是数组")
    for document in documents:
        if not isinstance(document, dict):
            raise ValueError("项目文档目录包含无效文档节点")
        yield document
    directories = node.get("directories", [])
    if not isinstance(directories, list):
        raise ValueError("项目文档目录的 directories 必须是数组")
    for directory in directories:
        if not isinstance(directory, dict):
            raise ValueError("项目文档目录包含无效目录节点")
        yield from iter_documents(directory)


def iter_directories(node: dict[str, Any]) -> Iterator[dict[str, Any]]:
    """递归遍历项目文档目录节点。"""
    directories = node.get("directories", [])
    if not isinstance(directories, list):
        raise ValueError("项目文档目录的 directories 必须是数组")
    for directory in directories:
        if not isinstance(directory, dict):
            raise ValueError("项目文档目录包含无效目录节点")
        yield directory
        yield from iter_directories(directory)


def load_catalog(catalog_path: Path) -> dict[str, Any]:
    """读取项目文档目录。"""
    catalog = json.loads(catalog_path.read_text(encoding="utf-8"))
    if not isinstance(catalog, dict):
        raise ValueError("项目文档目录根节点必须是对象")
    return catalog


def prepare_catalog(catalog: dict[str, Any]) -> dict[str, dict[str, str]]:
    """校验目录并提取旧格式 locale，供首次拆分时复用。"""
    legacy: dict[str, dict[str, str]] = {}
    paths: set[str] = set()
    for document in iter_documents(catalog):
        document_path = document.get("path")
        content = document.get("content")
        if not isinstance(document_path, str) or not document_path:
            raise ValueError("项目文档 path 必须是非空字符串")
        if document_path in paths:
            raise ValueError(f"项目文档路径重复: {document_path}")
        paths.add(document_path)
        if not isinstance(content, str):
            raise ValueError(f"项目文档 content 必须是字符串: {document_path}")
        name = document.get("name")
        if name is not None and (not isinstance(name, str) or not name):
            raise ValueError(f"项目文档 name 必须是非空字符串: {document_path}")
        localized_value = document.pop("locale", None)
        if localized_value is None:
            continue
        if not isinstance(localized_value, dict):
            raise ValueError(f"项目文档 locale 必须是对象: {document_path}")
        localized: dict[str, str] = {}
        for locale, translated in localized_value.items():
            if not isinstance(locale, str) or not isinstance(translated, str):
                raise ValueError(f"项目文档 locale 必须使用字符串键值: {document_path}")
            locale_key = normalize_locale(locale)
            if not locale_key or locale_key in localized:
                raise ValueError(f"项目文档 locale 存在重复语言代码: {document_path} ({locale})")
            localized[locale_key] = translated
        legacy[document_path] = localized
    directory_paths: set[str] = set()
    for directory in iter_directories(catalog):
        directory_path = directory.get("path")
        name = directory.get("name")
        if not isinstance(directory_path, str) or not directory_path:
            raise ValueError("项目文档目录 path 必须是非空字符串")
        if directory_path in directory_paths:
            raise ValueError(f"项目文档目录路径重复: {directory_path}")
        directory_paths.add(directory_path)
        if not isinstance(name, str) or not name:
            raise ValueError(f"项目文档目录 name 必须是非空字符串: {directory_path}")
    return legacy


def index_documents(catalog: dict[str, Any] | None) -> dict[str, dict[str, Any]]:
    """按稳定文档路径索引目录中的文档节点。"""
    if catalog is None:
        return {}
    return {document["path"]: document for document in iter_documents(catalog)}


def find_localized_source(
    sources: dict[str, SourceDocument], target_locale: str
) -> SourceDocument | None:
    """按完整语言、基础语言顺序选择对应的语言源文件。"""
    target_key = normalize_locale(target_locale)
    candidates = [target_key]
    if "-" in target_key:
        candidates.append(target_key.split("-", 1)[0])
    for candidate in candidates:
        if candidate in sources:
            return sources[candidate]
    return None


def i18n_catalog(
    source: dict[str, Any],
    existing: dict[str, Any] | None,
    localized_sources: dict[str, dict[str, SourceDocument]],
    legacy: dict[str, dict[str, str]],
    source_locale: str,
    target_locale: str,
    offline: bool,
    batch_chars: int,
) -> tuple[dict[str, Any], int, int]:
    """生成单个语言目录，并优先复用仓库内已有语言文档。"""
    localized = copy.deepcopy(source)
    source_documents = index_documents(source)
    localized_documents = index_documents(localized)
    existing_documents = index_documents(existing)
    changed = 0
    reused_sources = 0
    target_key = normalize_locale(target_locale)
    for document_path, source_document in source_documents.items():
        source_content = source_document["content"]
        source_name = source_document["name"]
        output_document = localized_documents[document_path]
        existing_document = existing_documents.get(document_path)
        translated_content = ""
        localized_source = find_localized_source(
            localized_sources.get(document_path, {}), target_locale
        )
        if localized_source is not None:
            translated_content = localized_source.content
            reused_sources += 1
        elif existing_document is not None:
            existing_content = existing_document.get("content")
            if (
                isinstance(existing_content, str)
                and existing_content
                and existing_document.get("updated_at") == source_document.get("updated_at")
                and (offline or existing_content != source_content)
            ):
                translated_content = existing_content
        if not translated_content:
            legacy_content = legacy.get(document_path, {}).get(target_key, "")
            if legacy_content and (offline or legacy_content != source_content):
                translated_content = legacy_content
        if not translated_content and source_content:
            candidate, succeeded = i18n_markdown_with_status(
                source_content, source_locale, target_locale, offline, batch_chars
            )
            if candidate and (succeeded or candidate != source_content):
                translated_content = candidate
                if succeeded:
                    changed += 1
            elif existing_document is not None:
                previous_content = existing_document.get("content")
                if isinstance(previous_content, str) and previous_content:
                    translated_content = previous_content
        output_document["content"] = repair_markdown_document(
            source_content,
            remove_translation_marker_artifacts(
                localize_content_literals(translated_content or source_content, target_locale)
            ),
        )
        output_document.pop("locale", None)

        translated_name = ""
        if source_name == "README.md":
            translated_name = source_name
        else:
            translated_name = localized_document_name(source_name, target_locale)
        if not translated_name and existing_document is not None:
            existing_name = existing_document.get("name")
            if (
                isinstance(existing_name, str)
                and existing_name
                and existing_document.get("updated_at") == source_document.get("updated_at")
                and (offline or existing_name != source_name)
            ):
                translated_name = existing_name
        if not translated_name:
            candidate, succeeded = i18n_document_name_with_status(
                source_name, source_locale, target_locale, offline, batch_chars
            )
            if succeeded:
                translated_name = candidate
                changed += 1
        output_document["name"] = translated_name or source_name
    return localized, changed, reused_sources


def write_file_if_changed(output_path: Path, data: str) -> bool:
    """仅在内容变化时原子替换生成文件。"""
    if output_path.exists() and output_path.read_text(encoding="utf-8") == data:
        return False
    output_path.parent.mkdir(parents=True, exist_ok=True)
    temporary_path = output_path.with_name(f".{output_path.name}.{os.getpid()}.tmp")
    try:
        temporary_path.write_text(data, encoding="utf-8")
        temporary_path.chmod(0o644)
        os.replace(temporary_path, output_path)
    finally:
        try:
            temporary_path.unlink()
        except FileNotFoundError:
            pass
    return True


def write_catalog(catalog_path: Path, catalog: dict[str, Any]) -> bool:
    """以稳定 JSON 格式写入项目文档目录。"""
    return write_file_if_changed(
        catalog_path, json.dumps(catalog, ensure_ascii=False, indent=2) + "\n"
    )


def generate_go_source(output_dir: Path, catalog_path: Path) -> str:
    """生成嵌入默认和多语言目录的 Go 源文件。"""
    package_name = output_dir.name
    if not GO_IDENTIFIER_PATTERN.fullmatch(package_name):
        raise ValueError(f"Go 生成目录名不是有效包名: {package_name}")
    try:
        embed_path = catalog_path.relative_to(output_dir).as_posix()
    except ValueError as error:
        raise ValueError(f"项目文档目录必须位于 Go 生成文件目录内: {catalog_path}") from error
    embed_pattern = f"{PurePosixPath(embed_path).parent}/docs*.json"
    return f'''// Code generated by project_docs.py. DO NOT EDIT.

package {package_name}

import "embed"

// DocsFS 包含默认语言和各语言后缀的项目文档目录。
//
//go:embed "{embed_pattern}"
var DocsFS embed.FS
'''


def remove_stale_catalogs(catalog_path: Path, expected_names: set[str]) -> list[str]:
    """删除目标语言列表之外且符合约定命名的旧语言目录。"""
    removed: list[str] = []
    prefix = "docs."
    suffix = ".json"
    for path in catalog_path.parent.glob("docs.*.json"):
        locale = path.name[len(prefix) : -len(suffix)]
        if path.name in expected_names or not LOCALE_PATTERN.fullmatch(locale):
            continue
        path.unlink()
        removed.append(path.name)
    return sorted(removed)


def resolve_output(root: Path, value: str) -> Path:
    """解析输出目录，未指定时按项目结构选择默认位置。"""
    if value:
        path = Path(value)
        return path.resolve() if path.is_absolute() else (root / path).resolve()
    return root / (BACKEND_OUTPUT_PATH if (root / "backend").is_dir() else DEFAULT_OUTPUT_PATH)


def main() -> int:
    """先收集默认和现有语言文档，再生成全部目录文件。"""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=".", help="项目根目录")
    parser.add_argument("--output", "-o", default="", help="项目文档生成目录")
    parser.add_argument("--source-locale", default=DEFAULT_SOURCE_LOCALE, help="源语言")
    parser.add_argument("--locales", default="", help="逗号分隔的目标语言列表")
    parser.add_argument("--offline", action="store_true", help="跳过网络翻译")
    parser.add_argument(
        "--batch-chars",
        type=int,
        default=DEFAULT_I18N_BATCH_CHARS,
        help="单次翻译请求的最大字符数",
    )
    args = parser.parse_args()
    if args.batch_chars < 1:
        parser.error("--batch-chars 必须大于 0")
    try:
        locales = parse_locales(args.locales)
    except ValueError as error:
        parser.error(str(error))

    root = Path(args.root).resolve()
    output_dir = resolve_output(root, args.output)
    catalog_path = output_dir / "assets" / "docs.json"
    go_output_path = output_dir / "docs.go"
    try:
        existing_source = load_catalog(catalog_path) if catalog_path.exists() else None
        legacy = prepare_catalog(existing_source) if existing_source is not None else {}
        documents, localized_sources = scan_source(root)
        source_catalog = build_catalog(documents)
        write_catalog(catalog_path, source_catalog)
        write_file_if_changed(go_output_path, generate_go_source(output_dir, catalog_path))
    except (OSError, ValueError, json.JSONDecodeError) as error:
        print(f"收集项目文档失败: {error}", file=sys.stderr)
        return 1

    generated: list[Path] = []
    changed = 0
    reused_sources = 0
    removed: list[str] = []
    try:
        source_locale = normalize_locale(args.source_locale)
        expected_names = {
            f"docs.{locale}.json"
            for locale in locales
            if normalize_locale(locale) != source_locale
        }
        for locale in locales:
            if normalize_locale(locale) == source_locale:
                continue
            localized_path = catalog_path.with_name(f"docs.{locale}.json")
            existing = load_catalog(localized_path) if localized_path.exists() else None
            if existing is not None:
                prepare_catalog(existing)
            localized, locale_changed, locale_reused_sources = i18n_catalog(
                source_catalog,
                existing,
                localized_sources,
                legacy,
                args.source_locale,
                locale,
                args.offline,
                args.batch_chars,
            )
            write_catalog(localized_path, localized)
            generated.append(localized_path)
            changed += locale_changed
            reused_sources += locale_reused_sources
        if locales:
            removed = remove_stale_catalogs(catalog_path, expected_names)
    except (OSError, ValueError, json.JSONDecodeError) as error:
        print(f"生成项目文档语言目录失败: {error}", file=sys.stderr)
        return 1

    print(f"已收集 {len(documents)} 篇项目文档到 {catalog_path}，并生成 {go_output_path}")
    if generated:
        generated_names = ", ".join(path.name for path in generated)
        print(
            f"已生成 {generated_names}，复用 {reused_sources} 篇语言源文档，"
            f"补充或刷新 {changed} 个本地化字段"
        )
    if removed:
        print(f"已删除过期项目文档语言文件: {', '.join(removed)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
