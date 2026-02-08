#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat <<'USAGE'
Usage: scripts/check-media-storage.sh

Optional flags:
	--tf-dir <path>     Load defaults from terraform outputs in this directory
											(default: terraform/environments/dev)
	--az-check          Use Azure CLI to verify storage state (list counts + spot-check blobs)
	--no-az-check       Skip Azure CLI checks even if az is available
	--sample <n>        How many DB blob identifiers to spot-check (default: 20)
	-h, --help          Show this help

Checks whether animal images and protocol documents are stored in Postgres (bytea)
or external storage (Azure Blob), by inspecting database columns:
- animal_images.storage_provider / animal_images.image_data / animal_images.blob_identifier
- animals.protocol_document_provider / animals.protocol_document_data / animals.protocol_document_blob_identifier

Connection precedence:
1) Uses DATABASE_URL if set
2) Otherwise uses DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME (with dev defaults)
3) If psql isn't available, tries: docker compose exec -T postgres_dev psql ...

Environment variables (optional):
	DATABASE_URL
	DB_HOST (default: localhost)
	DB_PORT (default: 5432)
	DB_USER (default: postgres)
	DB_PASSWORD (default: postgres)
	DB_NAME (default: volunteer_media_dev)
	DB_SSLMODE (recommended for Azure: require)

Exit codes:
	0 success
	2 unable to connect / missing tooling
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
	exit 0
fi

die() {
	echo "ERROR: $*" >&2
	exit 2
}

have_cmd() {
	command -v "$1" >/dev/null 2>&1
}

python_cmd() {
	if have_cmd python3; then
		echo "python3"
		return 0
	fi
	if have_cmd python; then
		echo "python"
		return 0
	fi
	return 1
}

tf_dir="terraform/environments/dev"
do_az_check="auto"
sample_n=20

while [[ $# -gt 0 ]]; do
	case "$1" in
		--tf-dir)
			tf_dir="${2:-}";
			[[ -n "$tf_dir" ]] || die "--tf-dir requires a path"
			shift 2
			;;
		--az-check)
			do_az_check="true"
			shift
			;;
		--no-az-check)
			do_az_check="false"
			shift
			;;
		--sample)
			sample_n="${2:-}";
			[[ "$sample_n" =~ ^[0-9]+$ ]] || die "--sample must be an integer"
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			die "Unknown argument: $1 (use --help)"
			;;
	esac
done

load_tf_defaults() {
	local dir="$1"

	have_cmd terraform || return 0
	[[ -d "$dir" ]] || return 0

	local py
	py="$(python_cmd)" || return 0

	# Terraform output requires an initialized working directory; if it fails, just skip.
	local out
	if ! out="$(terraform -chdir="$dir" output -json -no-color 2>/dev/null)"; then
		return 0
	fi

	# Terraform may sometimes emit non-JSON text (e.g., warnings) on stdout.
	# Best-effort: extract the JSON object boundaries before parsing.
	out="$($py - <<PY 2>/dev/null || true
import sys
s = '''$out'''
start = s.find('{')
end = s.rfind('}')
if start == -1 or end == -1 or end <= start:
		sys.exit(0)
print(s[start:end+1])
PY
)"
	if [[ -z "$out" ]]; then
		return 0
	fi

	# Helper: only set var if not already set
	local set_if_empty
	set_if_empty() {
		local var_name="$1"
		local var_value="$2"
		if [[ -z "${!var_name:-}" && -n "$var_value" && "$var_value" != "null" ]]; then
			export "$var_name=$var_value"
		fi
	}

	# Pull values from known output names.
	local db_host db_port db_name db_user storage_account storage_container container_app_id rg

	local fields
	fields="$($py - <<PY 2>/dev/null || true
import json

try:
    o = json.loads('''$out''')
except Exception:
    raise SystemExit(0)

db = o.get('database_connection_info', {}).get('value', {})

vals = [
    str(db.get('host', '') or ''),
    str(db.get('port', '') or ''),
    str(db.get('database', '') or ''),
    str(db.get('username', '') or ''),
    str(o.get('storage_account_name', {}).get('value', '') or ''),
    str(o.get('storage_container_name', {}).get('value', '') or ''),
    str(o.get('container_app_id', {}).get('value', '') or ''),
    str(o.get('resource_group_name', {}).get('value', '') or ''),
]

