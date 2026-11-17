#!/usr/bin/env python3
"""先发布 backend/api，再统一升级后端模块和前端 npm 包并推送 tag。"""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PROJECT_VERSION_FILE = ROOT / "backend/internal/const/project.go"
API_MODULE = "github.com/liujitcn/kratos-admin/backend/api"
API_DEPENDENCY_FILES = (
    ROOT / "backend/go.mod",
    ROOT / "backend/client/go.mod",
)
PACKAGE_FILES = (
    ROOT / "frontend/admin/packages/core/package.json",
    ROOT / "frontend/admin/packages/modules/system/package.json",
    ROOT / "frontend/admin/packages/modules/rag/package.json",
    ROOT / "frontend/admin/packages/modules/bi/package.json",
    ROOT / "frontend/admin/packages/cli/package.json",
    ROOT / "frontend/uni-app/packages/core/package.json",
    ROOT / "frontend/uni-app/packages/modules/system/package.json",
    ROOT / "frontend/uni-app/packages/cli/package.json",
    ROOT / "frontend/taro-app/packages/core/package.json",
    ROOT / "frontend/taro-app/packages/ui/package.json",
    ROOT / "frontend/taro-app/packages/modules/system/package.json",
    ROOT / "frontend/taro-app/packages/cli/package.json",
)
TAG_RE = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")
PROJECT_VERSION_RE = re.compile(r'(?m)^(\s*Version\s*=\s*")[^"]+("\s*)$')
API_MODULE_VERSION_RE = re.compile(
    rf"(?m)^([ \t]*(?:require[ \t]+)?{re.escape(API_MODULE)}[ \t]+)v[^\s]+([ \t]*(?://.*)?$)"
)
NPM_WORKFLOW = "publish-npm.yml"


def run(
    cmd: list[str], *, cwd: Path = ROOT, check: bool = True, env: dict[str, str] | None = None
) -> str:
    """执行命令并返回标准输出。"""
    process_env = None
    if env:
        process_env = os.environ.copy()
        process_env.update(env)
    result = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, env=process_env)
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


def latest_tag(prefix: str, *, pattern: str | None = None) -> tuple[tuple[int, int, int], str] | None:
    """读取指定前缀下最大的语义化版本 tag。"""
    tags = run(["git", "tag", "--list", pattern or f"{prefix}v*"]).splitlines()
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
    backend_latest = latest_tag("backend/", pattern="backend/v*")
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


def tag_commit(tag: str) -> str:
    """读取 tag 指向的提交。"""
    return run(["git", "rev-list", "-n", "1", tag], check=False)


def release_tags(version: tuple[int, int, int]) -> tuple[str, ...]:
    """生成统一版本对应的 API、客户端、后端和发布 tag。"""
    root_tag = tag_text(version)
    return (
        f"backend/api/{root_tag}",
        f"backend/client/{root_tag}",
        f"backend/{root_tag}",
        root_tag,
        f"npm/{root_tag}",
    )


def resolve_release_tags(
    version: tuple[int, int, int], *, skip: tuple[str, ...] = ()
) -> tuple[str, ...]:
    """生成本次发布需要推送的 tag。"""
    tags = release_tags(version)
    for tag in tags:
        if tag in skip:
            continue
        if tag_exists_locally(tag) or tag_exists_remotely(tag):
            raise RuntimeError(f"tag 已存在，拒绝覆盖: {tag}")
    return tags


def release_version_at_head() -> tuple[int, int, int] | None:
    """工作区无改动且最新根 tag 指向 HEAD 时返回已发布版本。"""
    if run(["git", "status", "--porcelain"]):
        return None

    root_latest = latest_tag("")
    if root_latest is None:
        return None

    head = run(["git", "rev-parse", "HEAD"])
    if tag_commit(root_latest[1]) != head:
        return None

    for prefix in ("backend/api/", "backend/client/"):
        component_latest = latest_tag(prefix)
        if component_latest and component_latest[0] > root_latest[0]:
            if tag_commit(component_latest[1]) == head:
                return None
            raise RuntimeError(
                f"{prefix.rstrip('/')} 最新 tag {component_latest[1]} 高于根目录 {root_latest[1]}，请先修复版本线"
            )
    for prefix, pattern in (("backend/", "backend/v*"), ("npm/", None)):
        component_latest = latest_tag(prefix, pattern=pattern)
        if component_latest and component_latest[0] > root_latest[0]:
            raise RuntimeError(
                f"{prefix.rstrip('/')} 最新 tag {component_latest[1]} 高于根目录 {root_latest[1]}，请先修复版本线"
            )
    return root_latest[0]


def update_package_versions(version: tuple[int, int, int]) -> list[Path]:
    """将全部前端 npm 包统一更新到目标版本。"""
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


