#!/usr/bin/env sh
# Example: the EOS flow a Fall Guys-style title uses - client token, connect to a
# Product User Id, then create / find / join a lobby.
set -e
BASE="${1:-http://localhost:8476}"
echo "== client credentials =="
curl -s -X POST -d 'grant_type=client_credentials' "$BASE/auth/v1/oauth/token"; echo
echo "== connect (external id -> PUID) =="
HOST=$(curl -s -X POST -d 'external_auth_token=host-1' "$BASE/connect/v1/oauth/token")
echo "$HOST"
TOKEN=$(printf '%s' "$HOST" | sed -n 's/.*access_token":"\([^"]*\).*/\1/p')
echo "== create a lobby =="
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"bucketId":"fallguys:show","maxMembers":40}' "$BASE/lobby/v1/lobbies"; echo
echo "== find lobbies in that bucket =="
curl -s "$BASE/lobby/v1/lobbies?bucket=fallguys:show"; echo
