#!/usr/bin/env bash
set -Eeuo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
compose_file="${script_dir}/../docker-compose.yaml"
env_file="${script_dir}/../gotenberg/.env"

command -v docker >/dev/null 2>&1 || {
  printf 'docker is required\n' >&2
  exit 1
}

docker compose version >/dev/null
mkdir -p "${script_dir}/data"
docker compose --env-file "${env_file}" -f "${compose_file}" up -d --build markitdown-mcp
docker compose --env-file "${env_file}" -f "${compose_file}" ps markitdown-mcp