def update_project_version(version: tuple[int, int, int]) -> bool:
    """将后端服务版本更新为不带 v 前缀的发布版本。"""
    target = version_text(version)
    try:
        content = PROJECT_VERSION_FILE.read_text(encoding="utf-8")
    except OSError as err:
        raise RuntimeError(f"无法读取后端版本文件: {PROJECT_VERSION_FILE}: {err}") from err
    updated, count = PROJECT_VERSION_RE.subn(rf"\g<1>{target}\g<2>", content)
    if count != 1:
        raise RuntimeError(f"后端版本文件中应有且只有一个 Version 定义: {PROJECT_VERSION_FILE}")
    if updated == content:
        return False
    PROJECT_VERSION_FILE.write_text(updated, encoding="utf-8")
    return True


def update_api_module_version(path: Path, version: tuple[int, int, int]) -> bool:
    """将指定 Go 模块的 backend/api 依赖更新到发布版本。"""
    try:
        content = path.read_text(encoding="utf-8")
    except OSError as err:
        raise RuntimeError(f"无法读取 Go 模块文件: {path}: {err}") from err
    updated, count = API_MODULE_VERSION_RE.subn(
        rf"\g<1>v{version_text(version)}\g<2>", content
    )
    if count != 1:
        raise RuntimeError(f"Go 模块文件中应有且只有一个 {API_MODULE} 依赖: {path}")
    if updated == content:
        return False
    path.write_text(updated, encoding="utf-8")
    return True


def update_api_dependencies(version: tuple[int, int, int]) -> list[Path]:
    """更新 backend 与 backend/client 的 API 依赖并同步模块校验文件。"""
    changed: list[Path] = []
    for path in API_DEPENDENCY_FILES:
        if update_api_module_version(path, version):
            changed.append(path)
        run(["go", "mod", "tidy"], cwd=path.parent, env={"GOWORK": "off"})
    return changed


def commit_api_changes(version: tuple[int, int, int]) -> None:
    """先提交 backend/api 的工作区改动，确保 API tag 包含当前契约。"""
    if not run(["git", "status", "--porcelain", "--", "backend/api"]):
        return
    run(["git", "add", "-A", "--", "backend/api"])
    run(
        [
            "git",
            "commit",
            "--only",
            "backend/api",
            "-m",
            f"chore(release): 发布 backend/api v{version_text(version)}",
        ]
    )


def commit_all_changes(version: tuple[int, int, int]) -> None:
    """提交版本更新以及工作区中的全部本地改动。"""
    status = run(["git", "status", "--porcelain"])
    if not status:
        return
    run(["git", "add", "-A"])
    run(["git", "commit", "-m", f"chore(release): 统一升级 v{version_text(version)}"])


def run_frontend_package() -> None:
    """检查并打包全部前端 npm 包。"""
    run_stream(["make", "-C", "frontend", "package"])


def push_branch(branch: str) -> None:
    """推送版本提交到默认分支。"""
    run(["git", "push", "origin", branch])


def push_tag(tag: str) -> None:
    """创建并推送单个 tag。"""
    run(["git", "tag", tag])
    run(["git", "push", "origin", tag])
    print(f"已推送 tag: {tag}")


def ensure_api_tag(version: tuple[int, int, int], commit: str) -> None:
    """先创建并推送指向当前 API 提交的模块 tag。"""
    tag = release_tags(version)[0]
    current = tag_commit(tag)
    if current and current != commit:
        raise RuntimeError(f"tag {tag} 已指向其他提交，拒绝覆盖")
    if not current:
        run(["git", "tag", tag, commit])
    if tag_exists_remotely(tag):
        print(f"已存在 tag: {tag}")
        return
    run(["git", "push", "origin", tag])
    print(f"已推送 tag: {tag}")


def ensure_release_tags(version: tuple[int, int, int], commit: str) -> None:
    """复用当前版本 tag，并补推中断时缺失的同版本 tag。"""
    api_tag = release_tags(version)[0]
    for tag in release_tags(version):
        current = tag_commit(tag)
        if tag == api_tag:
            if not current:
                raise RuntimeError(f"缺少 API tag: {tag}，无法恢复发布")
            if tag_exists_remotely(tag):
                print(f"已存在 tag: {tag}")
                continue
            run(["git", "push", "origin", tag])
            print(f"已补推 tag: {tag}")
            continue
        if current and current != commit:
            raise RuntimeError(f"tag {tag} 已指向其他提交，拒绝覆盖")
        if not current:
            run(["git", "tag", tag, commit])
        if tag_exists_remotely(tag):
            print(f"已存在 tag: {tag}")
            continue
        run(["git", "push", "origin", tag])
        print(f"已补推 tag: {tag}")