print("\t".join(vals))
PY
)"

	if [[ -n "$fields" ]]; then
		IFS=$'\t' read -r db_host db_port db_name db_user storage_account storage_container container_app_id rg <<< "$fields"
	fi

	set_if_empty DB_HOST "$db_host"
	set_if_empty DB_PORT "$db_port"
	set_if_empty DB_NAME "$db_name"
	set_if_empty DB_USER "$db_user"

	set_if_empty AZURE_STORAGE_ACCOUNT_NAME "$storage_account"
	set_if_empty AZURE_STORAGE_CONTAINER_NAME "$storage_container"

	set_if_empty TF_CONTAINER_APP_ID "$container_app_id"
	set_if_empty TF_RESOURCE_GROUP_NAME "$rg"
}

# Run a SQL snippet via psql.
run_sql() {
	local sql="$1"

	if have_cmd psql; then
		if [[ -n "${DATABASE_URL:-}" ]]; then
			PGPASSWORD="${PGPASSWORD:-}" \
				PGSSLMODE="${PGSSLMODE:-${DB_SSLMODE:-}}" \
				psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -X -q -P pager=off -c "$sql"
			return 0
		fi

		local host="${DB_HOST:-localhost}"
		local port="${DB_PORT:-5432}"
		local user="${DB_USER:-postgres}"
		local db="${DB_NAME:-volunteer_media_dev}"

		# Prefer DB_PASSWORD but allow user-set PGPASSWORD.
		local pw="${PGPASSWORD:-${DB_PASSWORD:-postgres}}"

		PGPASSWORD="$pw" \
			PGSSLMODE="${PGSSLMODE:-${DB_SSLMODE:-}}" \
			psql -h "$host" -p "$port" -U "$user" -d "$db" -v ON_ERROR_STOP=1 -X -q -P pager=off -c "$sql"
		return 0
	fi

	if have_cmd docker-compose; then
		local user="${DB_USER:-postgres}"
		local db="${DB_NAME:-volunteer_media_dev}"
		docker-compose exec -T postgres_dev psql -U "$user" -d "$db" -v ON_ERROR_STOP=1 -X -q -P pager=off -c "$sql"
		return 0
	fi

	if have_cmd docker && docker compose version >/dev/null 2>&1; then
		local user="${DB_USER:-postgres}"
		local db="${DB_NAME:-volunteer_media_dev}"
		docker compose exec -T postgres_dev psql -U "$user" -d "$db" -v ON_ERROR_STOP=1 -X -q -P pager=off -c "$sql"
		return 0
	fi

	die "psql not found and docker compose exec not available. Install psql or run with docker-compose."
}

# Run SQL and return tuples only (no headers/footers) - for spot-check queries.
run_sql_tuples() {
	local sql="$1"

	if have_cmd psql; then
		if [[ -n "${DATABASE_URL:-}" ]]; then
			PGPASSWORD="${PGPASSWORD:-}" \
				PGSSLMODE="${PGSSLMODE:-${DB_SSLMODE:-}}" \
				psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -X -q -t -A -P pager=off -c "$sql"
			return 0
		fi

		local host="${DB_HOST:-localhost}"
		local port="${DB_PORT:-5432}"
		local user="${DB_USER:-postgres}"
		local db="${DB_NAME:-volunteer_media_dev}"
		local pw="${PGPASSWORD:-${DB_PASSWORD:-postgres}}"

		PGPASSWORD="$pw" \
			PGSSLMODE="${PGSSLMODE:-${DB_SSLMODE:-}}" \
			psql -h "$host" -p "$port" -U "$user" -d "$db" -v ON_ERROR_STOP=1 -X -q -t -A -P pager=off -c "$sql"
		return 0
	fi

	if have_cmd docker-compose; then
		local user="${DB_USER:-postgres}"
		local db="${DB_NAME:-volunteer_media_dev}"
		docker-compose exec -T postgres_dev psql -U "$user" -d "$db" -v ON_ERROR_STOP=1 -X -q -t -A -P pager=off -c "$sql"
		return 0
	fi

	if have_cmd docker && docker compose version >/dev/null 2>&1; then
		local user="${DB_USER:-postgres}"
		local db="${DB_NAME:-volunteer_media_dev}"
		docker compose exec -T postgres_dev psql -U "$user" -d "$db" -v ON_ERROR_STOP=1 -X -q -t -A -P pager=off -c "$sql"
		return 0
	fi

	die "psql not found and docker compose exec not available."
}

