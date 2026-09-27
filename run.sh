#!/usr/bin/env sh
# nextendo-eos-nx — home-lab launch (Epic Online Services backend). Serves plain
# HTTP (TLS via the sni-router); in-memory auth/lobby, runs straight from repo.
# For the Fall Guys deployment specifics, use the fall-guys branch.
set -e
echo "[eos] starting on default ports (EOS_PORT 8476, DASH_PORT 8105)"
exec go run .
