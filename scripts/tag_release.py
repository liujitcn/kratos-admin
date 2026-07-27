#!/usr/bin/env python3
"""统一升级根项目、后端模块和前端 npm 包的版本并推送 tag。"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PACKAGE_FILES = (
    ROOT / "frontend/admin/package.json",
    ROOT / "frontend/app/package.json",
)
TAG_RE = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")


def run(cmd: list[str], *, cwd: Path = ROOT, check: bool = True) -> str:
    """执行命令并返回标准输出。"""
    result = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True)
    if check and result.returncode != 0:
        detail = result.stderr.strip() or result.stdout.strip()
        raise RuntimeError(f"命令执行失败: {' '.join(cmd)}\n{detail}")
    return result.stdout.strip()


def run_stream(cmd: list[str], *, cwd: Path = ROOT) -> None:
    """执行命令并直接转发输出。"""
    result = subprocess.run(cmd, cwd=cwd)
    if result.returncode != 0:
        raise RuntimeError(f"命令执行失败: {' '.join(cmd)}")


def parse_version(value: str) -> tuple[int, int, int]:
    """解析 vX.Y.Z 格式的版本。"""
    raw = value.strip()
    if raw.startswith("v"):
        raw = raw[1:]
    match = TAG_RE.fullmatch(f"v{raw}")
    if not match:
        raise RuntimeError(f"非法版本号: {value}，应为 X.Y.Z 或 vX.Y.Z")
    return tuple(int(part) for part in match.groups())


def version_text(version: tuple[int, int, int]) -> str:
    """格式化 npm 版本号。"""
    return ".".join(str(part) for part in version)


def tag_text(version: tuple[int, int, int]) -> str:
    """格式化根目录 tag。"""
    return f"v{version_text(version)}"


def detect_remote_branch() -> str:
    """读取 origin 默认分支名称。"""
    remote_head = run(
        ["git", "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"],
        check=False,
    )
    if remote_head.startswith("origin/"):
        return remote_head.split("/", 1)[1]
    if remote_head:
        return remote_head

    symref = run(["git", "ls-remote", "--symref", "origin", "HEAD"], check=False)
    for line in symref.splitlines():
        if line.startswith("ref: refs/heads/") and line.endswith("\tHEAD"):
            return line.removeprefix("ref: refs/heads/").removesuffix("\tHEAD")
    return "main"


def fetch_remote(branch: str) -> None:
    """刷新远程分支和 tag。"""
    run(["git", "fetch", "origin", branch, "--tags", "--prune"])


def ensure_release_branch(branch: str) -> str:
    """确保当前分支是远程默认分支且已同步。"""
    current = run(["git", "branch", "--show-current"])
    if current != branch:
        raise RuntimeError(f"发布必须在远程默认分支 {branch} 上执行，当前分支为 {current or '<detached>'}")

    remote_ref = f"refs/remotes/origin/{branch}"
    remote_head = run(["git", "rev-parse", "--verify", remote_ref], check=False)
    if not remote_head:
        raise RuntimeError(f"远程分支不存在: origin/{branch}")
    local_head = run(["git", "rev-parse", "HEAD"])
    if local_head != remote_head:
        raise RuntimeError("当前分支未与 origin 同步，请先提交并推送代码后再发布")
    return remote_ref


def latest_tag(prefix: str) -> tuple[tuple[int, int, int], str] | None:
    """读取指定前缀下最大的语义化版本 tag。"""
    tags = run(["git", "tag", "--list", f"{prefix}v*"]).splitlines()
    parsed: list[tuple[tuple[int, int, int], str]] = []
    for tag in tags:
        if not tag.startswith(prefix):
            continue
        raw = tag[len(prefix) :]
        if not TAG_RE.fullmatch(raw):
            continue
        parsed.append((parse_version(raw), tag))
    if not parsed:
        return None
    parsed.sort(key=lambda item: item[0], reverse=True)
    return parsed[0]


def next_version(latest: tuple[int, int, int] | None) -> tuple[int, int, int]:
    """计算下一个 patch 版本。"""
    if latest is None:
        return 0, 0, 1
    major, minor, patch = latest
    patch += 1
    if patch >= 100:
        minor += 1
        patch = 0
    if minor >= 100:
        major += 1
        minor = 0
    return major, minor, patch


def resolve_version(requested: str | None) -> tuple[int, int, int]:
    """解析显式版本，未指定时按根目录最新 tag 自动递增。"""
    if requested:
        return parse_version(requested)

    root_latest = latest_tag("")
    backend_latest = latest_tag("backend/")
    if root_latest and backend_latest and root_latest[0] != backend_latest[0]:
        raise RuntimeError(
            f"根目录与 backend 最新 tag 不一致: {root_latest[1]} / {backend_latest[1]}，请显式指定 VERSION 修复版本线"
        )
    latest = root_latest[0] if root_latest else backend_latest[0] if backend_latest else None
    return next_version(latest)


def tag_exists_locally(tag: str) -> bool:
    """判断本地 tag 是否存在。"""
    return bool(run(["git", "rev-parse", "--verify", f"refs/tags/{tag}"], check=False))


def tag_exists_remotely(tag: str) -> bool:
    """判断远程 tag 是否存在。"""
    return subprocess.run(
        ["git", "ls-remote", "--exit-code", "--tags", "origin", f"refs/tags/{tag}"],
        cwd=ROOT,
        capture_output=True,
        text=True,
    ).returncode == 0


def ensure_tags_available(version: tuple[int, int, int]) -> tuple[str, str]:
    """确保根目录和 backend 的目标 tag 尚未使用。"""
    root_tag = tag_text(version)
    backend_tag = f"backend/{root_tag}"
    for tag in (root_tag, backend_tag):
        if tag_exists_locally(tag) or tag_exists_remotely(tag):
            raise RuntimeError(f"tag 已存在，拒绝覆盖: {tag}")
    return root_tag, backend_tag


def update_package_versions(version: tuple[int, int, int]) -> list[Path]:
    """将两个前端包统一更新到目标版本。"""
    target = version_text(version)
    changed: list[Path] = []
    for path in PACKAGE_FILES:
        try:
            package = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as err:
            raise RuntimeError(f"无法读取 npm 包配置: {path}: {err}") from err
        current = package.get("version")
        if not isinstance(current, str):
            raise RuntimeError(f"npm 包缺少合法 version: {path}")
        if parse_version(current) > version:
            raise RuntimeError(f"npm 包版本 {current} 高于目标版本 {target}: {path}")
        if current == target:
            continue
        package["version"] = target
        content = json.dumps(package, ensure_ascii=False, indent=2) + "\n"
        path.write_text(content, encoding="utf-8")
        changed.append(path)
    return changed


def commit_all_changes(version: tuple[int, int, int]) -> None:
    """提交版本更新以及工作区中的全部本地改动。"""
    status = run(["git", "status", "--porcelain"])
    if not status:
        return
    run(["git", "add", "-A"])
    run(["git", "commit", "-m", f"chore(release): 统一升级 v{version_text(version)}"])


def run_frontend_package() -> None:
    """检查并打包两个前端 npm 包。"""
    run_stream(["make", "-C", "frontend", "package"])


def push_branch(branch: str) -> None:
    """推送版本提交到默认分支。"""
    run(["git", "push", "origin", branch])


def push_tag(tag: str) -> None:
    """创建并推送单个 tag。"""
    run(["git", "tag", tag])
    run(["git", "push", "origin", tag])
    print(f"已推送 tag: {tag}")


def run_backend_tests() -> None:
    """发布前执行后端完整测试。"""
    run_stream(["go", "test", "./..."], cwd=ROOT / "backend")


def main() -> int:
    parser = argparse.ArgumentParser(description="统一发布根项目、backend tag 和 frontend npm 包")
    parser.add_argument("--version", help="目标版本，支持 X.Y.Z 或 vX.Y.Z；不指定时自动递增")
    parser.add_argument("--package", action="store_true", help="推送前检查并打包两个前端 npm 包")
    parser.add_argument("--dry-run", action="store_true", help="仅检查并打印目标，不修改文件、不提交、不推送")
    args = parser.parse_args()

    try:
        branch = detect_remote_branch()
        fetch_remote(branch)
        ensure_release_branch(branch)
        version = resolve_version(args.version)
        root_tag, backend_tag = ensure_tags_available(version)
        target = version_text(version)
        print(f"发布版本: {target}")
        print(f"发布顺序: 全量提交 -> {'frontend package -> ' if args.package else ''}推送分支 -> {root_tag} -> {backend_tag}")

        if args.dry_run:
            print("dry-run：未修改文件、未提交、未推送。")
            return 0

        changed = update_package_versions(version)
        run_backend_tests()
        commit_all_changes(version)
        if args.package:
            run_frontend_package()
        push_branch(branch)
        push_tag(root_tag)
        push_tag(backend_tag)
        return 0
    except RuntimeError as err:
        print(str(err), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