section() {
	echo
	echo "== $1 =="
}

# Try to load defaults from Terraform outputs (best-effort).
load_tf_defaults "$tf_dir"

section "Database connectivity"
if [[ -n "${DATABASE_URL:-}" ]]; then
	echo "Using DATABASE_URL (value not printed)"
else
	echo "Using DB_* env (host=${DB_HOST:-localhost} port=${DB_PORT:-5432} user=${DB_USER:-postgres} db=${DB_NAME:-volunteer_media_dev})"
fi

# Sanity check connection.
run_sql "SELECT version() AS postgres_version;" >/dev/null || die "Unable to query database"
echo "Connected OK"

section "Animal images (animal_images)"

# Only run if the table exists.
if run_sql "SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='animal_images';" | grep -q 1; then
	run_sql "
WITH img AS (
	SELECT
		COALESCE(NULLIF(storage_provider, ''), 'postgres') AS provider,
		image_data,
		blob_identifier,
		blob_extension,
		deleted_at
	FROM animal_images
)
SELECT
	provider,
	COUNT(*) FILTER (WHERE deleted_at IS NULL) AS active_rows,
	COUNT(*) FILTER (WHERE deleted_at IS NULL AND image_data IS NOT NULL) AS active_with_db_bytes,
	COUNT(*) FILTER (WHERE deleted_at IS NULL AND COALESCE(blob_identifier, '') <> '') AS active_with_blob_identifier,
	COUNT(*) FILTER (WHERE deleted_at IS NULL AND COALESCE(blob_identifier, '') <> '' AND COALESCE(blob_extension, '') <> '') AS active_with_blob_identifier_and_ext
FROM img
GROUP BY provider
ORDER BY provider;
" || true

	run_sql "
WITH img AS (
	SELECT
		COALESCE(NULLIF(storage_provider, ''), 'postgres') AS provider,
		image_data IS NOT NULL AS has_db_bytes,
		COALESCE(blob_identifier, '') <> '' AS has_blob_id,
		deleted_at
	FROM animal_images
)
SELECT
	provider,
	COUNT(*) FILTER (WHERE deleted_at IS NULL AND has_db_bytes AND has_blob_id) AS inconsistent_db_and_blob,
	COUNT(*) FILTER (WHERE deleted_at IS NULL AND provider = 'azure' AND has_db_bytes) AS inconsistent_azure_has_db_bytes,
	COUNT(*) FILTER (WHERE deleted_at IS NULL AND provider = 'postgres' AND has_blob_id) AS inconsistent_postgres_has_blob_id
FROM img
GROUP BY provider
ORDER BY provider;
" || true
else
	echo "Table public.animal_images not found; skipping."
fi

section "Protocol documents (animals)"

# Only run if the table exists.
if run_sql "SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='animals';" | grep -q 1; then
	# Only run protocol-related queries if the expected columns exist.
	if run_sql "SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='animals' AND column_name='protocol_document_provider';" | grep -q 1; then
		run_sql "
WITH docs AS (
	SELECT
		COALESCE(NULLIF(protocol_document_provider, ''), 'postgres') AS provider,
		protocol_document_url,
		protocol_document_data,
		protocol_document_blob_identifier,
		protocol_document_blob_extension
	FROM animals
)
SELECT
	provider,
	COUNT(*) FILTER (WHERE COALESCE(protocol_document_url, '') <> '') AS animals_with_doc_url,
	COUNT(*) FILTER (WHERE COALESCE(protocol_document_url, '') <> '' AND protocol_document_data IS NOT NULL) AS with_db_bytes,
	COUNT(*) FILTER (WHERE COALESCE(protocol_document_url, '') <> '' AND COALESCE(protocol_document_blob_identifier, '') <> '') AS with_blob_identifier,
	COUNT(*) FILTER (WHERE COALESCE(protocol_document_url, '') <> '' AND COALESCE(protocol_document_blob_identifier, '') <> '' AND COALESCE(protocol_document_blob_extension, '') <> '') AS with_blob_identifier_and_ext
