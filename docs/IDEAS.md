# car-search — 100 product ideas

Backlog of ideas for the product "search sightings of vehicle license plates"
(photo + time + place; search by plate; plate page with map, history and gallery; plate clarifications;
complaints; admin panel; i18n). Written against `master` at commit `994dfab` (2026-11-13).

Columns: **Layers** — `fe` / `be` / `db` / `infra` / `docs`; **Size** — S (≤ 1 day), M (2–4 days), L (week+);
**Priority** — P1 (do next), P2 (valuable, later), P3 (nice to have / research).
All ideas must be implementable without paid external API keys (OSM, open-source models, self-hosted services only).

## Already done or in progress (not counted below)

| Status | Feature |
|---|---|
| Done | Prefix search by plate words (`to_tsvector('simple')` + GIN), look-alike characters folded by `plate_search_key()` (Cyrillic/Latin, O/0, hyphen, CN dot) |
| Done | Search by accepted clarifications; distinct-per-plate results; infinite scroll + "Load more" |
| Done | Quick search dropdown with recent queries and look-alike-aware highlight |
| Done | Plate page `/plate/<id>`: all sightings of the plate, timeline, OpenStreetMap with numbered route, gallery lightbox, "See also" cards |
| Done | Plate formats RU/EU/US/UA/CN with rendering and format checks in Clarify; consensus of clarifications |
| Done | Photo complaints (reasons: wrong plate, bad photo, personal data, other) and `/admin` panel with `admin-token` |
| Done | i18n: RU/UK/EN/DE/ZH, localized addresses, settings on logo (language, theme auto/light/dark) |
| Done | Home by language: "For you" block (market of the UI language) + country tabs + "Other" (`?tab=`) |
| Done | Test data as goose migration with `is_test_data`, `include-test-data` header, blurred test photos with TEST PHOTO mark |
| Done | GitHub Actions CI: go vet / test -race with Postgres, fe tsc/build, registry check, generated-code check |
| In progress | Photo upload by users with OCR plate recognition |
| In progress | Camera requests (users ask to install/connect a camera at a location) |

## A. Data and sources (cameras, crowdsourcing, OCR)

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 1 | OCR confidence and bbox storage | Lets UI highlight the plate on the photo and moderators prioritize low-confidence reads | Add `ocr_confidence`, `plate_bbox` (jsonb) to `plate_sightings`; return in `PlateSighting`; draw a frame in lightbox | db, be, fe | M | P1 |
| 2 | Multiple OCR candidates per photo | The top-1 read is often wrong on similar glyphs; alternatives feed search and clarifications | Store top-N OCR candidates in a side table; search matches them with lower rank; show "Maybe: …" chips | db, be, fe | M | P2 |
| 3 | EXIF time/GPS extraction on upload | Users rarely type exact time and place; the photo already knows them | Parse EXIF `DateTimeOriginal` and GPS in be on upload, prefill form, let user correct before submit | be, fe | S | P1 |
| 4 | Reverse geocoding via self-hosted Nominatim | Uploaded photos have coordinates but no human address | Optional `nominatim` service in docker-compose; be resolves address per locale and caches by rounded lat/lon | be, infra, db | M | P2 |
| 5 | Camera ingestion API (push) | Approved cameras need a machine path, not the web form | `IngestSighting` RPC with per-camera API key, idempotency key, photo upload to S3; cameras table with owner and location | be, db, docs | L | P2 |
| 6 | RTSP/folder watcher sidecar | Hobbyist cameras dump JPEGs to a folder or stream RTSP | Small Go sidecar that watches a directory, runs OCR, calls `IngestSighting`; documented docker-compose profile | be, infra, docs | L | P3 |
| 7 | Duplicate sighting detection | One car in a traffic jam yields dozens of near-identical sightings | On insert, merge sightings of the same plate within N minutes and M meters into one with photo set | be, db | M | P1 |
| 8 | Perceptual hash of photos | Re-uploads of the same photo pollute history | Compute pHash at upload, reject or link exact/near duplicates, show "already uploaded" | be, db | S | P2 |
| 9 | Vehicle attributes (color, make) | Plate alone is ambiguous after misreads; color/make help verify | Optional fields filled by user or open-source classifier; shown on card and used in clarifications | db, be, fe | M | P3 |
| 10 | Bulk import tool for archives | Early adopters have years of dashcam photos | CLI `be/cmd/import` reading a folder + CSV (plate, time, lat, lon), dry-run report, uses the same usecase as upload | be, docs | M | P2 |