def ensure_github_cli() -> None:
    """确保本机可通过 GitHub CLI 等待 npm 发布结果。"""
    if shutil.which("gh") is None:
        raise RuntimeError("make tag 需要 GitHub CLI，请先安装 gh 并执行 gh auth login")
    result = subprocess.run(
        ["gh", "auth", "status"],
        cwd=ROOT,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError("GitHub CLI 尚未登录，请先执行 gh auth login")


def find_npm_workflow(commit: str) -> tuple[dict[str, object] | None, str]:
    """查询指定提交最近一次 npm workflow。"""
    result = subprocess.run(
        [
            "gh",
            "run",
            "list",
            "--workflow",
            NPM_WORKFLOW,
            "--event",
            "push",
            "--limit",
            "20",
            "--json",
            "databaseId,headSha,status,conclusion,url",
        ],
        cwd=ROOT,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        return None, result.stderr.strip() or result.stdout.strip()
    try:
        runs = json.loads(result.stdout)
    except json.JSONDecodeError as err:
        return None, str(err)
    workflow = next((item for item in runs if item.get("headSha") == commit), None)
    return workflow, ""


def watch_npm_workflow(workflow: dict[str, object]) -> None:
    """等待 npm workflow 完成并透传执行结果。"""
    print(f"npm 发布任务: {workflow['url']}")
    run_stream(["gh", "run", "watch", str(workflow["databaseId"]), "--exit-status"])


def wait_npm_workflow(commit: str) -> None:
    """等待当前发布提交对应的 npm workflow 完成。"""
    last_error = ""
    for _ in range(30):
        workflow, last_error = find_npm_workflow(commit)
        if workflow is not None:
            watch_npm_workflow(workflow)
            return
        time.sleep(2)
    detail = f": {last_error}" if last_error else ""
    raise RuntimeError(f"未找到 npm 发布 workflow，请到 GitHub Actions 检查{detail}")


def resume_npm_workflow(commit: str) -> None:
    """复用当前版本的 npm workflow，失败时重跑，运行中则继续等待。"""
    workflow, error = find_npm_workflow(commit)
    if workflow is None:
        if error:
            raise RuntimeError(f"查询 npm 发布 workflow 失败: {error}")
        wait_npm_workflow(commit)
        return

    if workflow.get("status") == "completed":
        if workflow.get("conclusion") == "success":
            print(f"npm 发布已完成: {workflow['url']}")
            return
        print(f"重新运行失败的 npm 发布任务: {workflow['url']}")
        run(["gh", "run", "rerun", str(workflow["databaseId"])])
    else:
        print(f"继续等待 npm 发布任务: {workflow['url']}")
    watch_npm_workflow(workflow)


def run_backend_tests() -> None:
    """发布前执行后端完整测试。"""
    run_stream(["go", "test", "./..."], cwd=ROOT / "backend")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="统一发布 backend/api、backend/client、backend 和 frontend npm 包"
    )
    parser.add_argument("--version", help="目标版本，支持 X.Y.Z 或 vX.Y.Z；不指定时自动递增")
    parser.add_argument("--dry-run", action="store_true", help="仅检查并打印目标，不修改文件、不提交、不推送")
    args = parser.parse_args()

    try:
        branch = detect_remote_branch()
        fetch_remote(branch)
        ensure_release_branch(branch)
        existing_version = release_version_at_head()
        if existing_version is not None:
            if args.version and parse_version(args.version) != existing_version:
                raise RuntimeError(
                    f"当前提交已发布为 {tag_text(existing_version)}，没有新改动，拒绝创建新版本 {args.version}"
                )
            commit = run(["git", "rev-parse", "HEAD"])
            print(f"当前提交已发布为 {tag_text(existing_version)}，不升级版本。")
            if args.dry_run:
                print("dry-run：未补推 tag、未重跑或等待 npm workflow。")
                return 0
            ensure_github_cli()
            ensure_release_tags(existing_version, commit)
            resume_npm_workflow(commit)
            return 0

        version = resolve_version(args.version)
        api_tag = release_tags(version)[0]
        tags = resolve_release_tags(version, skip=(api_tag,))
        target = version_text(version)
        print(f"发布版本: {target}")
        print(
            "发布顺序: 提交 backend/api 改动（如有） -> backend/api tag "
            "-> 更新 backend/client 与 backend API 依赖 "
            f"-> 提交 -> frontend package -> 推送分支 -> {' -> '.join(tags)}"
        )

        if args.dry_run:
            print("dry-run：未修改文件、未提交、未推送。")
            return 0

        ensure_github_cli()

        commit_api_changes(version)
        ensure_api_tag(version, run(["git", "rev-parse", "HEAD"]))
        update_api_dependencies(version)
        update_project_version(version)
        update_package_versions(version)
        run_backend_tests()
        commit_all_changes(version)
        run_frontend_package()
        push_branch(branch)
        commit = run(["git", "rev-parse", "HEAD"])
        for tag in tags:
            push_tag(tag)
        wait_npm_workflow(commit)
        return 0
    except RuntimeError as err:
        print(str(err), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
