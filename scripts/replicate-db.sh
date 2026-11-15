#!/usr/bin/env bash
# replicate-db.sh — copy a deployed database into the local dev Postgres, anonymized.
#
#   1. pg_dump the source (Azure prod/dev) database — read-only, nothing is written there
#   2. restore it into a local *staging* database
#   3. run scripts/sql/mask-replica.sql on the staging copy (one transaction,
#      self-verifying; any failure discards the copy)
#   4. only then swap the staging database in under the name the app uses
#   5. delete the raw (unmasked) dump
#
# The unmasked data exists locally only in a mode-700 temp dir and in the
# staging database, and both are removed on success and on failure.
#
# Source connection (env). Get the password with: scripts/get-db-password.sh --env prod
#   SRC_DB_HOST      e.g. psql-volunteer-media-prod.postgres.database.azure.com  (required)
#   SRC_DB_PASSWORD  (required)
#   SRC_DB_USER      default: pgadmin
#   SRC_DB_NAME      default: volunteermedia
#   SRC_DB_PORT      default: 5432
#   SRC_DB_SSLMODE   default: require
#
# The Azure server's firewall only admits Azure services. Add a temporary rule
# for your IP first, and remove it afterwards:
#   az postgres flexible-server firewall-rule create -g <rg> -n <server> \
#     --rule-name tmp-replicate --start-ip-address <ip> --end-ip-address <ip>
#
# Target (local) connection (env) — defaults match docker-compose.yml postgres_dev:
#   DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=volunteer_media_dev
#
# Postgres client tools: by default they run inside the postgres_dev container
# (pg15, matches Azure) when it is running, otherwise the host's pg_dump/psql
# (must be major version >= 15). Force with PG_TOOLS=docker|local.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MASK_SQL="$SCRIPT_DIR/sql/mask-replica.sql"

usage() {
  cat <<'EOF'
Usage: scripts/replicate-db.sh [options]

Options:
  --from-dump FILE   Restore from an existing pg_dump custom-format file instead
                     of connecting to the source (the file is left in place).
  --scrub-text       Replace comment/update/announcement/animal text outright
                     instead of redacting emails, phone numbers and names in it.
  --no-media         Drop image/document bytes stored in Postgres (smaller copy;
                     photos 404 locally).
  --dev-password PW  Password for every masked account (default: LocalDev-2026!).
                     Also settable via REPLICA_DEV_PASSWORD.
  --yes              Don't ask before replacing the local database.
  -h, --help         Show this help.

Every account becomes user<id> / <dev password>. The login list (with roles)
is printed at the end.
EOF
}

FROM_DUMP=""
SCRUB_TEXT=off
STRIP_MEDIA=off
ASSUME_YES=0
DEV_PASSWORD="${REPLICA_DEV_PASSWORD:-LocalDev-2026!}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --from-dump)    FROM_DUMP="${2:?--from-dump needs a file}"; shift 2 ;;
    --scrub-text)   SCRUB_TEXT=on; shift ;;
    --no-media)     STRIP_MEDIA=on; shift ;;
    --dev-password) DEV_PASSWORD="${2:?--dev-password needs a value}"; shift 2 ;;
    --yes|-y)       ASSUME_YES=1; shift ;;
    -h|--help)      usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

die()  { echo "❌ $*" >&2; exit 1; }
info() { echo "▶ $*"; }

[[ -f "$MASK_SQL" ]] || die "missing $MASK_SQL"

# ----------------------------------------------------------------------------
# Target (local) configuration and safety rails
# ----------------------------------------------------------------------------
TGT_HOST="${DB_HOST:-localhost}"
TGT_PORT="${DB_PORT:-5432}"
TGT_USER="${DB_USER:-postgres}"
TGT_PASSWORD="${DB_PASSWORD:-postgres}"
TGT_DB="${DB_NAME:-volunteer_media_dev}"
STAGING_DB="${TGT_DB}_replica_staging"
CONTAINER="${PG_CONTAINER:-volunteer_media_db_dev}"

case "$TGT_HOST" in
  localhost|127.0.0.1|::1|postgres_dev|host.docker.internal) ;;
  *) die "target DB_HOST=$TGT_HOST is not local. This script only writes to a local database." ;;
esac
[[ "$TGT_DB" =~ ^[a-zA-Z0-9_]+$ ]] || die "DB_NAME must be alphanumeric/underscore"

if [[ -z "${PG_TOOLS:-}" ]]; then
  if command -v docker >/dev/null 2>&1 &&
     docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$CONTAINER"; then
    PG_TOOLS=docker
  else
    PG_TOOLS=local
  fi
