// Страница /upload: фото машины → бэк (rpc Ingest) распознаёт номер (OCR), сохраняет фото и создаёт
// фиксацию. Фото — перетаскиванием, выбором файла или камерой телефона (capture=environment);
// время и координаты бэк берёт из EXIF, координаты можно задать кнопкой «Моё место» (геолокация).
import "./upload.css";
import { ApiError, ingest, type PlateCandidate } from "./api";
import { plateHref } from "./router";
import { escapeHtml, plateHtml } from "./ui";
import { t } from "./i18n/i18n";
import { mountSettings } from "./settings";

/** Предел бэка (implementation.MaxPhotoBytes). Больше — сжимаем в браузере (EXIF при этом теряется). */
const MAX_PHOTO_BYTES = 8 << 20;
const SHRINK_MAX_SIDE = 2560;

const state = {
  file: null as Blob | null,
  previewUrl: "",
  coords: null as { lat: number; lon: number } | null,
  busy: false,
};

let $root: HTMLElement;
const $ = <T extends HTMLElement>(sel: string) => $root.querySelector<T>(sel)!;

/** Рисует страницу загрузки в root (разметка создаётся один раз, при повторном входе — сброс формы). */
export function renderUploadPage(root: HTMLElement): void {
  document.title = `${t("upload.title")} — ${t("app.title")}`;
  window.scrollTo({ top: 0 });
  if (root.querySelector(".up")) return;
  $root = root;
  root.innerHTML = `
    <header class="up-bar">
      <div class="up-bar__settings"></div>
      <h1 class="up-bar__title">${t("upload.title")}</h1>
    </header>
    <main class="up">
      <p class="up__hint">${t("upload.hint")}</p>
      <form class="up__form" novalidate>
        <label class="up-drop" tabindex="0">
          <input class="up-drop__input" type="file" accept="image/*" hidden />
          <img class="up-drop__preview" alt="" hidden />
          <span class="up-drop__text">${t("upload.drop")}</span>
        </label>
        <div class="up__row">
          <label class="btn btn--ghost up__camera">
            <input type="file" accept="image/*" capture="environment" hidden />${t("upload.camera")}
          </label>
          <button class="btn btn--ghost up__pick" type="button">${t("upload.pick")}</button>
        </div>
        <label class="up__field">
          <span class="up__label">${t("upload.address")}</span>
          <input class="up__input" name="address" maxlength="300" autocomplete="street-address" placeholder="${t("upload.addressPlaceholder")}" />
        </label>
        <div class="up__row">
          <button class="btn btn--ghost up__geo" type="button">${t("upload.geo")}</button>
          <span class="up__coords" aria-live="polite"></span>
        </div>
        <p class="up__error" role="alert" hidden></p>
        <button class="btn btn--primary up__submit" type="submit" disabled>${t("upload.submit")}</button>
      </form>
      <section class="up-result" aria-live="polite" hidden></section>
    </main>`;
  mountSettings($(".up-bar__settings"));

  const $file = $<HTMLInputElement>(".up-drop__input");
  const $drop = $(".up-drop");
  $(".up__pick").addEventListener("click", () => $file.click());
  $drop.addEventListener("keydown", (e) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      $file.click();
    }
  });
  for (const input of root.querySelectorAll<HTMLInputElement>("input[type=file]")) {
    input.addEventListener("change", () => input.files?.[0] && setFile(input.files[0]));
  }
  $drop.addEventListener("dragover", (e) => {
    e.preventDefault();
    $drop.classList.add("up-drop--over");
  });
  $drop.addEventListener("dragleave", () => $drop.classList.remove("up-drop--over"));
  $drop.addEventListener("drop", (e) => {
    e.preventDefault();
    $drop.classList.remove("up-drop--over");
    const f = e.dataTransfer?.files[0];
    if (f) setFile(f);
  });
  $(".up__geo").addEventListener("click", locate);
  $<HTMLFormElement>(".up__form").addEventListener("submit", (e) => {
    e.preventDefault();
    void submit();
  });
}

