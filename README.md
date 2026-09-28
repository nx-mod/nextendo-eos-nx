# nextendo-eos-nx

*(still in alpha testing)*

**A new service implementation by nx-mod** for the Nextendo Network.

An **Epic Online Services (EOS)** backend for [Nextendo Network](https://nextendo.network), so EOS-based cross-play titles (Fall Guys and other Unreal games) work on the stack instead of Epic's servers. Source only. Not affiliated with Epic Games.

## Why

EOS is the backend beyond NEX/NPLN (Nintendo) and Demonware (Activision) — a huge slice of third-party and cross-play Switch games run on it. A game gets a client token, authenticates a user, is issued a **Product User Id (PUID)**, then creates/joins lobbies and sessions.

## What it serves

```
POST /auth/v1/oauth/token       client-credentials / external -> access token
POST /epic/oauth/v2/token       Epic account services user token
POST /connect/v1/oauth/token    external identity (e.g. NSA id) -> PUID + token
POST /lobby/v1/lobbies          create a lobby (Bearer PUID)
GET  /lobby/v1/lobbies?bucket=  find lobbies
POST /lobby/v1/lobbies/{id}/join
```

Tokens and PUIDs are stable per external identity. **Every other EOS endpoint is recorded and 200s** (a capture surface, shown on the dashboard's `unhandled` list) so a game keeps going and the missing surface is visible.

## Branches

- `main` — the game-independent EOS core.
- `fall-guys` — Fall Guys deployment/client specifics on top of the core.

## Run

```sh
go build -o server .   # Go 1.23+, stdlib only
go test ./...
./server
```

| setting | default | meaning |
|---|---|---|
| `EOS_PORT` | 8476 | HTTP(S) port |
| `EOS_DEPLOYMENT_ID` | — | the game's EOS deployment (per game / branch) |
| `DASH_PORT`/`DASH_TOKEN` | 8105 | `/api/stats` (incl. `unhandled`), `/healthz` |

## Status

The auth → connect → lobby core is implemented and tested. The exact EOS request/response bodies vary by SDK version and deployment; these are the sensible shapes, and the `unhandled` recorder maps what a real title actually calls. See NOTES.md.

## Credits

- **[Nextendo Network](https://nextendo.network)** — the stack this plugs into.
- **[Epic Online Services documentation](https://dev.epicgames.com/docs/game-services)** — the EOS auth/connect/lobby model (public docs).

Protocol facts were read and reimplemented; no code was copied.

## Credits

Built by nx-mod for the **Nextendo Network**, on the work of the Nextendo Network team — https://nextendo.network. Nextendo is awesome.
