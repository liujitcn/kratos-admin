#!/bin/sh
set -eu

web_seed_directory="${KRATOS_WEB_SEED_DIRECTORY:-/opt/kratos/web}"
web_directory="${KRATOS_WEB_DIRECTORY:-/app/web}"
config_seed_directory="${KRATOS_CONFIG_SEED_DIRECTORY:-/opt/kratos/configs}"
config_directory="${KRATOS_CONFIG_DIRECTORY:-/app/configs}"

mkdir -p "$web_directory"
if [ -d "$web_seed_directory" ]; then
  cp -R "$web_seed_directory"/. "$web_directory"/
fi

mkdir -p "$config_directory"
if [ -d "$config_seed_directory" ]; then
  cp -Rn "$config_seed_directory"/. "$config_directory"/
fi

exec "$@"
