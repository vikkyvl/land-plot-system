# Arqetra web app

Arqetra is the web client of the Land Plot System: a Vue 3 and TypeScript single-page app that lets a user draw a land plot on a map, see its area from the server, save it, follow the saga that confirms it, keep the version history and build reports. It is responsive (phone, tablet, desktop) and installable as a PWA.

The app calculates nothing itself. Area, perimeter and polygon validation come from `POST /api/v1/plots/preview` and the plot endpoints of the gateway; the client only formats the numbers and draws the map.

## Requirements

- Node.js 24 or newer and npm.
- The backend from the repository root running (`docker compose up -d --build`): the gateway on `http://localhost:8080` and Keycloak on `http://localhost:8180`.

## Development

```sh
cd apps/web
npm ci
npm run dev        # http://localhost:5173
```

The dev server talks to the gateway directly; the gateway allows the origin `http://localhost:5173` in CORS and Keycloak accepts its `/auth/callback`. Sign in with a test user from the realm (for example `taras.melnyk@example.com`, the password is `TARAS_MELNYK_PASSWORD` in the root `.env`), register a new user or use Google if the identity provider is configured.

### Configuration

The settings are `VITE_*` variables. They are built into the bundle, so they must never hold secrets. Every variable has a local default, and `config/env.ts` checks that the URLs are valid. To override them, copy `.env.example` to `.env.local`.

| Variable | Default | Meaning |
|----------|---------|---------|
| `VITE_GATEWAY_URL` | `http://localhost:8080` | API gateway |
| `VITE_OIDC_AUTHORITY` | `http://localhost:8180/realms/land-plot` | Keycloak realm |
| `VITE_OIDC_CLIENT_ID` | `land-plot-web` | public OIDC client (code flow with PKCE) |
| `VITE_GEOCODING_URL` | `https://nominatim.openstreetmap.org` | place search |
| `VITE_MAP_TILES_URL` | OpenStreetMap tiles | map layer |
| `VITE_SATELLITE_TILES_URL` | Esri World Imagery tiles | satellite layer |

## Checks

```sh
npm run lint    # ESLint, including the layer boundaries
npm run test    # Vitest
npm run build   # type check with vue-tsc and production build
```

From the repository root `make web-check` runs the three commands (and `npm ci` when the lock file changed), and `make check-all` runs the Go checks as well.

Tests never call the real gateway or Keycloak: clients get a fake `fetch`, services get fake ports, and timers are faked for the preview delay and the saga polling.

## Structure

The app repeats the layers of the Go services:

```
src/
├── domain/      types and pure functions: Plot, Report, Coordinate, Profile, DomainError, GeoJSON export,
│                survey (capture and obstructed sides), accuracy, terrain and climate
├── port/        interfaces the services need: PlotGateway, ReportGateway, ProfileGateway,
│                GeocodingGateway, AuthSession, DraftStore, PreferencesStore
├── service/     use cases: calculator, sessions, plots, reports, place search, saga confirmation
├── client/      HTTP adapters for the gateway and Nominatim, error mapping to DomainError
├── auth/        Keycloak sign-in with oidc-client-ts
├── repository/  guest draft in IndexedDB, settings (map layer, language) in localStorage
├── config/      checked VITE_* settings
├── ui/          router, views, components, Pinia stores, composables, styles, design tokens,
│                i18n setup and texts (locales/uk.json, locales/en.json)
└── bootstrap/   composition root: builds the adapters and services and provides them to the UI
```

`ui → service → port ← client / repository / auth`, and `domain` depends on nothing. `eslint-plugin-boundaries` enforces these rules, so a forbidden import fails `npm run lint`; `tests/boundaries.test.ts` checks that it does. `domain`, `port` and `service` may only use relative imports, so no Vue, HTTP or browser library gets into them.

## Languages

The app speaks Ukrainian and English through `vue-i18n`. Every text lives in `src/ui/locales/uk.json` and `src/ui/locales/en.json`; components use keys only, and the ESLint rule `vue/no-bare-strings-in-template` fails `npm run lint` on raw text or a raw `aria-label`, `title`, `placeholder` or `alt` in a template.

- The first language comes from `navigator.language`: Ukrainian for `uk*`, English otherwise. The **УК / EN** switch in the header (and in the avatar menu on phones) changes it, and `SettingsService` keeps the choice in `localStorage` through the `SettingsStore` port.
- Switching also sets `lang` on `<html>` and the number and date formats (`Intl` with `uk-UA` or `en-GB`): `2,48 га` and `27.10.2026`, `2.48 ha` and `27/10/2026`. Ukrainian counts have three plural forms.
- Sign-in, registration and Google sign-in pass `ui_locales` with the current language, so Keycloak shows its pages in the same language.
- Server error codes have texts in both languages; the code itself is shown unchanged.
- Plot and report names are user data and are not translated. The PWA manifest stays Ukrainian.

Tests check that both files have the same keys and parameters, that the lint rule fails on raw text, and that switching the language changes the texts, the number and date formats and `lang`.

To add a text, add the same key to both files and use `$t('key')` in a template or `t('key')` from `ui/i18n.ts` in code.

## Behaviour worth knowing