FROM docs
GROUP BY provider
ORDER BY provider;
" || true

		run_sql "
WITH docs AS (
	SELECT
		COALESCE(NULLIF(protocol_document_provider, ''), 'postgres') AS provider,
		COALESCE(protocol_document_url, '') <> '' AS has_url,
		protocol_document_data IS NOT NULL AS has_db_bytes,
		COALESCE(protocol_document_blob_identifier, '') <> '' AS has_blob_id
	FROM animals
)
SELECT
	provider,
	COUNT(*) FILTER (WHERE has_url AND has_db_bytes AND has_blob_id) AS inconsistent_db_and_blob,
	COUNT(*) FILTER (WHERE has_url AND provider = 'azure' AND has_db_bytes) AS inconsistent_azure_has_db_bytes,
	COUNT(*) FILTER (WHERE has_url AND provider = 'postgres' AND has_blob_id) AS inconsistent_postgres_has_blob_id
FROM docs
GROUP BY provider
ORDER BY provider;
" || true
	else
		echo "Column animals.protocol_document_provider not found; skipping protocol document checks."
	fi
else
	echo "Table public.animals not found; skipping."
fi

az_blob_list_count() {
	local account="$1"
	local container="$2"
	local prefix="$3"
	local max_results="${4:-5000}"  # Limit to prevent timeouts on large containers

	have_cmd az || die "Azure CLI (az) not found"
	local py
	py="$(python_cmd)" || die "python not found (needed to parse az json output)"

	# Prefer RBAC auth when possible.
	if [[ -n "${AZURE_STORAGE_ACCOUNT_KEY:-}" ]]; then
		az storage blob list \
			--account-name "$account" \
			--account-key "$AZURE_STORAGE_ACCOUNT_KEY" \
			--container-name "$container" \
			--prefix "$prefix" \
			--num-results "$max_results" \
			--output json \
			| "$py" -c 'import json,sys; print(len(json.load(sys.stdin)))'
	else
		az storage blob list \
			--auth-mode login \
			--account-name "$account" \
			--container-name "$container" \
			--prefix "$prefix" \
			--num-results "$max_results" \
			--output json \
			| "$py" -c 'import json,sys; print(len(json.load(sys.stdin)))'
	fi
}

az_blob_exists() {
	local account="$1"
	local container="$2"
	local blob_name="$3"

	have_cmd az || die "Azure CLI (az) not found"
	local py
	py="$(python_cmd)" || die "python not found (needed to parse az json output)"

	if [[ -n "${AZURE_STORAGE_ACCOUNT_KEY:-}" ]]; then
		az storage blob exists \
			--account-name "$account" \
			--account-key "$AZURE_STORAGE_ACCOUNT_KEY" \
			--container-name "$container" \
			--name "$blob_name" \
			--output json \
			| "$py" -c 'import json,sys; print("true" if json.load(sys.stdin).get("exists") else "false")'
	else
		az storage blob exists \
			--auth-mode login \
			--account-name "$account" \
			--container-name "$container" \
			--name "$blob_name" \
			--output json \
			| "$py" -c 'import json,sys; print("true" if json.load(sys.stdin).get("exists") else "false")'
	fi
}

section "Terraform / Azure discovery"
echo "Terraform dir: $tf_dir"
echo "Storage account: ${AZURE_STORAGE_ACCOUNT_NAME:-<unset>}"
echo "Storage container: ${AZURE_STORAGE_CONTAINER_NAME:-<unset>}"
echo "Resource group (tf output): ${TF_RESOURCE_GROUP_NAME:-<unset>}"
echo "Container App ID (tf output): ${TF_CONTAINER_APP_ID:-<unset>}"

