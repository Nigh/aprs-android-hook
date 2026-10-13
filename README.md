# aprs-android-hook

**English** | [简体中文](README.zh-CN.md)

A self-hosted webhook server for [aprs-android](https://github.com/Nigh/aprs-android). Receive location updates from your phone, filter journeys by date, review distance and recorded duration, and inspect tracks on an OSM map.

Each deployment has one account and supports up to 16 phone upload tokens. Go + SQLite + Svelte run as a single service, without a separate database server. The interface follows [xianii-theme](https://github.com/Nigh/xianii-theme) and uses the same OpenFreeMap dark map style as the Android app.

The web interface supports English and Simplified Chinese, with English as the default. Use the language selector at the top of the page; your choice is saved in this browser. Language does not change the deployment timezone or stored data.

## Deploy with Docker Compose

Requires Git, Docker, and Docker Compose v2. The first deployment builds the image from source.

```sh
git clone https://github.com/Nigh/aprs-android-hook.git
cd aprs-android-hook
cp .env.example .env
docker compose up -d --build
```

By default, the HTTP service is available only at `127.0.0.1:18080` on the host. Open `http://localhost:18080` and create the only account. Passwords require at least 12 characters and at most 72 bytes. **Register locally before exposing the service publicly:** the first person to register becomes its only user.

For a remote server, open the registration page through an SSH tunnel:

```sh
ssh -L 18080:127.0.0.1:18080 user@your-server
```

Then open `http://localhost:18080` on your own computer. Do not run `docker compose down -v`; it deletes the volume containing your account and tracks.

### Change the port and timezone

Edit `.env`, then run `docker compose up -d`:

```dotenv
PORT=18081
BIND_ADDRESS=127.0.0.1
TZ=Asia/Shanghai
PUBLIC_URL=
```

`PORT` is the host port; the container always uses port 8080. `TZ` controls date filters and displayed timestamps and accepts IANA timezone names. The default loopback binding works with Caddy on the host.

## Provide HTTPS with Caddy

Android accepts only HTTPS webhook URLs. Point your domain's DNS at the server and make ports 80 and 443 available to Caddy. After registering locally, set this in `.env`:

```dotenv
PUBLIC_URL=https://tracks.example.com
```

`PUBLIC_URL` must match the browser's complete origin and must not include a path. It controls same-origin checks and Secure login cookies; update it when changing the external address.

For Caddy running on the host, use this Caddyfile:

```caddyfile
tracks.example.com {
    reverse_proxy 127.0.0.1:18080
}
```

If you changed `PORT`, update the upstream port too. Apply both configurations:

```sh
docker compose up -d
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

Sign in at `https://tracks.example.com`. Proxy the entire site, including `/api/webhook`, to the app. Do not add trailing-slash redirects. The app uses `PUBLIC_URL` to identify its external origin rather than trusting arbitrary client-supplied proxy headers. Caddy handles HTTP-to-HTTPS redirects. [Caddy reverse proxy documentation](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy)

### Caddy running in a container

Inside a Caddy container, `127.0.0.1` refers to Caddy itself. Connect your existing Caddy container to the app's Compose network and use its service name. Caddy does not need to be added to this project's Compose file.

The default network is `aprs-android-hook_default`; a custom Compose project name changes it:

```sh
docker network connect aprs-android-hook_default your-caddy-container
```

Use `reverse_proxy app:8080` in the Caddyfile. To keep the connection after Caddy is recreated, declare the external network in Caddy's own Compose configuration:

```yaml
services:
  caddy:
    # Keep your existing image, ports, volumes, and other configuration.
    networks:
      - default
      - aprs
networks:
  aprs:
    external: true
    name: aprs-android-hook_default
```

Do not expose the app's port publicly just to connect the containers.

## Connect aprs-android

1. On your phone, open **Settings → Webhook** in aprs-android and select **Copy token**. The phone generates the token; it survives app updates until you explicitly refresh it.
2. Sign in to the web app, open **Upload settings**, paste the phone token, and optionally add a note.
3. On your phone, turn on **Enable webhook** and enter:
   - **Webhook HTTPS URL**: `https://tracks.example.com/api/webhook`
   - **Webhook reporting ID**: a name such as `my-phone` to identify your records. This is separate from your web login username.
4. After the phone obtains and reports a reliable location, check its operation log for **Webhook HTTP 200**.
5. In the web app, open **Track history**, select the dates, and click **Show tracks**. Filter by reporting ID or device, then select a track to see the map and location details.

The phone may still upload webhook locations when stop zones or other rules block APRS transmission. The server records only the points it receives; it cannot recover points that were never uploaded because of network failures.

Refreshing a phone token does not update the server automatically. Delete the old entry and register the new token. Deletion immediately blocks subsequent uploads while keeping existing tracks. The server stores only a hash and cannot recover the full token.

### Test the webhook with curl

Replace `TOKEN` with the phone token you have registered. Do not commit real tokens or post them in public logs.

```sh
TOKEN='replace-with-your-64-character-phone-token'
SESSION_ID=$(python3 -c 'import uuid; print(uuid.uuid4())')
curl --fail-with-body https://tracks.example.com/api/webhook \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  --data "{\"type\":\"test\",\"session_id\":\"$SESSION_ID\"}"
```

A successful test returns HTTP 200 and `{"ok":true,"type":"test"}`. Check both the status and JSON response. This verifies connectivity, token authentication, and the generated session ID's format without adding a location or changing the token's last upload time. The server validates UUID format, not the quality of the client's randomness. A real location upload is still needed to verify location storage. Coordinates `(0,0)` are valid locations, not test signals.

### Packet types and recording sessions

`POST /api/webhook` requires an explicit `type` and a valid `session_id` on **every request**, including tests:

| `type` | Behavior |
| --- | --- |
| `"location"` | Store a location in the supplied recording session. |
| `"test"` | Validate the session ID and authenticate the Bearer token, then return a test response; no location fields are required or stored. |

Missing, empty, null, or unknown types return HTTP 400. `start` and `end` are not supported. A location requires `id`, `device_hash`, `session_id`, `timestamp_ms`, `latitude`, and `longitude`. Optional nullable measurements are `accuracy_m`, `speed_mps`, `bearing_deg`, and `altitude_m`. For example:

```json
{"type":"location","id":"my-phone","device_hash":"0123456789abcdef","session_id":"e0629f7c-57a5-4d47-9afc-00c3c6d4d3f9","timestamp_ms":1767225600000,"latitude":31.2304,"longitude":121.4737,"accuracy_m":5,"speed_mps":null,"bearing_deg":null,"altitude_m":null}
```

`session_id` must be a **canonical lowercase UUID v4**: exactly 36 ASCII characters, including hyphens, with version `4` and variant `8`, `9`, `a`, or `b`. The accepted pattern is:

```text
^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$
```

Missing, null, uppercase, padded, or malformed IDs return HTTP 400 with `error.sessionID`. Other JSON types (numbers, booleans, arrays, or objects) return HTTP 400 with `error.json`. IDs are not trimmed or normalized.

Android should generate one ID with `java.util.UUID.randomUUID().toString()` when a recording begins, **before acquiring GPS or attempting an upload**. Reuse that ID for every point and retry in the same session, including after network/GPS failures or channel toggles. Generate a new ID for each new recording; never recycle a previous ID. If a recording is resumed after a process restart, persist and restore its ID; if a new recording is started, generate a new one. A standalone manual report gets a new ID; a manual report belonging to an active recording reuses that recording's ID. Generate a fresh random ID for each `test` request using the same generator, without replacing the active recording ID. The example UUID above is illustrative; do not hardcode it in Android.

Location uploads return `{"ok":true}`. Deduplication uses `(id, device_hash, session_id, timestamp_ms)`; retries must preserve these fields. Points from different sessions can share a timestamp. Session IDs identify recordings, not authorization: a registered Bearer token is still required, and changing the token does not create a new session.

Android must implement this protocol to upload successfully. Updating the server alone does not generate session IDs on the phone. This server targets a fresh database; no earlier payload or database schema is supported.

## Tracks and statistics

- A track consists of all received points with the same `(id, device_hash, session_id)`, sorted by measured timestamp. Different sessions remain separate even with overlapping or identical timestamps. The same session stays together across gaps of any length; there is no 30-minute fallback or start/end state.
- Distance is the sum of spherical distances between consecutive points in each session, not a road-navigation distance. GPS drift can increase the result; this version does not automatically filter it. Missing points cannot be recovered, and a straight connection across a gap may differ from the actual route.
- Recorded duration is the sum of each session's last received timestamp minus its first received timestamp. It includes stops and gaps within a session, excludes intervals between sessions, and is not actual movement time or the exact duration between the user's start/stop actions.
- A single-point track has zero distance and duration. Different reporting IDs or devices are never connected, even if they reuse a session ID. Reusing an ID for separate recordings on the same reporting ID/device merges them.
- Date ranges include the entire end date in the displayed deployment timezone. Statistics and maps use only points inside the selected range; sessions crossing the boundary are clipped.
- Lists contain 50 tracks per page; totals cover the entire filter result. Track summaries and details expose `session_id`. Track keys identify the reporting ID/device/session and stay stable across date clipping and late uploads, although points and statistics can change.

Maps are provided by [OpenFreeMap](https://openfreemap.org/) using OpenStreetMap data. Your browser needs network access to the map service and WebGL support. Map failures do not prevent browsing the list and statistics. The PWA caches only the static app shell; private tracks cannot be queried offline. The installed PWA name and description use English; its interface follows your saved language choice.

## Back up, restore, and upgrade

The Compose `data` volume contains the SQLite database: your account, sessions, token hashes, and all location points. Keep backups in a trusted place.

Stop the app before archiving the entire volume so the database and WAL remain consistent:

```sh
docker compose stop app
docker compose run --rm --no-deps --user 0 --entrypoint tar app -C /data -czf - . > aprs-backup.tar.gz
docker compose start app
```

Restore to the same deployment or a new deployment with its data volume already created. Stop the app first; restoration overwrites files with matching names:

```sh
docker compose stop app
docker compose run --rm --no-deps --user 0 --entrypoint sh app -c 'rm -f /data/aprs.sqlite /data/aprs.sqlite-wal /data/aprs.sqlite-shm'
docker compose run --rm --no-deps -T --user 0 --entrypoint tar app -C /data -xzf - < aprs-backup.tar.gz
docker compose run --rm --no-deps --user 0 --entrypoint chown app -R 65532:65532 /data
docker compose start app
```

Confirm your backup is usable before deleting the database. These binary-redirection examples target Linux/macOS shells. On Windows, use WSL to avoid older PowerShell versions modifying archive bytes.

Back up before upgrading, then run:

```sh
git pull --ff-only
docker compose up -d --build
docker compose ps
```

There is no password recovery feature in this version. Keep your password and database backups safe.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Phone HTTP 401 | Was the complete token registered? Has the phone refreshed it? |
| Phone HTTP 400 | Are location fields, timestamps, and the session ID valid? Is the test JSON complete? |
| Web HTTP 403 | Does `PUBLIC_URL` match the browser origin, including scheme and port? |
| Signing in returns to the login page | Use Caddy's HTTPS URL when the service is configured for HTTPS. |
| No tracks | Check dates, deployment timezone, ID/device filters, and the phone's HTTP 200 log. |
| A token shows “No uploads yet” | Enable the phone webhook, set the correct URL, and obtain a reliable location. |
| Map does not appear | Check OpenFreeMap connectivity, WebGL, and the browser console. |
| Port is occupied | Change `PORT` and the Caddy upstream port together. |

Inspect status and logs with `docker compose ps` and `docker compose logs --tail=100 app`. Logs do not contain upload tokens or location request bodies.

## Local development

Requires Go 1.26.4+, Node.js 22.19+, and Corepack.

```sh
cd web
corepack enable
yarn install --immutable
yarn check
yarn lint
yarn test
yarn build
cd ..
go test ./...
go run .
```

Open `http://localhost:8080`. For frontend hot reload, run `yarn dev` in `web` from another terminal and configure the Go service with `PUBLIC_URL=http://localhost:5173`; Vite proxies `/api` to port 8080.

Environment variables: `LISTEN_ADDR` (default `:8080`), `DB_PATH` (default `data/aprs.sqlite`), `WEB_DIR` (default `web/build`), `TZ`, and `PUBLIC_URL`. Docker keeps its container port fixed; use Compose to change the host port.

### Add another UI language

Add a JSON catalog under `web/src/lib/locales`, using the keys and named placeholders from `en.json`. Import it and add an entry `{ name, messages }` to the `locales` registry in `index.js`, with a BCP 47 language tag as its key. The selector reads that registry; there is no two-language flag or hardcoded language toggle.

Messages can be strings or plural-form objects using native `Intl.PluralRules` categories, such as `one` and `other`. Missing messages fall back to English. Date and number formatting use `Intl`, while the deployment timezone remains independent. Tests check key coverage, placeholders, fallback behavior, pluralization, and registering a third language.

API errors retain the existing `error` message and also provide a stable `code` translation key. The UI translates that code and uses a generic localized error for unknown codes.

## License and credits

MIT. The frontend is based on [svelte-static-pwa-template](https://github.com/Nigh/svelte-static-pwa-template), with its [MIT license](web/LICENSE) preserved. Theme tokens come from [xianii-theme](https://github.com/Nigh/xianii-theme). Map attribution for OpenFreeMap, OpenMapTiles, and OpenStreetMap is retained.
