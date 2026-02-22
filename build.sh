#!/usr/bin/env sh

set -e

EXTENSION="ci"
VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo 'dev')}"

build_target() {
    GOOS=$1
    GOARCH=$2
    EXT=$3
    OUT="gh-${EXTENSION}-${GOOS}-${GOARCH}${EXT}"
	echo "$OUT building..."
    env GOOS="$GOOS" GOARCH="$GOARCH" \
        go build \
        -ldflags="-s -w -X main.version=$VERSION" \
        -o "${OUT}" .
}

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

go mod download

pids=""
targets "$@"

fail=0
for pid in $pids; do
    wait "$pid" || fail=1
done

exit $fail