if have_cmd az && [[ -n "${TF_RESOURCE_GROUP_NAME:-}" && -n "${TF_CONTAINER_APP_ID:-}" ]]; then
	# Derive container app name from its ARM ID.
	ca_name="${TF_CONTAINER_APP_ID##*/}"
	echo "Container App name (derived): $ca_name"
	# Show high-level env var wiring without exposing secrets.
	if az containerapp show -g "$TF_RESOURCE_GROUP_NAME" -n "$ca_name" --query "properties.template.containers[0].env[].{name:name,value:value,secretRef:secretRef}" -o table 2>/dev/null; then
		:
	else
		echo "(Azure CLI available but unable to query container app env; check az login/permissions.)"
	fi
else
	echo "(Skipping container app env inspection; requires az + TF outputs.)"
fi

if [[ "$do_az_check" == "auto" ]]; then
	if have_cmd az && [[ -n "${AZURE_STORAGE_ACCOUNT_NAME:-}" && -n "${AZURE_STORAGE_CONTAINER_NAME:-}" ]]; then
		do_az_check="true"
	else
		do_az_check="false"
	fi
fi

if [[ "$do_az_check" == "true" ]]; then
	section "Azure blob verification"
	[[ -n "${AZURE_STORAGE_ACCOUNT_NAME:-}" ]] || die "AZURE_STORAGE_ACCOUNT_NAME is required for --az-check"
	[[ -n "${AZURE_STORAGE_CONTAINER_NAME:-}" ]] || die "AZURE_STORAGE_CONTAINER_NAME is required for --az-check"

	echo "Listing blob counts (may take a moment)..."
	img_count="$(az_blob_list_count "$AZURE_STORAGE_ACCOUNT_NAME" "$AZURE_STORAGE_CONTAINER_NAME" "images/animals/")"
	doc_count="$(az_blob_list_count "$AZURE_STORAGE_ACCOUNT_NAME" "$AZURE_STORAGE_CONTAINER_NAME" "documents/protocols/")"
	echo "Azure blobs under images/animals/: $img_count"
	echo "Azure blobs under documents/protocols/: $doc_count"

	section "Spot-check DB blob identifiers exist in Azure (sample=${sample_n})"
	local_py="$(python_cmd)" || die "python not found"

	# Images
	img_ids="$(run_sql_tuples "
SELECT blob_identifier || COALESCE(blob_extension, '')
FROM animal_images
WHERE deleted_at IS NULL
	AND COALESCE(NULLIF(storage_provider, ''), 'postgres') = 'azure'
	AND COALESCE(blob_identifier, '') <> ''
ORDER BY id DESC
LIMIT ${sample_n};
" | sed -e 's/^ *//' -e 's/ *$//' | grep -v '^$' || true)"

	if [[ -n "$img_ids" ]]; then
		while IFS= read -r identifier; do
			[[ -n "$identifier" ]] || continue
			exists="$(az_blob_exists "$AZURE_STORAGE_ACCOUNT_NAME" "$AZURE_STORAGE_CONTAINER_NAME" "images/animals/${identifier}")"
			echo "images/animals/${identifier}: ${exists}"
		done <<< "$img_ids"
	else
		echo "No azure-stored images found in DB to check."
	fi

	# Documents
	doc_ids="$(run_sql_tuples "
SELECT protocol_document_blob_identifier || COALESCE(protocol_document_blob_extension, '')
FROM animals
WHERE COALESCE(NULLIF(protocol_document_provider, ''), 'postgres') = 'azure'
	AND COALESCE(protocol_document_blob_identifier, '') <> ''
ORDER BY id DESC
LIMIT ${sample_n};
" | sed -e 's/^ *//' -e 's/ *$//' | grep -v '^$' || true)"

	if [[ -n "$doc_ids" ]]; then
		while IFS= read -r identifier; do
			[[ -n "$identifier" ]] || continue
			exists="$(az_blob_exists "$AZURE_STORAGE_ACCOUNT_NAME" "$AZURE_STORAGE_CONTAINER_NAME" "documents/protocols/${identifier}")"
			echo "documents/protocols/${identifier}: ${exists}"
		done <<< "$doc_ids"
	else
		echo "No azure-stored protocol docs found in DB to check."
	fi
else
	section "Azure blob verification"
	echo "Skipping Azure CLI checks. Use --az-check to force, or ensure az + TF outputs are available."
fi

echo
echo "Done."