## B. Search and analytics

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 11 | Partial-plate wildcard search | Witnesses remember "A1?3" or only digits | Support `?`/`*` in query, translate to `LIKE` on `plate_search_key(plate)` with trigram-free prefix fallback; document limits | be, fe | M | P2 |
| 12 | Date range, country and "has photo/map" filters | Results for a popular region are too long to scroll | Extend `SearchRequest` with `from`, `to`, `market`, `has_coords`; filter chips under search field, state in URL | be, fe | M | P1 |
| 13 | Search near a place | "What was seen near my house last night" is a common question | `SearchRequest.near{lat,lon,radius_m}` with bbox prefilter on (lat, lon) index; map picker on home | db, be, fe | M | P2 |
| 14 | Sort options | Users want newest, oldest, or most-seen plates | `sort` enum (recent, oldest, sightings_count); sort select in results header | be, fe | S | P2 |
| 15 | Result count by country facet | Shows where matches are before filtering | Return `facets` (market → count) alongside `total`; render as counters on filter chips | be, fe | M | P3 |
| 16 | Search suggestions from popular plates | Empty dropdown is a missed hint | `Suggest` RPC returning top prefixes by sightings; quick search shows them under recent queries | be, fe | S | P3 |
| 17 | Plate activity heatmap (hour × weekday) | Reveals regular patterns of a vehicle | Aggregate query over sightings of a plate; small SVG heatmap block on plate page | be, fe | M | P2 |
| 18 | Public stats page | Shows the dataset is alive and growing | `/stats`: sightings per day, top regions, coverage map; computed by a cached aggregate RPC | be, fe | M | P3 |
| 19 | Query analytics (privacy-safe) | Learn which queries return nothing to improve OCR and coverage | Count zero-result query *shapes* (masked: letters→L, digits→D) per day, no raw plates; admin chart | be, db, fe | S | P2 |

## C. Plate page and map

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 20 | Marker clustering on the map | Dozens of sightings at one parking lot overlap | Cluster markers client-side (simple grid clustering, no new deps) with count badges; expand on zoom | fe | S | P2 |
| 21 | Share button (Web Share API + copy link) | Users send plate pages to friends/police | Share button in plate header: `navigator.share` with fallback to clipboard and toast | fe | S | P1 |
| 22 | Time slider on the map | Route over weeks is unreadable at once | Range slider filters timeline and markers by date; numbered route recomputed for the range | fe | M | P2 |
| 23 | "Last seen" summary card | The key answer should be visible without scrolling | Header block: last place, time ago, total sightings, first seen; computed from `Get` items | fe | S | P1 |
| 24 | Subscribe to a plate | Owners of stolen cars want to know about new sightings | Email-free web push or RSS/Atom feed per plate (`/feed/plate/<id>`); no account needed for RSS | be, fe | M | P2 |
| 25 | Export sightings (CSV/GPX/GeoJSON) | Lets people use the data in other tools or hand over to authorities | `ExportPlate` returning CSV/GPX; download buttons on plate page | be, fe | S | P2 |
| 26 | Photo comparison view | Clarifying a misread needs side-by-side photos | Lightbox mode showing two photos with synced zoom; entry point from timeline checkboxes | fe | M | P3 |
| 27 | Stolen/wanted flag with source link | Most valuable use case: finding stolen vehicles | Moderator-set flag on plate with public source URL; red badge on cards and plate page | db, be, fe | M | P2 |
| 28 | Printable plate report | Victims need a document for police or insurance | Print stylesheet + "Print report" button: map snapshot, timeline table, photo thumbnails | fe | S | P3 |

