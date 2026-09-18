# Start a throwaway PostgreSQL for local product paths (dev.sh, QA, snapshots).
# Source it:  . scripts/ephemeral-postgres.sh
# If DATABASE_URL is already set (CI), this is a no-op.
# Sets DATABASE_URL, DB_BACKEND, HAUSV_EPHEMERAL_PG_CONTAINER.
# Call hausv_ephemeral_postgres_stop from the caller's cleanup.
#
# Targets bash 3.2 (macOS /bin/bash).
#
# Image pin matches .github/workflows/ci.yml.

hausv_postgres_image='postgres:17.6-alpine3.22@sha256:ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94'

hausv_ephemeral_postgres_stop() {
    if [ -n "${HAUSV_EPHEMERAL_PG_CONTAINER:-}" ]; then
        docker rm -f "$HAUSV_EPHEMERAL_PG_CONTAINER" >/dev/null 2>&1 || true
        HAUSV_EPHEMERAL_PG_CONTAINER=""
    fi
}

if [ -n "${DATABASE_URL:-}" ]; then
    export DB_BACKEND=postgres
    return 0 2>/dev/null || true
fi

if ! command -v docker >/dev/null 2>&1; then
    echo "PostgreSQL is required. Install Docker or set DATABASE_URL." >&2
    return 1 2>/dev/null || exit 1
fi

hausv_pg_name="hausv-pg-$$"
# Prefer an explicit port; otherwise let Docker pick a loopback port.
if [ -n "${HV_DEV_PG_PORT:-}" ]; then
    hausv_pg_publish="127.0.0.1:${HV_DEV_PG_PORT}:5432"
else
    hausv_pg_publish="127.0.0.1:0:5432"
fi

HAUSV_EPHEMERAL_PG_CONTAINER=$(
    docker run -d --rm \
        --name "$hausv_pg_name" \
        -e POSTGRES_DB=hausv \
        -e POSTGRES_USER=hausv \
        -e POSTGRES_PASSWORD=hausv-dev \
        -p "$hausv_pg_publish" \
        "$hausv_postgres_image"
) || {
    echo "Could not start PostgreSQL ($hausv_postgres_image)." >&2
    return 1 2>/dev/null || exit 1
}
export HAUSV_EPHEMERAL_PG_CONTAINER

hausv_pg_ready=0
for _ in $(seq 40); do
    if docker exec "$HAUSV_EPHEMERAL_PG_CONTAINER" pg_isready -U hausv -d hausv >/dev/null 2>&1; then
        hausv_pg_ready=1
        break
    fi
    sleep 0.25
done
if [ "$hausv_pg_ready" -eq 0 ]; then
    echo "PostgreSQL did not become ready." >&2
    hausv_ephemeral_postgres_stop
    return 1 2>/dev/null || exit 1
fi

hausv_pg_port=$(docker port "$HAUSV_EPHEMERAL_PG_CONTAINER" 5432/tcp | awk -F: 'END { print $NF }')
if [ -z "$hausv_pg_port" ]; then
    echo "Could not read the published PostgreSQL port." >&2
    hausv_ephemeral_postgres_stop
    return 1 2>/dev/null || exit 1
fi

export DB_BACKEND=postgres
export DATABASE_URL="postgres://hausv:hausv-dev@127.0.0.1:${hausv_pg_port}/hausv?sslmode=disable"
unset hausv_pg_name hausv_pg_publish hausv_pg_ready hausv_pg_port hausv_postgres_image
