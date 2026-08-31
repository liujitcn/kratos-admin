#!/usr/bin/env bash
set -euo pipefail

backup_file="${1:-}"
if [[ -z "${backup_file}" || ! -f "${backup_file}" ]]; then
  echo "用法: $0 <备份文件.sql.enc|备份文件.sql.gz.enc>" >&2
  exit 2
fi
if [[ -z "${MYSQLDUMP_DATABASE:-}" ]]; then
  echo "请设置 MYSQLDUMP_DATABASE" >&2
  exit 2
fi

checksum_file="${backup_file}.sha256"
if [[ ! -f "${checksum_file}" ]]; then
  echo "备份缺少 SHA-256 校验文件: ${checksum_file}" >&2
  exit 2
fi
if [[ -z "${BACKUP_INTEGRITY_KEY:-}" ]]; then
  echo "请设置 BACKUP_INTEGRITY_KEY 以校验备份完整性" >&2
  exit 2
fi
hmac_file="${backup_file}.hmac"
if [[ ! -f "${hmac_file}" ]] || ! command -v openssl >/dev/null 2>&1; then
  echo "备份缺少 HMAC 校验文件或 openssl" >&2
  exit 2
fi
expected_hmac="$(awk 'NR == 1 {print $1}' "${hmac_file}")"
actual_hmac="$(openssl dgst -sha256 -hmac "${BACKUP_INTEGRITY_KEY}" "${backup_file}" | awk '{print $NF}')"
if [[ -z "${expected_hmac}" || "${actual_hmac}" != "${expected_hmac}" ]]; then
  echo "备份 HMAC 校验失败" >&2
  exit 2
fi
expected_checksum="$(awk 'NR == 1 {print $1}' "${checksum_file}")"
if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum="$(sha256sum "${backup_file}" | awk '{print $1}')"
else
  actual_checksum="$(shasum -a 256 "${backup_file}" | awk '{print $1}')"
fi
if [[ -z "${expected_checksum}" || "${actual_checksum}" != "${expected_checksum}" ]]; then
  echo "备份 SHA-256 校验失败" >&2
  exit 2
fi

restore_file="${backup_file}"
decrypted_file=""
cleanup() {
  [[ -z "${decrypted_file}" ]] || rm -f -- "${decrypted_file}"
}
trap cleanup EXIT
if [[ "${backup_file}" == *.enc ]]; then
  if [[ -z "${BACKUP_ENCRYPTION_KEY:-}" ]]; then
    echo "请设置 BACKUP_ENCRYPTION_KEY 以解密备份" >&2
    exit 2
  fi
  if ! command -v openssl >/dev/null 2>&1; then
    echo "恢复加密备份需要 openssl" >&2
    exit 2
  fi
  decrypted_file="$(mktemp "${TMPDIR:-/tmp}/kratos-admin-restore.XXXXXX")"
  chmod 600 "${decrypted_file}"
  if ! openssl enc -d -aes-256-cbc -pbkdf2 -iter 100000 -in "${backup_file}" -out "${decrypted_file}" -pass env:BACKUP_ENCRYPTION_KEY; then
    echo "备份解密失败" >&2
    exit 2
  fi
  restore_file="${decrypted_file}"
fi

mysql_bin="${MYSQL_BIN:-mysql}"
mysql_args=("--database=${MYSQLDUMP_DATABASE}")
[[ -n "${MYSQLDUMP_HOST:-}" ]] && mysql_args+=("--host=${MYSQLDUMP_HOST}")
[[ -n "${MYSQLDUMP_PORT:-}" ]] && mysql_args+=("--port=${MYSQLDUMP_PORT}")
[[ -n "${MYSQLDUMP_USER:-}" ]] && mysql_args+=("--user=${MYSQLDUMP_USER}")

if [[ "${backup_file}" == *.gz || "${backup_file}" == *.gz.enc ]]; then
  gzip -dc -- "${restore_file}" | MYSQL_PWD="${MYSQLDUMP_PASSWORD:-}" "${mysql_bin}" "${mysql_args[@]}"
else
  MYSQL_PWD="${MYSQLDUMP_PASSWORD:-}" "${mysql_bin}" "${mysql_args[@]}" < "${restore_file}"
fi
