#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
backend_root="$repo_root/backend"

# base、admin 与 app 是内置服务，其余一级目录视为可选业务模块。
base_dirs=(admin app base)
modules=()
for module_dir in "$backend_root"/internal/service/*/; do
  module_name="$(basename "$module_dir")"
  is_base=false
  for base_dir in "${base_dirs[@]}"; do
    [[ "$module_name" == "$base_dir" ]] && is_base=true
  done
  [[ "$is_base" == true ]] || modules+=("$module_name")
done

if [[ ${#modules[@]} -eq 0 ]]; then
  printf '未发现后端业务模块，跳过后端边界检查。\n'
  exit 0
fi

go_module="$(sed -n 's/^module[[:space:]]*//p' "$backend_root/go.mod" | head -1)"

cd "$backend_root"
edges="$(go list -f '{{$package := .ImportPath}}{{range .Imports}}{{$package}} {{.}}{{"\n"}}{{end}}' ./...)"
for module_name in "${modules[@]}"; do
  violations="$(awk -v module_name="$module_name" -v go_module="$go_module" '
    NF == 2 {
      module_pattern = "^" go_module "/(internal/service/" module_name "|internal/server/" module_name "|api/gen/go/" module_name ")(/|$)"
      composition_pattern = "^" go_module "(/cmd/|$)"
      if ($2 ~ module_pattern && $1 !~ module_pattern && $1 !~ composition_pattern) {
        print "  " $1 " -> " $2
      }
    }' <<<"$edges")"
  if [[ -n "$violations" ]]; then
    printf '后端模块边界检查失败：其他模块不得依赖业务模块「%s」\n%s\n' "$module_name" "$violations" >&2
    exit 1
  fi
done

printf '后端模块边界检查通过（业务模块：%s）。\n' "${modules[*]}"
