#!/usr/bin/env sh

if ! tail -f "$HOME/.config/gh-ci/ci.log"; then
    echo "error: tail ci.log"
    exit 1
fi