fi

# pg <password> <sslmode> <command> [args...]
# Passwords travel via the environment, never on a command line.
pg() {
  local pw="$1" ssl="$2"; shift 2
  if [[ "$PG_TOOLS" == docker ]]; then
    PGPASSWORD="$pw" PGSSLMODE="$ssl" docker exec -i -e PGPASSWORD -e PGSSLMODE "$CONTAINER" "$@"
  else
    PGPASSWORD="$pw" PGSSLMODE="$ssl" "$@"
  fi
}

# In docker mode the tools run inside the DB container, where the server is localhost.
if [[ "$PG_TOOLS" == docker ]]; then
  TGT_CONN=(-h localhost -p 5432 -U "$TGT_USER")
else
  for tool in pg_dump pg_restore psql; do
    command -v "$tool" >/dev/null || die "$tool not found. Start the dev DB (make db-start) or install the PostgreSQL client."
  done
  major="$(pg_dump --version | sed -E 's/[^0-9]*([0-9]+).*/\1/')"
  (( major >= 15 )) || die "local pg_dump is v$major; need >= 15 to dump the Azure (PG15) server"
  TGT_CONN=(-h "$TGT_HOST" -p "$TGT_PORT" -U "$TGT_USER")
fi

tpsql() { pg "$TGT_PASSWORD" disable psql "${TGT_CONN[@]}" -v ON_ERROR_STOP=1 -X -q "$@"; }

tpsql -d postgres -c 'SELECT 1' >/dev/null 2>&1 ||
  die "cannot reach local Postgres at $TGT_HOST:$TGT_PORT as $TGT_USER (tools: $PG_TOOLS). Run: make db-start"

# ----------------------------------------------------------------------------
# Source configuration
# ----------------------------------------------------------------------------
if [[ -z "$FROM_DUMP" ]]; then
  SRC_HOST="${SRC_DB_HOST:?Set SRC_DB_HOST (or use --from-dump)}"
  SRC_PASSWORD="${SRC_DB_PASSWORD:?Set SRC_DB_PASSWORD (scripts/get-db-password.sh --env prod)}"
  SRC_USER="${SRC_DB_USER:-pgadmin}"
  SRC_NAME="${SRC_DB_NAME:-volunteermedia}"
  SRC_PORT="${SRC_DB_PORT:-5432}"
  SRC_SSLMODE="${SRC_DB_SSLMODE:-require}"
  if [[ "$SRC_HOST" == "$TGT_HOST" && "$SRC_PORT" == "$TGT_PORT" && "$SRC_NAME" == "$TGT_DB" ]]; then
    die "source and target are the same database"
  fi
  SOURCE_DESC="$SRC_USER@$SRC_HOST:$SRC_PORT/$SRC_NAME"
else
  [[ -f "$FROM_DUMP" ]] || die "dump file not found: $FROM_DUMP"
  SOURCE_DESC="dump file $FROM_DUMP"
fi

echo ""
echo "  Source:  $SOURCE_DESC"
echo "  Target:  $TGT_USER@$TGT_HOST:$TGT_PORT/$TGT_DB  (tools: $PG_TOOLS)"
echo "  Masking: scrub-text=$SCRUB_TEXT  no-media=$STRIP_MEDIA"
echo ""
echo "  The local database '$TGT_DB' will be DROPPED and replaced."
echo "  Stop the local API first (it is disconnected regardless)."
echo ""
if (( ! ASSUME_YES )); then
  read -r -p "Continue? [y/N] " reply
  [[ "$reply" =~ ^[Yy]$ ]] || die "aborted"
fi

# ----------------------------------------------------------------------------
# Cleanup: raw dump and staging DB hold unmasked data — always remove them.
# ----------------------------------------------------------------------------
WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/replicate-db.XXXXXX")"
chmod 700 "$WORK_DIR"
cleanup() {
  local rc=$?
  rm -rf "$WORK_DIR"
  tpsql -d postgres -c "DROP DATABASE IF EXISTS \"$STAGING_DB\" WITH (FORCE)" >/dev/null 2>&1 || true
  if (( rc != 0 )); then
    echo "❌ Failed. Unmasked staging data and the raw dump were removed; '$TGT_DB' was left untouched." >&2
  fi
}
trap cleanup EXIT
trap 'exit 130' INT TERM