function setFile(f: File): void {
  if (!f.type.startsWith("image/")) return showError(t("upload.notImage"));
  state.file = f;
  if (state.previewUrl) URL.revokeObjectURL(state.previewUrl);
  state.previewUrl = URL.createObjectURL(f);
  const $img = $<HTMLImageElement>(".up-drop__preview");
  $img.src = state.previewUrl;
  $img.hidden = false;
  $(".up-drop__text").hidden = true;
  $<HTMLButtonElement>(".up__submit").disabled = false;
  $(".up-result").hidden = true;
  showError("");
}

function locate(): void {
  const $coords = $(".up__coords");
  if (!navigator.geolocation) return void ($coords.textContent = t("upload.geoUnavailable"));
  $coords.textContent = t("upload.geoLocating");
  navigator.geolocation.getCurrentPosition(
    (p) => {
      state.coords = { lat: p.coords.latitude, lon: p.coords.longitude };
      $coords.textContent = `${state.coords.lat.toFixed(5)}, ${state.coords.lon.toFixed(5)}`;
    },
    () => ($coords.textContent = t("upload.geoDenied")),
    { enableHighAccuracy: true, timeout: 15000 }
  );
}

function showError(msg: string): void {
  const $err = $(".up__error");
  $err.textContent = msg;
  $err.hidden = !msg;
}

async function submit(): Promise<void> {
  if (!state.file || state.busy) return;
  state.busy = true;
  const $btn = $<HTMLButtonElement>(".up__submit");
  $btn.disabled = true;
  $btn.textContent = t("upload.sending");
  showError("");
  try {
    const photo = state.file.size > MAX_PHOTO_BYTES ? await shrink(state.file) : state.file;
    const address = $<HTMLInputElement>(".up__input").value.trim();
    const res = await ingest({ photo, address, ...(state.coords ?? {}) });
    showResult(res.sighting.id, res.candidates);
  } catch (e) {
    showError(e instanceof ApiError ? e.message || t("upload.failed") : t("upload.failed"));
  } finally {
    state.busy = false;
    $btn.disabled = false;
    $btn.textContent = t("upload.submit");
  }
}

function showResult(id: string, cands: PlateCandidate[]): void {
  const [best, ...rest] = cands;
  const pct = (c: PlateCandidate) => `${Math.round(c.confidence * 100)}%`;
  const $res = $(".up-result");
  $res.innerHTML = `
    <h2 class="up-result__title">${t("upload.recognized")}</h2>
    <div class="up-result__plate">${plateHtml(best, "xl")}</div>
    <p class="up-result__conf">${t("upload.confidence", { pct: pct(best) })} · <span class="up-result__raw">${escapeHtml(best.raw)}</span></p>
    <a class="btn btn--primary" href="${plateHref(id)}" data-link>${t("upload.open")}</a>
    ${
      rest.length
        ? `<details class="up-result__more"><summary>${t("upload.others")}</summary><ul>${rest
            .map((c) => `<li>${plateHtml(c)} <span>${pct(c)}</span></li>`)
            .join("")}</ul></details>`
        : ""
    }`;
  $res.hidden = false;
}

/** Сжимает фото до SHRINK_MAX_SIDE по большей стороне (JPEG). EXIF не сохраняется. */
async function shrink(f: Blob): Promise<Blob> {
  const bmp = await createImageBitmap(f);
  const k = Math.min(1, SHRINK_MAX_SIDE / Math.max(bmp.width, bmp.height));
  const canvas = document.createElement("canvas");
  canvas.width = Math.round(bmp.width * k);
  canvas.height = Math.round(bmp.height * k);
  canvas.getContext("2d")!.drawImage(bmp, 0, 0, canvas.width, canvas.height);
  return new Promise((resolve, reject) =>
    canvas.toBlob((b) => (b ? resolve(b) : reject(new Error("toBlob"))), "image/jpeg", 0.9)
  );
}
