# aprs-android-hook

<!-- impeccable:product-schema 1 -->

## Platform
web

## Stack
Go standard-library HTTP server, SQLite, SvelteKit static frontend based on Nigh/svelte-static-pwa-template. One Docker container, external Caddy HTTPS proxy.

## Users and purpose
Self-hosting aprs-android users privately review their own GPS history. One account per deployment. Android-generated bearer tokens authorize uploads independently of browser sessions.

## Capabilities and constraints
Date and device filters, paginated track list, map, distance and recorded-duration totals. Tracks group points by reporting ID, device, and Android-generated session ID, independent of gaps or marker delivery. Every location and test packet requires a canonical lowercase UUID v4 session ID. Authenticated test packets never persist locations or update upload times. Raw fixes persist for future changes. Duration includes stops within a segment. Date boundaries use the deployment timezone, default Asia/Shanghai.

## Brand commitments
Use Nigh/xianii-theme tokens and interaction constraints. English-default interface and README, with Simplified Chinese available. Extensible locale registry and per-language catalogs; browser-local language preference independent of deployment timezone. Dark theme and OpenFreeMap dark map matching aprs-android.

## Confirmed interface
Desktop: filters and totals above a track list and map side by side. Mobile: list and map stacked. Build directly in code; inspect desktop and mobile. No generated visual mockups required.

## Scope
No multiple users, export, or motion detection. Every upload requires an explicit location/test type; no legacy payload compatibility or database migrations. Android session ID generation is outside this repository.