# ----------------------------------------------------------------------------
# 1. Dump
# ----------------------------------------------------------------------------
if [[ -z "$FROM_DUMP" ]]; then
  DUMP_FILE="$WORK_DIR/source.dump"
  info "Dumping $SOURCE_DESC (read-only)…"
  pg "$SRC_PASSWORD" "$SRC_SSLMODE" pg_dump \
      -h "$SRC_HOST" -p "$SRC_PORT" -U "$SRC_USER" -d "$SRC_NAME" \
      --format=custom --no-owner --no-privileges --schema=public \
      > "$DUMP_FILE"
  info "Dump size: $(du -h "$DUMP_FILE" | cut -f1)"
else
  DUMP_FILE="$FROM_DUMP"
fi

# ----------------------------------------------------------------------------
# 2. Restore into staging
# ----------------------------------------------------------------------------
info "Restoring into staging database $STAGING_DB…"
tpsql -d postgres -c "DROP DATABASE IF EXISTS \"$STAGING_DB\" WITH (FORCE)"
tpsql -d postgres -c "CREATE DATABASE \"$STAGING_DB\""
# The dump is limited to the public schema (skipping Azure's own schemas and
# extensions), so create the extensions the app uses up front. vector is
# optional — the app degrades without it.
tpsql -d "$STAGING_DB" -c 'CREATE EXTENSION IF NOT EXISTS pg_trgm'
tpsql -d "$STAGING_DB" -c 'CREATE EXTENSION IF NOT EXISTS vector' 2>/dev/null ||
  echo "  (pgvector not available locally — embedding columns will fail to restore; use the postgres_dev container)"

# Render the dump to SQL and stream it into psql. The schema-only dump says
# CREATE SCHEMA public, which already exists in a fresh database — drop that line.
pg "$TGT_PASSWORD" disable pg_restore --no-owner --no-privileges --no-comments -f - \
    < "$DUMP_FILE" |
  grep -v '^CREATE SCHEMA public;$' |
  tpsql -d "$STAGING_DB" >/dev/null

# Raw dump is no longer needed — delete it now rather than at exit.
if [[ -z "$FROM_DUMP" ]]; then rm -f "$DUMP_FILE"; fi

# ----------------------------------------------------------------------------
# 3. Mask (transactional, self-verifying)
# ----------------------------------------------------------------------------
info "Masking personal data…"
tpsql -d "$STAGING_DB" \
    -v dev_password="$DEV_PASSWORD" \
    -v scrub_text="$SCRUB_TEXT" \
    -v strip_media="$STRIP_MEDIA" \
    < "$MASK_SQL"

if [[ "$STRIP_MEDIA" == on ]]; then
  info "Reclaiming space from stripped media…"
  tpsql -d "$STAGING_DB" -c 'VACUUM FULL animal_images, animals, scripts, group_documents'
fi

# ----------------------------------------------------------------------------
# 4. Swap staging in
# ----------------------------------------------------------------------------
info "Replacing $TGT_DB…"
tpsql -d postgres -c "DROP DATABASE IF EXISTS \"$TGT_DB\" WITH (FORCE)"
tpsql -d postgres -c "ALTER DATABASE \"$STAGING_DB\" RENAME TO \"$TGT_DB\""

# ----------------------------------------------------------------------------
# 5. Report
# ----------------------------------------------------------------------------
echo ""
echo "✅ Local database '$TGT_DB' now holds a masked copy of $SOURCE_DESC."
echo ""
echo "Logins (password for all: $DEV_PASSWORD):"
tpsql -d "$TGT_DB" -P footer=off -c "
  SELECT u.username,
         CASE WHEN u.is_admin THEN 'site admin'
              WHEN bool_or(ug.is_group_admin) THEN 'group admin'
              ELSE 'volunteer' END AS role,
         u.first_name || ' ' || u.last_name AS name,
         coalesce(string_agg(g.name || CASE WHEN ug.is_group_admin THEN '*' ELSE '' END, ', ' ORDER BY g.name), '') AS \"groups (* = admin)\"
  FROM users u
  LEFT JOIN user_groups ug ON ug.user_id = u.id
  LEFT JOIN groups g ON g.id = ug.group_id AND g.deleted_at IS NULL
  WHERE u.deleted_at IS NULL
  GROUP BY u.id
  ORDER BY u.is_admin DESC, bool_or(ug.is_group_admin) DESC NULLS LAST, u.id
  LIMIT 25"
echo ""
echo "Notes:"
echo "  • Media stored in Azure Blob (storage provider 'azure') is not copied and will 404 locally."
echo "  • Embeddings were cleared. Set VOYAGE_API_KEY locally to have them rebuilt from the masked text."
echo "  • Start the API as usual (make dev-backend); migrations run on startup."