- **Sign-in.** Authorization Code with PKCE through `oidc-client-ts`. Tokens are kept in `sessionStorage`, never in `localStorage`, and renewed with the refresh token. The app never sees the password: login, registration (`prompt=create`) and Google (`kc_idp_hint=google`) happen on Keycloak pages. On `401` the client renews the token once (parallel requests share that renewal) and otherwise sends the user to `/login?redirect=…`.
- **Preview.** The calculator asks the server 300 ms after a point is added, removed, inserted or dropped after dragging, never while a vertex is being dragged, and only from three points. A self-intersection is drawn in red with a button that swaps the points named by the server. On `429` the panel counts down the seconds from `Retry-After`.
- **Guest draft.** The points and the name are kept in IndexedDB. When a guest saves, the draft is marked, the user signs in and the plot is saved automatically after returning.
- **Saga.** A new or changed plot is `PENDING` until `report-service` confirms it. The app checks the plot every 2 seconds for at most 60 seconds of visible time, pauses while the tab is hidden, runs one check per plot, and reports confirmation, cancellation (`404`) or a rolled-back change.
- **Area margin.** The calculator keeps for every point the smallest zoom at which it was placed or moved, and the layer and pointer (mouse, touch, pen) of the last change, and sends them as `capture` with the marked sides as `obstructions` in preview, create and update. So zooming in only at the end does not make the margin smaller; moving a point again at a closer zoom keeps its smallest zoom. The area reads «2,48 га ± 0,05 га» with the main source of the margin, the possible ground shift and a hint to zoom in when the map scale adds the most. A plot saved without a capture shows «≈ ± …» as approximate with a hint to change the boundary. The server calculates the margin; the app only formats it.
- **Obstructed sides.** A tap or click on a side or its midpoint opens a menu (a popover on the web, a bottom sheet on phones) with «+ Додати точку тут» first and the reasons the boundary is hidden: trees, buildings and shadows, hard to see, or nothing. A marked side is drawn dashed with its reason. The points card lists every side with a select for keyboard and screen reader users. Marks follow their sides when points are inserted, undone or swapped, and a mark disappears with its side.
- **Reports.** A report shows the boundary of the plot version it was built for. GeoJSON is generated in the browser from the report and that boundary. The blocks «Точність вимірювання», «Рельєф» (with «Оцінка груба» for plots under 1 ha and a note that the cadastral area is the horizontal projection) and «Клімат» (a table on the web, cards on phones) come from the report snapshot. A block whose source did not answer says it is temporarily unavailable and offers «Оновити звіт», which creates a new report; an older report without the blocks says there is no data. The data sources are credited under the blocks: Open-Meteo.com (CC BY 4.0), Copernicus C3S ERA5, Copernicus DEM GLO-90.

## Docker

The `Dockerfile` builds the app with `node:26.12.1-alpine` and serves `dist` with `nginx:1.30.5-alpine` as the unprivileged `nginx` user. From the repository root:

```sh
docker compose up -d --build web    # http://localhost:4173
```

The service waits for the gateway and Keycloak to be healthy and has its own health check (`/health`). `VITE_*` values are passed as build arguments, and the variables in the root `.env.example` change them:

| Variable | Used for |
|----------|----------|
| `WEB_PORT` | host port, `4173` by default (the origin allowed in CORS and in Keycloak) |
| `WEB_GATEWAY_URL`, `WEB_OIDC_AUTHORITY`, `WEB_OIDC_CLIENT_ID`, `WEB_GEOCODING_URL`, `WEB_MAP_TILES_URL`, `WEB_SATELLITE_TILES_URL` | build arguments for the `VITE_*` settings |
| `WEB_CSP_GATEWAY_ORIGIN`, `WEB_CSP_KEYCLOAK_ORIGIN`, `WEB_CSP_GEOCODING_ORIGIN`, `WEB_CSP_TILE_ORIGINS` | origins allowed by the Content Security Policy |

nginx serves every unknown path with `index.html` (`try_files $uri /index.html`), sends `index.html`, `sw.js` and the manifest with `Cache-Control: no-cache` and hashed files under `/assets/` with `immutable` for a year. The headers are set at start-up from `deploy/nginx/web.conf.template` through `envsubst`:

- `Content-Security-Policy`: scripts, styles, fonts and workers only from the app itself; `connect-src` also allows the gateway, Keycloak, Nominatim and the tile servers; images also allow the tile servers; `frame-ancestors 'none'`, `object-src 'none'`;
- `X-Content-Type-Options: nosniff`;
- `Referrer-Policy: strict-origin-when-cross-origin`.

When the gateway or Keycloak moves to another address, change the build arguments and the CSP origins together.

## PWA

`vite-plugin-pwa` generates the manifest (Arqetra, standalone, theme colour `#2F6B4F` from `--color-primary`, icons 192, 512 and maskable, an Apple touch icon) and the service worker. The service worker:

- precaches the app shell and only the latin and cyrillic Montserrat fonts;
- keeps up to 200 map tiles (OpenStreetMap and Esri), least recently used first out;
- never caches the API, Keycloak or Nominatim, and lets `/auth/callback` reach the network, so signing in and out work with the worker active.

`tests/pwa.test.ts` checks these rules in `pwa.config.ts`. Without a network the app opens from the cache and shows the banner «Немає з'єднання»; the API features come back with the connection.

The service worker runs only in a production build. To try it without Docker:

```sh
npm run build
npm run preview    # http://localhost:4173
```

### Installing the app

- **Android (Chrome):** open the site, menu **⋮ → Install app** (or **Add to Home screen**).
- **iPhone and iPad (Safari):** open the site, **Share → Add to Home Screen**.
- **Desktop (Chrome, Edge):** the install icon at the right of the address bar, or menu **→ Install Arqetra**.

Chrome and Safari install only from `https://` or `localhost`. To check the manifest and the worker, open DevTools → **Application → Manifest** and **Service workers**.
