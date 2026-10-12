# aprs-android-hook

<!-- impeccable:product-schema 1 -->

## Platform
web

## Stack
Go standard-library HTTP server, SQLite, SvelteKit static frontend based on Nigh/svelte-static-pwa-template. One Docker container, external Caddy HTTPS proxy.

## Users and purpose
Self-hosting aprs-android users privately review their own GPS history. One account per deployment. Android-generated bearer tokens authorize uploads independently of browser sessions.

## Capabilities and constraints
Date and device filters, paginated track list, map, distance and recorded-duration totals. Separate reporting ID/device streams; gaps over 30 minutes split tracks. Raw fixes persist for future changes. Duration includes stops within a segment. Date boundaries use the deployment timezone, default Asia/Shanghai.

## Brand commitments
Use Nigh/xianii-theme tokens and interaction constraints. Chinese interface and user documentation. Dark theme and OpenFreeMap dark map matching aprs-android.

## Confirmed interface
Desktop: filters and totals above a track list and map side by side. Mobile: list and map stacked. Build directly in code; inspect desktop and mobile. No generated visual mockups required.

## Scope
No multiple users, export, motion detection, or speculative start/end upload markers. Existing Android webhook payload is the source of truth.
