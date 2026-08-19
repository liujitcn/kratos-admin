#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
admin_root="$repo_root/frontend/admin"

check_absent() {
  local description="$1"
  local pattern="$2"
  shift 2
  local matches
  matches="$(rg -n --glob '*.cjs' --glob '*.js' --glob '*.jsx' --glob '*.mjs' --glob '*.ts' --glob '*.tsx' --glob '*.vue' "$pattern" "$@" || true)"
  if [[ -n "$matches" ]]; then
    printf '模块边界检查失败：%s\n%s\n' "$description" "$matches" >&2
    exit 1
  fi
}

check_absent \
  "基础前端代码不得反向导入业务模块" \
  "[\"'](@liujitcn/kratos-admin-[^/\"']+|@[^/\"']+/admin-module|\.\.[^\"']*/modules/)" \
  "$admin_root/packages/core/src"

pnpm --dir "$admin_root" check:exports
printf '管理端模块边界检查通过。\n'