## D. UX and visual

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 29 | Skeleton loaders | Blank grid during load looks broken on slow networks | Card-shaped skeletons in home, results and plate page while requests are pending | fe | S | P2 |
| 30 | Empty and error states with retry | Network errors currently show a bare message | Illustrated empty state ("nothing found — try fewer characters"), error state with Retry button, offline hint | fe | S | P1 |
| 31 | Keyboard navigation | Power users and accessibility | `/` focuses search, arrows move in quick search and grid, `Esc` closes modals, `←/→` in lightbox (audit and complete) | fe | S | P2 |
| 32 | Accessibility audit (WCAG AA) | Public site should be usable with screen readers | ARIA labels on icon buttons, focus traps in modals, contrast check of both themes, `prefers-reduced-motion` | fe | M | P2 |
| 33 | Relative time with exact tooltip | "3 hours ago" reads faster; exact time still needed | `Intl.RelativeTimeFormat` in all 5 locales, `<time datetime>` with full date on hover | fe | S | P2 |
| 34 | Plate input with virtual plate keyboard | Typing Cyrillic look-alikes on a Latin keyboard is confusing | Optional on-screen keyboard showing only valid plate letters for the chosen format | fe | M | P3 |
| 35 | Onboarding hints on first visit | New users don't know what the service does | Dismissible 3-step hint (search, plate page, clarify), state in localStorage | fe | S | P3 |
| 36 | Card density switch | Some prefer large photos, others dense lists | Grid/list toggle in settings; list shows plate, place, time without photo | fe | S | P3 |
| 37 | Consistent design tokens | Ad-hoc colors across CSS files slow down theming | Move colors/spacing/radii to CSS variables in `theme.css`, replace literals in feature CSS | fe | M | P2 |

