# Data sources for plate sightings

car-search shows **sightings**: a photo of a car with a readable licence plate, the time it was taken and the
place (address and/or coordinates). The demo database holds generated test data
(`be/migrations/0002_test_data.sql`, CC0 photos with blurred plates); signed-in users can also upload photos that
are recognized by OCR (`rpc Ingest`, see [OCR_PIPELINE.md](OCR_PIPELINE.md)). This document lists where real data
could come from, what each source actually gives, on what terms, and how risky it is.

Scope: **data sources only**. The app's UI and storage are out of scope; the upload path that already exists is
described in [OCR_PIPELINE.md](OCR_PIPELINE.md).

> Not legal advice. Laws and terms change; check the current text before relying on anything below.
> Links were checked in November 2026.

**Contents**

1. [What a sighting needs](#1-what-a-sighting-needs)
2. [Open plate / ALPR datasets](#2-open-plate--alpr-datasets)
3. [City cameras and open government data](#3-city-cameras-and-open-government-data)
4. [Crowdsourcing: phone and dashcam uploads, street-level imagery](#4-crowdsourcing-phone-and-dashcam-uploads-street-level-imagery)
5. [Commercial ALPR data](#5-commercial-alpr-data)
6. [Own pipeline: camera → recognition → API](#6-own-pipeline-camera--recognition--api)
7. [Legal side](#7-legal-side)
8. [Summary table](#8-summary-table)
9. [Recommended plan](#9-recommended-plan)

## 1. What a sighting needs

| Field | Column | Notes |
|---|---|---|
| Photo + thumbnail | `photo_key`, `photo_thumb_key` (S3) | the plate must be readable for the record to make sense |
| Plate text | `plates.plate` + `meta.type/region` | from a human or from ALPR (OCR) |
| Time | `created_at` | when the photo was taken, not when it was uploaded |
| Place | `address`, `address_locale`, `lat`/`lon` | coordinates are optional; an address can be reverse-geocoded from them |

The hard part is that **all three together** (readable plate + real timestamp + real location) are rare in open
data: research datasets have plates but no time/place, street-level imagery has time/place but blurs plates,
and city cameras give place/time but low-resolution traffic views without plate reading.

## 2. Open plate / ALPR datasets

These are made for training and benchmarking detectors and OCR. None of the ones below ships capture time
or GPS per image, so they **cannot fill the sightings table as real data** — but they are useful for
(a) training/evaluating a recognizer for the own pipeline (section 6) and (b) realistic demo photos, with care
about the licence.

| Dataset | What is inside | Time / GPS | Licence / access |
|---|---|---|---|
| [CCPD](https://github.com/detectRecog/CCPD) (Chinese City Parking Dataset, ECCV 2018) | 300k+ images from Chinese parking lots, one plate per image; plate text, box, corners, tilt, blur encoded in the file name; CCPD-Green subset with new-energy plates | no | repository under MIT; downloads from Google Drive / Baidu; cite the paper. Images show real Chinese plates — redistributing them on a public site is still a privacy question (section 7) |
| [UFPR-ALPR](https://github.com/raysonlaroca/ufpr-alpr-dataset) | 4,500 frames 1920×1080 of 150 vehicles (cars and motorcycles, Brazil), recorded from a moving car; plate text, plate and character boxes, vehicle make/model | no | **academic, non-commercial only**, licence agreement required |
| [RodoSol-ALPR](https://github.com/raysonlaroca/rodosol-alpr-dataset) | 20,000 images 1280×720 from toll-plaza cameras in Brazil (Brazilian and Mercosur plates, day/night, rain); plate text and corners; faces blurred | no | **academic, non-commercial only**; signed agreement sent from a university e-mail |
| [OpenALPR benchmarks](https://github.com/openalpr/benchmarks) | small benchmark image sets (detection, end-to-end, OCR) for several regions used to test OpenALPR | no | AGPL-3.0 repository |
| [Nomeroff Net / AUTO.RIA datasets](https://github.com/ria-com/nomeroff-net) | plate detection set from the Ukrainian car marketplace AUTO.RIA plus OCR sets for UA, RU, EU, KZ, BY, GE, AM, KG and others; download links in the README | no | code GPL-3.0; dataset terms are not stated separately — ask the authors before reuse beyond research |
| [Car_plate_OCR_dataset](https://huggingface.co/datasets/AY000554/Car_plate_OCR_dataset) (Hugging Face) | ~45.5k crops of Russian plates with text labels, derived from Nomeroff Net | no | see the dataset card; crops only (no scene, no context) |
| [Car License Plate Detection](https://www.kaggle.com/datasets/andrewmvd/car-plate-detection) (Kaggle) | 433 images with plate boxes (Pascal VOC), mixed countries | no | listed as public domain; detection only, no plate text |
| [License Plate Recognition](https://universe.roboflow.com/roboflow-universe-projects/license-plate-recognition-rxg4e) (Roboflow Universe) | ~10k images with plate boxes, YOLO/COCO exports; many other plate sets on Roboflow Universe | no | per-dataset licence on the page (often CC BY 4.0); known train/test overlap |
| [Open Images V7](https://storage.googleapis.com/openimages/web/index.html) | general dataset with a "Vehicle registration plate" class (boxes) on Flickr photos | no (EXIF stripped) | annotations CC BY 4.0, images listed as CC BY 2.0 — check per image |
| [Wikimedia Commons: vehicle registration plates](https://commons.wikimedia.org/wiki/Category:Vehicle_registration_plates) | photos of cars and plates from many countries; some files keep EXIF time/GPS, many have a place in the description | sometimes | per file (CC0, CC BY, CC BY-SA…); the current test photos come from here (`be/internal/test_utils/s3/cars/CREDITS.md`) |

**Applicability:** good for training and testing recognition; good for demo images (Commons, CC0/CC BY with
attribution). Not a source of real sightings: no time and place, and plates of real people would be shown
next to invented times and addresses, which is misleading. Keep such data under `is_test_data = true`.

## 3. City cameras and open government data

What cities publish is usually **camera locations** and **periodic low-resolution snapshots of traffic**, not
plate reads. Plate data from enforcement cameras (speed, bus lanes, parking) is personal data held by police
or transport authorities and is not open.

| Source | What is available | Usefulness for sightings |
|---|---|---|
| [data.mos.ru — locations of traffic-enforcement cameras (Moscow)](https://data.mos.ru/opendata/7704786030-raspolojenie-statsionarnyh-kompleksov-fotovideofiksatsii-narusheniy-pravil-dorojnogo-dvijeniya-rossiyskoy-federatsii), API: [apidata.mos.ru](https://apidata.mos.ru/Docs) (key required) | coordinates and addresses of fixed and mobile enforcement cameras; other sets (yard/entrance CCTV registers) list cameras, not footage; several sets are intermittently unavailable | **locations only** — could be used as "known camera points" for a map layer; no images, no plates. Moscow's video streams (the city video surveillance system) are available only to residents for their own yard/entrance after login, not for reuse |
| [TfL JamCams (London)](https://api.tfl.gov.uk/Place/Type/JamCam), terms: [TfL open data](https://tfl.gov.uk/info-for/open-data-users/) | ~800–900 traffic cameras: a JPEG and a short MP4 refreshed every ~5 minutes; no archive | wide shots, plates generally unreadable; TfL open data licence requires attribution. Fine for traffic context, not for sightings |
| [511NY developer API (New York)](https://511ny.org/developers/help) | camera list with snapshot/stream URLs, incidents, speeds; free key, Developer Access Agreement, rate limit | same: traffic views, no plates; agreement forbids altering the data and may revoke access |
| [Caltrans CCTV (California)](https://cwwp2.dot.ca.gov/documentation/cctv/cctv.htm) | JSON/CSV/XML lists of highway cameras with still-image URLs | highway overview images, no plates |
| [Digitraffic (Finland)](https://www.digitraffic.fi/en/road-traffic/) | road weather cameras (images), traffic counters, open API | CC BY 4.0; road condition images, plates not readable |
| [Windy Webcams API](https://api.windy.com/webcams) | aggregated public webcams worldwide with location and image URLs | scenic/traffic webcams; terms restrict storing and reusing images |

**Legal limits:** snapshots are published for traffic information. Running ALPR over them to track individual
cars is a different purpose: in the EU it needs its own legal basis (GDPR art. 5(1)(b) purpose limitation,
art. 6), and most portals' terms forbid such processing or storage. In practice the image resolution makes it
moot anyway.

**Applicability:** none as a source of plate sightings. At most, camera locations as an optional map layer.

## 4. Crowdsourcing: phone and dashcam uploads, street-level imagery

### 4.1 User uploads (the realistic source)

Users photograph a car with a phone or export a dashcam frame and upload it. Everything a sighting needs can
come from the file itself:

- **Time** — EXIF `DateTimeOriginal` (+ `OffsetTimeOriginal` for the time zone); dashcams also burn the time
  into the frame and write it into the video's metadata / GPS track (often NMEA in a sidecar or embedded).
- **Place** — EXIF GPS tags (`GPSLatitude`, `GPSLongitude`, refs, `GPSTimeStamp`;
  [tag reference](https://exiftool.org/TagNames/GPS.html)); dashcams with GPS log a track per second.
- **Plate** — typed by the user and/or suggested by ALPR (section 6); the existing **Clarify** flow and
  consensus already cover corrections.

Caveats:

- Messengers and many social apps strip EXIF; browsers and OS photo pickers may remove location unless the
  user explicitly allows it. Fall back to "take a photo now" in the web app (the browser's Geolocation API +
  current time) or to a manual place picker on the map.
- EXIF is trivially editable — treat it as user-provided, not proof. Plausibility checks (time not in the
  future, GPS within the country of the plate format, not hundreds of km from the previous sighting minutes
  ago) and the existing complaint/admin flow are the moderation tools.
- Strip EXIF from the stored image after extracting the fields (it can contain the device serial, the uploader's
  home location, etc.), and blur faces before publishing.
- Dashcam communities ([r/Dashcam](https://www.reddit.com/r/Dashcam/), regional forums and YouTube/Telegram
  channels) are a pool of potential contributors, but their published videos belong to their authors — reuse
  only with permission or via the authors' own uploads.

### 4.2 Mapillary and KartaView (street-level imagery)

| | [Mapillary](https://www.mapillary.com/developer) (Meta) | [KartaView](https://kartaview.org/terms) (Grab) |
|---|---|---|
| Data | billions of street-level images with time, GPS, heading; API v4, vector tiles | street-level image sequences with time and GPS |
| Licence | imagery CC BY-SA 4.0 ([terms](https://www.mapillary.com/terms)) | imagery CC BY-SA 4.0, code MIT |
| Plates | **automatically blurred** on upload, together with faces ([privacy policy](https://www.mapillary.com/privacy), [blurring help](https://help.mapillary.com/hc/en-us/articles/115001663705-Blurring-images-on-Mapillary)) | **automatically blurred** with faces |

**Applicability:** the blurring is a deliberate privacy policy, so these platforms are not a plate source and
should not be "un-blurred" or used to re-identify vehicles. They are usable as **context** (e.g. a street view
near a sighting point) under CC BY-SA with attribution.

## 5. Commercial ALPR data

Commercial ALPR networks are the only sources that actually have "plate + photo + time + place" at scale.
They are listed for understanding the market; **access is closed or paid and restricted to vetted customers**
(police, insurers, lenders, repossession, parking operators).

| Company | What it is | Access |
|---|---|---|
| [Flock Safety](https://www.flocksafety.com/) (US) | fixed ALPR cameras sold to police, HOAs and businesses; shared search network | customers only; data owned by the camera owners; heavy public scrutiny and state-law limits |
| [Motorola Solutions — Vigilant](https://www.motorolasolutions.com/) (US) | Vigilant PlateSearch: law-enforcement ALPR data, including commercially collected plates | law enforcement / vetted customers |
| [DRN Data](https://drndata.com/) (US) | private plate-scan database (collected largely by repossession agents' cameras) for lenders, insurers | contract only |
| [Rekor](https://rekor.ai/) (US) | ALPR software/cameras and a vehicle-recognition platform | commercial licences |
| [Plate Recognizer](https://platerecognizer.com/) | ALPR as an API/SDK (recognition, not data); has a free tier for low volumes | commercial, usable as the OCR step in section 6 |

In Russia, plate data from enforcement cameras belongs to traffic authorities (ЦОДД/ГИБДД); commercial
"car history" services aggregate it via B2B agreements and are not an open source either.

**Applicability:** none for a pet project (cost, contracts, and — for re-publishing — legally impossible).
Plate Recognizer–style recognition APIs are a possible replacement for self-hosted OCR.

## 6. Own pipeline: camera → recognition → API

The practical way to get real sightings is to collect them: the user's (or contributors') phone or dashcam →
plate recognition → upload.

```
phone / dashcam frame ──► detect plate ──► OCR plate text ──► blur faces (+ other plates) ──► upload API
       (EXIF time+GPS)     (YOLO model)     (fast-plate-ocr /     (face detector)             (photo, plate,
                                             EasyOCR / Nomeroff)                               time, lat/lon)
```

Open-source building blocks:

| Tool | Role | Licence | Notes |
|---|---|---|---|
| [fast-alpr](https://github.com/ankandrew/fast-alpr) + [fast-plate-ocr](https://github.com/ankandrew/fast-plate-ocr) | end-to-end ALPR: YOLO plate detector + plate OCR, ONNX Runtime (CPU ok) | MIT | `pip install fast-alpr[onnx]`; "global" OCR model; easiest start |
| [Nomeroff Net](https://github.com/ria-com/nomeroff-net) | detection + plate type classifier + OCR for UA/RU/EU/KZ/BY/… | GPL-3.0 | best coverage of Cyrillic plates (RU/UA), heavier (PyTorch) |
| [OpenALPR](https://github.com/openalpr/openalpr) | classic C++ ALPR (US/EU and some other regions) | AGPL-3.0 | open-source version is no longer actively developed; AGPL obligations if exposed as a service |
| [EasyOCR](https://github.com/JaidedAI/EasyOCR) | general OCR, 80+ languages incl. Cyrillic and Chinese | Apache-2.0 | use on a cropped plate; needs a separate detector |
| [Ultralytics YOLO](https://github.com/ultralytics/ultralytics) + plate weights, e.g. [yolov11-license-plate-detection](https://huggingface.co/morsetechlab/yolov11-license-plate-detection) | plate detector to crop/blur plates or feed OCR | AGPL-3.0 (Ultralytics) — check model card licences | many community weights on Hugging Face |

Where to run it: on the server after upload (simplest; a Python sidecar worker reading from S3) or on the
device (ONNX in the browser / mobile app — the photo with other people's faces never leaves the device
unblurred). Datasets from section 2 serve to evaluate accuracy on RU/UA/EU/CN plates.

**API: already exists.** `rpc Ingest` (`POST /api/carsearch.CarSearchService/Ingest`, page `/upload`) takes a
photo (base64, ≤ 8 MB), optional `taken_at`, `lat`/`lon`, `address`; reads EXIF time/GPS, runs the `ocr` service
(fast-alpr), normalizes the plate, stores re-encoded photos without EXIF and creates the sighting; sign-in and a
Free quota of 5 uploads/24 h are required ([OCR_PIPELINE.md](OCR_PIPELINE.md)). What a real-data rollout still needs
on top of it: face / other-plate blurring before storing, a `source` field and per-source API keys for cameras and
batch import, photo-hash deduplication, a "pending" state with pre-moderation, and a retention policy.

## 7. Legal side

### 7.1 Is a plate personal data?

| Jurisdiction | Status of a licence plate | What it means here |
|---|---|---|
| **EU / EEA (GDPR)** | Personal data when the person can be identified directly or indirectly ([GDPR art. 4(1)](https://eur-lex.europa.eu/eli/reg/2016/679/oj)). Authorities (and the EDPB [Guidelines 3/2019 on video devices](https://www.edpb.europa.eu/sites/default/files/files/file1/edpb_guidelines_201903_video_devices_en_0.pdf)) treat plates as identifying; a camera covering public space is not "household" use ([CJEU C-212/13 Ryneš](https://curia.europa.eu/juris/liste.jsf?num=C-212/13)). Germany's Federal Court of Justice allowed dashcam footage as accident evidence but found permanent recording unlawful under data-protection law ([BGH VI ZR 233/17, 2018](https://www.bundesgerichtshof.de/SharedDocs/Pressemitteilungen/DE/2018/2018088.html)). | A public "where was car X seen" database is **location tracking of identifiable people**: needs a legal basis (consent is impractical — the car owner is not the uploader; legitimate interest will very likely fail the balancing test for a public search), transparency, data-subject rights, a DPIA (systematic monitoring of public areas, art. 35). Realistically **not lawful as a public service in the EU**. |
| **Russia (152-ФЗ)** | Roskomnadzor's public position: a plate (or VIN) **by itself** is not personal data, it identifies a vehicle ([Fontanka, Aug 2026](https://www.fontanka.ru/2026/08/15/76590846/); [Garant consultation, 2023](https://base.garant.ru/77066099/)). But it becomes personal data **in combination** with other information that identifies the owner (name, phone, photo of the driver, regular addresses). Photos of people are biometric/personal data. Law text: [152-ФЗ](https://www.consultant.ru/document/cons_doc_LAW_61801/). | Plate + time + place alone is in a grey zone; a photo with a visible driver's face, or linking a plate to a person, is clearly personal data (consent, notification to Roskomnadzor as an operator, localization of Russian citizens' data in Russia). Positions can change — keep the data minimal. |
| **USA** | No federal law on plates; public-space photography is generally lawful. ALPR is regulated by **state** laws — retention limits from minutes to years, use restrictions, policies and audit logs ([NCSL overview](https://www.ncsl.org/technology-and-communication/automated-license-plate-readers-state-statutes)); e.g. California requires ALPR operators (including private ones) to have a usage and privacy policy, access logs and security ([Cal. Civ. Code § 1798.90.5 ff.](https://leginfo.legislature.ca.gov/faces/codes_displaySection.xhtml?lawCode=CIV&sectionNum=1798.90.5)). State privacy acts (CCPA/CPRA and others) can cover geolocation. | An operator of an ALPR system must follow the state's rules (policy, logs, retention); selling or sharing ALPR data is restricted in several states. |
| **Other** | UK GDPR (as EU), Ukraine's law on personal data protection and China's PIPL also treat identifiable vehicle/location data as personal information. | The safe assumption everywhere: plate + time + location = personal data. |

### 7.2 Photos

- **Faces** of drivers, passengers and pedestrians are personal data everywhere (and biometric data when
  processed for identification). Blur them before publishing — the complaint reason `personal_data` already
  exists for misses.
- Other cars' plates in the frame: blur everything except the plate the sighting is about.
- Copyright: the uploader owns the photo; the terms of service must grant the project a licence to publish it.
- EXIF: remove after extracting time/GPS.

### 7.3 Recommendations for this project

1. **Keep the public demo on test data** (`is_test_data`), as today: CC0 photos with blurred plates and the
   TEST PHOTO mark, generated plates, times and places. This is safe in every jurisdiction.
2. Real data only **opt-in and self-collected**: the uploader photographs in public places, accepts terms
   (they have the right to the photo; no faces; no private property; purpose of the service), and the service
   blurs faces and unrelated plates automatically.
3. **Minimize**: no owner data ever, no joining with registries, no VIN; coarse location (street/area) in public
   views, exact coordinates only where needed; a **retention limit** (e.g. delete sightings older than N months)
   and per-plate **takedown** (complaint → admin → delete), honoured quickly.
4. Do not offer real-data search **publicly in the EU/UK**; if it ever goes public, restrict geography and run a
   DPIA first; in the US follow the states' ALPR rules (policy, logs, retention); in Russia avoid faces and any
   link to a person, and check operator-notification duties.
5. Do not scrape city cameras, street-level imagery or social media for plates; do not un-blur.
6. Keep attribution for any CC BY/CC BY-SA material (the map already credits OpenStreetMap per its
   [tile usage policy](https://operations.osmfoundation.org/policies/tiles/)).

## 8. Summary table

| Source | What it gives | Access | Licence / terms | Risks | Recommendation |
|---|---|---|---|---|---|
| Research ALPR datasets (CCPD, UFPR-ALPR, RodoSol, OpenALPR benchmarks) | plate images + labels; **no time/place** | free download; UFPR/RodoSol by signed agreement | MIT (CCPD), academic non-commercial (UFPR, RodoSol), AGPL (OpenALPR) | real plates in public display; licence limits | use to **train/evaluate** recognition; do not publish as sightings |
| Nomeroff Net / HF / Kaggle / Roboflow plate sets | RU/UA/EU plates, crops, boxes; no time/place | free | GPL code; dataset terms vary (often CC BY 4.0 / public domain) | unclear dataset terms; overlap/quality | evaluation of RU/UA OCR; check terms per set |
| Wikimedia Commons / Open Images | photos of cars, sometimes EXIF time/GPS | free | CC0 / CC BY / CC BY-SA per file | attribution; real plates | **demo photos** (as now), blur plates, keep `is_test_data` |
| City open data (data.mos.ru, etc.) | camera locations | free, some via API key | open data licences | none for locations | optional map layer only |
| City traffic cameras (TfL, 511NY, Caltrans, Digitraffic, webcams) | periodic low-res traffic snapshots | free API (key for some) | open licences / developer agreements, attribution, no alteration | purpose change if used for ALPR; unreadable plates | **do not use** for sightings |
| Mapillary / KartaView | street images with time + GPS, **plates blurred** | free API | CC BY-SA 4.0 | re-identification is against policy | context imagery at most |
| User uploads (phone / dashcam, EXIF) | photo + plate + time + GPS — everything needed | own app | own terms of service | personal data, faces, fake EXIF, abuse | **main source for real data**, opt-in, with blurring and moderation |
| Commercial ALPR (Flock, Vigilant, DRN, Rekor) | large plate+time+place databases | closed, contracts, vetted customers | proprietary | cost; re-publication not allowed | not applicable |
| Recognition APIs / OSS (fast-alpr, Nomeroff, EasyOCR, YOLO, Plate Recognizer) | plate text from photos | free / freemium | MIT, GPL, AGPL, Apache, commercial | AGPL obligations; accuracy on RU/UA | fast-alpr to start, Nomeroff for Cyrillic |

## 9. Recommended plan

1. **Now:** keep test data as the public demo and treat real uploads as a private beta; optionally enlarge the CC0 photo set from Wikimedia Commons
   (with `blur.sh` and `CREDITS.md`), and add a camera-locations layer only if useful.
2. **Recognition quality:** the `ocr` service already runs fast-alpr; evaluate it against Nomeroff Net on
   UFPR/RodoSol/Nomeroff/Commons samples for RU/UA/EU/CN plates (the region of RU plates is the weak spot per
   OCR_PIPELINE.md) and swap in a better/fine-tuned model behind the same `POST /recognize` contract.
3. **Upload path:** `Ingest` already covers EXIF extraction/stripping, OCR, normalization and quotas; add face and
   other-plate blurring, a pending/pre-moderation state, deduplication and retention (section 6).
4. **Private beta:** real uploads from a small group of signed-in contributors, with retention limits and takedown; write the terms of service and privacy
   notice before any public release.
5. **Public release** of real data only after a legal review for the target country (in the EU — DPIA; most
   likely not feasible), with coarse locations and no data about people.
