# Deployment sizing (VPS)

Estimates are based on measurements of the full local stack (`make run`) on 2 vCPU / 6 GB
(Colima on Apple Silicon), with test data loaded and real photo uploads through `Ingest`.

## Measured footprint

| Service | RAM idle | Under load | Image size |
|---|---|---|---|
| Postgres 17 | ~32 MB | grows with data / `shared_buffers` | 415 MB |
| S3 (RustFS) | ~140 MB | — | 373 MB |
| OCR (`ocr/`, fast-alpr, ONNX on CPU) | 75–100 MB | ~100 MB, **0.10–0.15 s per photo** | 602 MB |
| Backend (Go) | 8–50 MB | — | 44 MB |
| Frontend (Vite dev server) | ~70 MB | — | — |

- Storage per sighting: **~0.45 MB** of photos (full ~400 KB + thumbnail ~50 KB) and **1–2 KB** in Postgres
  (including indexes).
- The OCR timing was measured on ARM (Apple Silicon). On a typical x86 VPS expect **2–3× slower**
  (~0.3–0.5 s per photo per core).
- Backend CPU is negligible for reads; search uses GIN indexes (mask search is a full scan over `plates`).

## Recommended VPS

| Stage | Load | VPS | Approx. cost |
|---|---|---|---|
| Start / demo | up to ~1,000 uploads and ~10,000 page views per day | 2 vCPU, 4 GB RAM, 40 GB SSD | ~€5–8 / month |
| Growth | ~10,000 uploads and ~100,000 page views per day | 4 vCPU, 8 GB RAM, 80 GB SSD + external object storage for photos | ~€15–25 / month + storage |

RAM budget at the start stage: OS ~300 MB, Postgres 0.3–1 GB, S3 150–300 MB, OCR 150–400 MB at peak,
backend ~100 MB, reverse proxy ~20 MB — about 1.5 GB minimum; 4 GB leaves headroom.

## The real bottleneck: photo storage

Disk for photos grows much faster than anything else: 10,000 uploads/day ≈ **4.5 GB/day ≈ 135 GB/month**.
At the growth stage move photos off the VPS disk to an external S3-compatible object storage
(Hetzner Object Storage, Backblaze B2, Cloudflare R2, etc. — roughly $5–6 per TB per month).
The backend already talks to S3 through `minio-go`, so this is a configuration change
(`S3_ENDPOINT`, credentials, bucket, `S3_PUBLIC_URL`), not a code change.

OCR is not a bottleneck for a long time: one core handles several photos per second.
For continuous camera feeds (see [CAMERAS.md](CAMERAS.md), if present) plan one extra core per ~2–5 cameras
sending one frame per second, or filter frames on the device before upload.

## Before deploying to a VPS

- **Frontend:** build static assets (`npm run build`) and serve them with Caddy or nginx with HTTPS,
  instead of the Vite dev server; proxy `/api` to the backend and `/s3` to object storage.
- **Secrets in `.env`:** Postgres and S3 passwords, `ADMIN_TOKEN`, `TOKEN_SIGNING_KEY`, `ADMIN_PATH`.
  `docker-compose.yml` ships local dev defaults — never use them in production.
- **Network:** expose only 80/443; keep Postgres, S3, OCR and the backend ports on the internal network.
- **Backups:** scheduled `pg_dump` (or WAL archiving) and bucket versioning / replication for photos.
- **Test data:** it is hidden without the `include-test-data` header, but migration `0002_test_data.sql`
  still loads it — skip it on a production database.
