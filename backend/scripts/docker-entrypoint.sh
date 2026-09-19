#!/bin/sh
set -eu

static_seed_directory="${KRATOS_STATIC_SEED_DIRECTORY:-/opt/kratos-admin/static}"
static_directory="${KRATOS_STATIC_DIRECTORY:-/app/web}"
config_seed_directory="${KRATOS_CONFIG_SEED_DIRECTORY:-/opt/kratos-admin/configs}"
config_directory="${KRATOS_CONFIG_DIRECTORY:-/app/configs}"

mkdir -p "$static_directory"
if [ -d "$static_seed_directory" ]; then
  cp -R "$static_seed_directory"/. "$static_directory"/
fi

mkdir -p "$config_directory"
if [ -d "$config_seed_directory" ]; then
  cp -Rn "$config_seed_directory"/. "$config_directory"/
fi

exec "$@"
