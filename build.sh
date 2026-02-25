#!/usr/bin/env sh

set -e

EXTENSION="ci"
TS_STRING="$(date -u +"%Y%m%dT%H%M%SZ")"
FALLBACK_VERSION="dev-${TS_STRING}"
VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo "$FALLBACK_VERSION")}"

ts() { 
    date -u +"%Y%m%dT%H%M%SZ"
}
info() { 
    echo "ℹ️ [$(ts)] INF: $*"
}
error() { 
    echo "❌ [$(ts)] ERR: $*" >&2
}
success() { 
    echo "✅ [$(ts)] YAY: $*"
}

build_target() {
    GOOS=$1
    GOARCH=$2
    EXT=$3
    OUT="gh-${EXTENSION}-${GOOS}-${GOARCH}${EXT}"
    info "$OUT building..."
    # strip symbols to minimize binary size, and override default version "dev" with git tag
    if ! env GOOS="$GOOS" GOARCH="$GOARCH" \
    go build \
        -ldflags "-s -w -X github.com/turkosaurus/gh-ci/internal/ui.Version=${VERSION}" \
        -o "${OUT}" .; then
        error "${OUT} FAILED"
        return 1 
    else
        success "$✅ {OUT} built successfully!"
    fi
}

# Build all the platforms, because a TUI client
# could be running on anything.
targets() {
    set -- \
        linux amd64 '' \
        linux 386 '' \
        linux arm '' \
        linux arm64 '' \
        windows amd64 '.exe' \
        windows 386 '.exe' \
        darwin amd64 '' \
        darwin arm64 ''
    while [ $# -gt 0 ]; do
        build_target "$1" "$2" "$3" &
        pids="$pids $!"
        shift 3
    done
}

# Download once, then build many times.
info "fetching dependencies..."
go mod download

pids=""
targets "$@"

fail=0
for pid in $pids; do
    wait "$pid" || fail=1
done

if [ "$fail" -ne 0 ]; then
    error "build failed"
else
    success "all builds succeeded!"
fi
exit $fail
