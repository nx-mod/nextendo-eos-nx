# nextendo-eos-nx — example usage

`./run.sh` starts the Epic Online Services backend (:8476). EOS titles (Fall Guys
and other Unreal games) authenticate, get a Product User Id, then create/join
lobbies.

```sh
./run.sh &
./test-eos.sh           # client token -> connect PUID -> create + find a lobby
```

Fall Guys deployment specifics live on the `fall-guys` branch. Unimplemented EOS
endpoints are recorded and 200'd (capture surface) — see the dashboard.
