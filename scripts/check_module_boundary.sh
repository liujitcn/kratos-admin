#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
backend_root="$repo_root/backend"

# 业务模块自动发现：backend/service 下除基础目录外的模块（如 cms、uba）。
# 边界规则：业务模块只能被组合根（internal/cmd）装配，基础模块与其他业务模块均不得依赖它。
base_dirs=(base system)
modules=()
for dir in "$backend_root"/service/*/; do
  name="$(basename "$dir")"
  is_base=false
  for b in "${base_dirs[@]}"; do
    [[ "$name" == "$b" ]] && is_base=true
  done
  [[ "$is_base" == true ]] || modules+=("$name")
done

if [[ ${#modules[@]} -eq 0 ]]; then
  echo '未发现业务模块，跳过边界检查。'
  exit 0
fi

go_module="$(sed -n 's/^module[[:space:]]*//p' "$backend_root/go.mod" | head -1)"

check_absent() {
  local description="$1"
  local pattern="$2"
  shift 2
  local matches
  matches="$(rg -n --glob '*.go' --glob '*.proto' --glob '*.ts' --glob '*.tsx' --glob '*.js' --glob '*.vue' "$pattern" "$@" || true)"
  if [[ -n "$matches" ]]; then
    printf '模块边界检查失败：%s\n%s\n' "$description" "$matches" >&2
    exit 1
  fi
}

# ---- 后端：基于 go list 的编译依赖图检查（覆盖别名导入，文本搜索无法覆盖） ----
cd "$backend_root"
edges="$(go list -f '{{$p := .ImportPath}}{{range .Imports}}{{$p}} {{.}}{{"\n"}}{{end}}' ./...)"
for mod in "${modules[@]}"; do
  violations="$(awk -v m="$mod" -v g="$go_module" '
    NF == 2 {
      pat = "^" g "/(service/" m "|server/" m "|api/gen/go/" m ")(/|$)"
      if ($1 ~ "^" g "/internal/") next
      if ($2 ~ pat && $1 !~ pat) print "  " $1 " -> " $2
    }' <<<"$edges")"
  if [[ -n "$violations" ]]; then
    printf '模块边界检查失败：其他代码不得依赖业务模块「%s」\n%s\n' "$mod" "$violations" >&2
    exit 1
  fi
done

# ---- 前端 admin：业务模块目录（api/views/rpc 下同名目录）之外不得导入该模块 ----
cd "$repo_root"
for mod in "${modules[@]}"; do
  check_absent \
    "基础前端代码不得导入业务模块「${mod}」" \
    "[\"'](@/(api|views|rpc)/${mod}|\.[^\"']*/${mod}/)" \
    --glob "!**/api/${mod}/**" --glob "!**/views/${mod}/**" --glob "!**/rpc/${mod}/**" \
    frontend/admin/src
done

# ---- 历史迁移专项检查：防止已下沉的业务类型回流到通用协议 ----
cd "$backend_root"
check_absent \
  '通用协议不得保留商城 SSE 或推荐调试枚举' \
  'AdvanceDataType|SseRefresh(Target|Reason)' \
  api/proto/common/v1

cd "$repo_root"
frontend_targets=(
  frontend/admin/src/api/base
  frontend/admin/src/rpc/base/v1
  frontend/admin/src/rpc/common/v1
)
check_absent \
  '基础前端 SSE 与通用 RPC 类型不得保留商城工作台或推荐调试类型' \
  'SseRefresh|AdvanceDataType' \
  "${frontend_targets[@]}"

printf '模块边界检查通过（业务模块：%s）。\n' "${modules[*]}"
