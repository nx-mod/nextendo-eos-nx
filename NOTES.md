# nextendo-eos-nx - notes

## Implemented (tested)
- /auth/v1/oauth/token, /epic/oauth/v2/token: client + user access tokens.
- /connect/v1/oauth/token: external identity -> Product User Id (PUID), stable.
- /lobby/v1/lobbies (create/find) + /lobby/v1/lobbies/{id}/join, in memory.
- Catch-all: records + 200s every other endpoint (dashboard `unhandled`).

## UNCONFIRMED - needs a capture (use the `unhandled` list)
1. Exact token grant params + response fields per EOS SDK version, and whether
   tokens must be JWTs the client validates (these are opaque bearer strings).
2. Real lobby/session paths and bodies (RTC, attributes, member updates), and
   the matchmaking/sessions API (/matchmaking/v1 or /sessions/v1).
3. How the external identity maps in (NSA id_token via connect) and any
   client-id / deployment-id / sandbox-id gating.
4. P2P relay (EOS P2P / RTC) - not modelled; the game may fall back to its own.

## Per game
Set EOS_DEPLOYMENT_ID and any client ids on a game branch (see `fall-guys`).

## fall-guys branch
- fallguys.go defaults EOS_DEPLOYMENT_ID to a placeholder and names the show
  lobby bucket (fallguys:show).
- TO CONFIRM from a Fall Guys client/capture: the real EOS deployment id,
  sandbox id and client id; the connect external-auth flow (NSA id_token); and
  the matchmaking/session endpoints Fall Guys actually calls (watch the
  dashboard `unhandled` list against a live game).