## E. Mobile and PWA

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 38 | PWA manifest + service worker | Installable app, faster repeat visits | `manifest.webmanifest`, icons, minimal service worker (cache app shell, network-first API) via a few lines of own code | fe | S | P1 |
| 39 | Camera capture for upload | On the street people shoot right away | `<input capture="environment">` in the upload form, client-side resize before upload | fe | S | P1 |
| 40 | Offline upload queue | Mobile network drops on the road | Queue pending uploads in IndexedDB, retry via service worker background sync, show queued badge | fe | M | P2 |
| 41 | Geolocation for "near me" | Mobile users want sightings around them | "Near me" chip using `navigator.geolocation`, calls near-search (#13) | fe | S | P3 |
| 42 | Swipe gestures in lightbox | Native-feeling gallery on phones | Touch swipe left/right, pinch-zoom, swipe down to close | fe | S | P2 |
| 43 | Bottom navigation on small screens | Header controls are hard to reach one-handed | Fixed bottom bar (Home, Search, Upload, Settings) under 600 px | fe | S | P3 |
| 44 | Responsive images (srcset) | Phones download full photos needlessly | Generate 2–3 sizes at upload; `srcset`/`sizes` on cards and lightbox | be, fe | M | P2 |
| 45 | Share target | Share a photo from the phone gallery straight into upload | `share_target` in manifest, upload page reads the shared file | fe | S | P3 |
| 46 | Mobile e2e smoke in CI | Mobile layout regressions go unnoticed | Puppeteer run at 390×844 on built fe with mocked API; screenshot artifacts | fe, infra | M | P3 |

## F. Moderation and trust (clarifications, complaints, reputation)

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 47 | Anonymous device reputation | Spam clarifications and complaints need weighting without accounts | Signed device token (random id + HMAC cookie); count accepted vs rejected actions; weight consensus by score | be, fe | M | P2 |
| 48 | Show clarification history on sighting | Transparency: why the plate changed | Collapsible "Clarifications" list in lightbox: proposed plate, count, status | be, fe | S | P2 |
| 49 | Auto-hide photo after N complaints | Personal-data complaints must act before an admin wakes up | `hidden_at` on sighting set when ≥ N `personal_data`/`bad_photo` complaints; excluded from Search/Get; admin can restore | db, be | S | P1 |
| 50 | Rate limit on Clarify/Complain | Cheap protection against floods | Token-bucket per IP (and device token) middleware for write RPCs; 429 with localized message | be, fe | S | P1 |
| 51 | Proof-of-work captcha for anonymous writes | Bots without paid captcha services | Lightweight hashcash challenge for writes (Clarify, Complain, upload), solved in a Web Worker | be, fe | M | P3 |
| 52 | Owner claim/removal request | Car owners may want their plate hidden | "I own this car" request with reason; admin queue; on approval plate is blurred/hidden | db, be, fe | M | P2 |
| 53 | Moderation audit log | Admin actions must be accountable | `moderation_log` table (who, action, target, before/after); written by every admin RPC; view in admin | db, be, fe | S | P2 |
| 54 | Complaint outcome feedback | Reporters don't know if anything happened | Complaint gets a public status URL; shows "resolved: photo hidden / kept" | be, fe | S | P3 |
| 55 | Uploader trust tiers | Trusted contributors' uploads shouldn't wait for moderation | Tiers by accepted uploads; tier ≥ 2 skips pre-moderation; demote on upheld complaints | db, be | M | P3 |

## G. Admin and operations

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 56 | Admin dashboard counters | Admin needs a quick view of what needs attention | Top of `/admin`: new complaints, pending clarifications, uploads today, hidden photos | be, fe | S | P2 |
| 57 | Admin photo actions | Resolving a complaint often means hiding or blurring a photo | Hide/restore sighting, re-assign plate, delete photo from S3 — from the complaint row | be, fe | M | P1 |
| 58 | Clarifications moderation queue | Consensus can be wrong; admin must accept/reject manually | `ListClarifications` / `ResolveClarification` admin RPCs; tab in `/admin` with photo, current and proposed plate | be, fe | M | P1 |
| 59 | Upload pre-moderation queue | User photos must be checked before public | Uploads land as `pending`; admin approve/reject with reason; OCR read editable | db, be, fe | M | P1 |
| 60 | Multiple admins with named tokens | Single shared token can't be revoked per person | `admins` table (name, token hash, role); token → admin name in audit log; CLI to add/revoke | db, be, docs | M | P2 |
| 61 | Admin search by sighting id/plate | Admin often starts from a user report link | Search box in admin accepting plate or `/plate/<id>` URL; shows all sightings with moderation status | fe, be | S | P2 |
| 62 | Bulk actions in admin | Spam waves produce hundreds of items | Checkbox selection + bulk resolve/hide with confirmation | fe, be | S | P3 |
| 63 | Camera requests admin view | Requests (in progress) need triage | Admin tab listing camera requests on a map, statuses (new, contacted, installed, rejected) | be, fe | M | P2 |
| 64 | Data retention job | Old sightings and resolved complaints shouldn't live forever | Scheduled be command: purge sightings older than policy (configurable), orphan S3 objects, old complaints | be, infra | M | P2 |

## H. Privacy and law (a plate is personal data)

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 65 | Automatic face and other-plate blurring | Photos contain bystanders and other cars | Open-source detector at upload (face + secondary plates), blur before storing public photo; keep original private or drop it | be | L | P1 |
| 66 | Strip EXIF from stored photos | Original EXIF leaks uploader device and exact location | Re-encode on upload after reading needed EXIF (#3); store only the cleaned file | be | S | P1 |
| 67 | Privacy policy and terms pages | Required before accepting user uploads | `/privacy` and `/terms` routes with localized texts in all 5 languages; link in footer and upload form | fe, docs | S | P1 |
| 68 | Coarsened location for public view | Exact home-parking coordinates are sensitive | Round public lat/lon (e.g. ~100 m grid) for sightings at night or repeated at one place; exact only for admins | be | M | P2 |
| 69 | Data subject request flow | GDPR/152-FZ style access and deletion requests | Form + admin queue for "delete my plate data"; documented SLA; execution hides plate and purges photos | db, be, fe, docs | M | P2 |
| 70 | No raw plates in logs | Logs must not become a shadow plate database | Log masked plates (hash + length) in be; review clay request logging; CI grep test for plate fields in log calls | be | S | P1 |
| 71 | Search abuse protection | Stalking: repeatedly searching one plate | Per-IP limits on Get/Search of the same plate; optional delay of fresh sightings (e.g. 24 h) for public | be | M | P2 |
| 72 | Consent checkbox on upload | Uploader confirms rights and lawful purpose | Required checkbox with link to terms; stored with upload (timestamp, terms version) | db, be, fe | S | P2 |
| 73 | Data protection notes in docs | Contributors must know the legal model | `docs/PRIVACY.md`: what is stored, retention, who can see what, how requests are processed | docs | S | P2 |

## I. Performance and reliability

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 74 | Keyset pagination for search | `OFFSET` degrades on deep scroll | Cursor `(created_at, id)` in `SearchRequest`/`next_cursor`; keep offset for compatibility for a while | be, fe | M | P2 |
| 75 | Approximate total | `count(*)` over big result sets is slow | Return exact total up to a cap (e.g. 1000), then "1000+"; UI shows "1000+ results" | be, fe | S | P2 |
| 76 | HTTP caching for read RPCs | Home and plate pages are mostly static minutes-wise | `Cache-Control`/`ETag` on Search (empty query, market) and Get; short TTL, varies by `include-test-data` | be | S | P2 |
| 77 | Immutable S3 keys + long cache | Photos never change after upload | Content-addressed keys, `Cache-Control: immutable, max-age=1y` on objects, served via proxy/CDN-friendly path | be, infra | S | P2 |
| 78 | Query plan regression tests | GIN index must stay used (expression mismatch silently kills it) | Repository test asserting `EXPLAIN` uses `plates_plate_gin_idx` for search and partial indexes for filters | be | S | P1 |
| 79 | Load test scenario | Know capacity before going public | k6 (open source) script for Search/Get mix against local stack; documented baseline numbers | infra, docs | S | P3 |
| 80 | Graceful shutdown and timeouts | Deploys shouldn't drop requests; slow queries shouldn't pile up | Context deadlines per RPC, `statement_timeout` on pool, graceful HTTP/gRPC stop on SIGTERM | be | S | P1 |
| 81 | Lazy-load map and lightbox code | Initial bundle should be search only | Dynamic `import()` for `map.ts`, `lightbox.ts`, `admin.ts`; check bundle size in CI | fe | S | P2 |
| 82 | Thumbnail generation worker | Upload request shouldn't wait for resizing/OCR | Job table (`pending → done/failed`) processed by a goroutine worker with retries | be, db | M | P2 |

## J. Infrastructure, CI and observability

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 83 | Health and readiness endpoints | Compose/orchestrators need to know when be is ready | `/healthz` (process) and `/readyz` (DB ping + S3 bucket check); compose `healthcheck` and `depends_on: service_healthy` | be, infra | S | P1 |
| 84 | Production compose profile | Dev stack (Vite dev server) is not for public hosting | Multi-stage fe build served by Caddy/nginx (Docker Hub), be image, TLS via Caddy, documented `.env` vars | infra, docs | M | P1 |
| 85 | golangci-lint and Dependabot | Catch bugs and outdated deps automatically | golangci-lint job in CI with a small config; `.github/dependabot.yml` for gomod, npm, actions, docker | infra | S | P1 |
| 86 | Structured logs with request id | Debugging needs correlation across be calls | slog JSON handler, request id middleware (header `x-request-id` in/out), log RPC name, duration, status | be | S | P1 |
| 87 | Prometheus metrics | See RPS, latency, errors per RPC and DB pool usage | `/metrics` with RPC histograms and pgx pool stats; optional Prometheus + Grafana compose profile | be, infra | M | P2 |
| 88 | OpenTelemetry traces | Find slow SQL inside an RPC | OTel SDK with OTLP exporter to optional Jaeger (Docker Hub) profile; spans for RPC, SQL, S3 | be, infra | M | P3 |
| 89 | Automated DB backups | Pet project data on a laptop disk is fragile | `make backup` / scheduled `pg_dump` + S3 bucket mirror to a separate directory; restore doc | infra, docs | S | P2 |
| 90 | E2E tests in CI | Screens break without anyone noticing | Puppeteer/Playwright suite on composed stack with test data: search, plate page, clarify, complain | fe, infra | M | P2 |
| 91 | Preview deploys of PRs | Reviewing UI changes needs a running build | Static fe build with mocked API published as workflow artifact or GitHub Pages per PR | infra | M | P3 |

## K. Monetization and community (no paid external keys)

| # | Idea | Why | What to do | Layers | Size | Prio |
|---|---|---|---|---|---|---|
| 92 | Contributor leaderboard (opt-in) | Gamification drives uploads | Weekly top uploaders by accepted sightings, pseudonymous nicknames, opt-in only | db, be, fe | M | P3 |
| 93 | Donations page | Cover hosting without ads or paywalls | Static page with crypto/sponsor links (GitHub Sponsors, no API keys), transparent cost breakdown | fe, docs | S | P3 |
| 94 | Open data dumps | Researchers and OSM-like community value | Monthly anonymized export (plates hashed, coarse location) under ODbL; documented schema | be, docs | M | P3 |
| 95 | Public read API with quotas | Third-party apps and bots can build on the service | Documented read-only RPCs with anonymous quota and free self-issued API keys for higher limits | be, docs | M | P3 |
| 96 | Telegram/Matrix bot | Many users live in messengers | Self-hosted bot: send plate → last sightings link; send photo → upload flow (bot token configured by owner) | be, infra | M | P3 |
| 97 | Community translations | Five languages hand-maintained; contributors can add more | `docs/TRANSLATING.md`, key-coverage check script in CI (all locale files have the same keys) | fe, docs, infra | S | P2 |
| 98 | Stolen-car community board | Bring together owners and volunteers | Page listing plates with stolen flag (#27) and recent sightings, sorted by freshness | fe, be | M | P3 |
| 99 | Embeddable plate widget | Owners post "have you seen my car" on forums | `<iframe>`/image badge with last-seen info for flagged plates only | be, fe | S | P3 |
| 100 | Contributor guide and good-first-issues | Public repo should attract help | `CONTRIBUTING.md` (layers, codegen, registries), issue templates, labeled starter issues from this list | docs | S | P2 |

## Top-10 for immediate implementation

Picked from P1 items that are small/medium and independent of the in-progress upload/OCR and camera-request work.
Waves run one after another; tasks inside a wave can run in parallel — they do not share files
(each fe task with UI texts owns the i18n locale files within its wave).

### Wave 1 — foundations

| Task | Idea | Files |
|---|---|---|
| 1 | #83 Health/readiness endpoints | `be/cmd/main.go`, new `be/internal/health/*`, `docker-compose.yml` |
| 2 | #85 golangci-lint + Dependabot | `.github/workflows/ci.yml`, `.github/dependabot.yml`, `be/.golangci.yml` |
| 3 | #30 Empty/error states with retry | `fe/src/main.ts`, `fe/src/ui.ts`, new `fe/src/states.css`, `fe/src/i18n/locales/*` |
| 4 | #49 Auto-hide photo after N complaints | `be/migrations/0001_init.sql`, `be/internal/repository/complaints.go`, `be/internal/usecase/complaints.go` (+ tests, mocks) |

### Wave 2 — search and write protection

| Task | Idea | Files |
|---|---|---|
| 5 | #12 Date range / country / has-map filters | `be/api/carsearch/carsearch.proto` (+ `make gen`), `be/internal/implementation/search.go`, `be/internal/usecase/plate_sightings.go`, `be/internal/repository/postgres.go`, `be/internal/models/search.go`, `fe/src/api/index.ts`, `fe/src/main.ts`, new `fe/src/filters.ts/.css`, `fe/src/i18n/locales/*` |
| 6 | #50 Rate limit on Clarify/Complain | new `be/internal/ratelimit/*`, `be/cmd/main.go` |

### Wave 3 — sharing, PWA, observability

| Task | Idea | Files |
|---|---|---|
| 7 | #21 Share button on plate page | `fe/src/plate.ts`, new `fe/src/share.ts/.css`, `fe/src/i18n/locales/*` |
| 8 | #38 PWA manifest + service worker | `fe/index.html`, `fe/vite.config.ts`, new `fe/public/manifest.webmanifest`, `fe/public/sw.js`, icons |
| 9 | #86 Structured logs with request id | new `be/internal/logging/*`, `be/cmd/main.go` |

### Wave 4 — moderation

| Task | Idea | Files |
|---|---|---|
| 10 | #58 Clarifications moderation queue | `be/api/carsearch/carsearch.proto` (+ `make gen`), new `be/internal/implementation/{list,resolve}_clarification*.go`, `be/internal/usecase/clarify_plate.go`, `be/internal/repository/postgres.go`, `fe/src/admin.ts`, `fe/src/admin.css`, `fe/src/api/index.ts`, `fe/src/i18n/locales/*` |
