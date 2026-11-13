// Страница /cameras — заявки на расположение камер. Карта (OpenStreetMap + Leaflet, грузится лениво,
// как на странице номера) со всеми заявками: цвет маркера — статус. Под картой — список с голосом «+1»
// (один раз на заявку с браузера — localStorage). «Предложить место» включает выбор точки: клик по
// карте ставит маркер (его можно перетащить), «Моё местоположение» — geolocation; затем форма
// (адрес и контакт необязательны) → «Заявка принята».
import "./cameras.css";
import type { LatLng, Map as LeafletMap, Marker } from "leaflet";
import {
  ApiError,
  CAMERA_STATUSES,
  listCameraRequests,
  submitCameraRequest,
  voteCameraRequest,
  type CameraRequest,
  type CameraStatus,
} from "./api";
import { ATTRIBUTION, MAX_ZOOM, TILE_URL } from "./map";
import { t, type Key } from "./i18n/i18n";
import { escapeHtml, formatDate } from "./ui";
import { mountSettings } from "./settings";

const PAGE_SIZE = 100;
/** Вид карты, пока заявок нет: Европа целиком. */
const DEFAULT_VIEW: [number, number, number] = [52, 25, 4];
const PICK_ZOOM = 16;

/** Голоса этого браузера: id заявок в localStorage — повторно «+1» не нажать. */
const VOTES_KEY = "camera-votes";
function votedIds(): Set<string> {
  try {
    return new Set(JSON.parse(localStorage.getItem(VOTES_KEY) ?? "[]") as string[]);
  } catch {
    return new Set();
  }
}
function rememberVote(id: string): void {
  const s = votedIds();
  s.add(id);
  localStorage.setItem(VOTES_KEY, JSON.stringify([...s]));
}

export const cameraStatusLabel = (s: CameraStatus) => t(`cameras.status.${s}` as Key);
export const cameraStatusHtml = (s: CameraStatus) =>
  `<span class="cam-status cam-status--${s}">${escapeHtml(cameraStatusLabel(s))}</span>`;

let L: typeof import("leaflet") | null = null;
let page: AbortController | null = null; // жизнь страницы: отмена — уход со страницы

const state = {
  items: [] as CameraRequest[],
  total: 0,
  next: null as number | null,
  map: null as LeafletMap | null,
  markers: new Map<string, Marker>(),
  picking: false,
  pick: null as Marker | null,
};

let $main: HTMLElement;

/** Уход со страницы: карта уничтожается, запросы отменяются. */
export function leaveCamerasPage(): void {
  page?.abort();
  page = null;
}

/** Рисует страницу заявок в root (заново при каждом входе — данные свежие). */
export function renderCamerasPage(root: HTMLElement): void {
  leaveCamerasPage();
  page = new AbortController();
  const signal = page.signal;
  Object.assign(state, { items: [], total: 0, next: null, map: null, markers: new Map(), picking: false, pick: null });

  root.innerHTML = `
    <header class="cam-bar">
      <div class="cam-bar__settings"></div>
      <h1 class="cam-bar__title">${t("cameras.title")}</h1>
    </header>
    <main class="cam">
      <section class="cam-intro">
        <p class="cam-intro__text">${t("cameras.intro")}</p>
        <button class="btn btn--primary cam-propose" type="button">${t("cameras.propose")}</button>
      </section>
      <ul class="cam-legend" aria-label="${t("cameras.legend")}">
        ${CAMERA_STATUSES.map((s) => `<li><i class="cam-dot cam-dot--${s}"></i>${escapeHtml(cameraStatusLabel(s))}</li>`).join("")}
      </ul>
      <div class="cam-mapbox">
        <div class="cam-map" role="region" aria-label="${t("cameras.mapAria")}"></div>
        <div class="cam-pickbar" hidden>
          <span class="cam-pickbar__text">${t("cameras.pickHint")}</span>
          <button class="btn btn--ghost cam-locate" type="button">${t("cameras.locate")}</button>
        </div>
      </div>
      <form class="cam-form" hidden novalidate>
        <h2 class="cam-form__title">${t("cameras.formTitle")}</h2>
        <p class="cam-form__point" aria-live="polite"></p>
        <label class="cam-field">
          <span>${t("cameras.address")} <small>${t("cameras.optional")}</small></span>
          <input name="address" maxlength="300" autocomplete="street-address" placeholder="${escapeHtml(t("cameras.addressPlaceholder"))}" />
        </label>
        <label class="cam-field">
          <span>${t("cameras.reason")}</span>
          <textarea name="reason" maxlength="1000" rows="3" required placeholder="${escapeHtml(t("cameras.reasonPlaceholder"))}"></textarea>
        </label>
        <label class="cam-field">
          <span>${t("cameras.contact")} <small>${t("cameras.optional")}</small></span>
          <input name="contact" maxlength="200" autocomplete="email" placeholder="${escapeHtml(t("cameras.contactPlaceholder"))}" />
          <small class="cam-field__hint">${t("cameras.contactHint")}</small>
        </label>
        <p class="cam-form__error" role="alert" hidden></p>
        <div class="cam-form__actions">
          <button class="btn btn--ghost cam-cancel" type="button">${t("cameras.cancel")}</button>
          <button class="btn btn--primary cam-send" type="submit">${t("cameras.send")}</button>
        </div>
      </form>
      <div class="cam-done" role="status" hidden>
        <strong>${t("cameras.done")}</strong> <span>${t("cameras.doneHint")}</span>
      </div>
      <div class="cam-head">
        <h2 class="cam-head__title">${t("cameras.listTitle")}</h2>
        <span class="cam-head__count" aria-live="polite"></span>
      </div>
      <ul class="cam-list"></ul>
      <p class="cam-note" hidden></p>
      <div class="more" hidden><button class="more__btn" type="button">${t("more.load")}</button></div>
    </main>`;
  mountSettings(root.querySelector<HTMLElement>(".cam-bar__settings")!);
  $main = root.querySelector<HTMLElement>(".cam")!;
  document.title = `${t("cameras.title")} — ${t("app.title")}`;
  window.scrollTo({ top: 0 });

  $main.querySelector(".cam-propose")!.addEventListener("click", () => setPicking(true));
  $main.querySelector(".cam-cancel")!.addEventListener("click", () => setPicking(false));
  $main.querySelector(".cam-locate")!.addEventListener("click", locate);
  $main.querySelector(".cam-form")!.addEventListener("submit", (e) => {
    e.preventDefault();
    void submit();
  });
  $main.querySelector(".more__btn")!.addEventListener("click", () => void load(signal));
  $main.querySelector(".cam-list")!.addEventListener("click", (e) => {
    const vote = (e.target as Element).closest<HTMLButtonElement>("[data-vote]");
    if (vote) return void doVote(vote);
    const show = (e.target as Element).closest<HTMLElement>("[data-show]");
    if (show) focusOnMap(show.dataset.show!);
  });

  void mountMap(signal);
  void load(signal);
}

// ---------------------------------------------------------------- карта

async function mountMap(signal: AbortSignal): Promise<void> {
  const $canvas = $main.querySelector<HTMLElement>(".cam-map")!;
  try {
    if (!L) {
      const [mod] = await Promise.all([import("leaflet"), import("leaflet/dist/leaflet.css")]);
      L = mod.default ?? mod;
    }
  } catch {
    if (!signal.aborted) $canvas.innerHTML = `<p class="cam-map__error">${t("map.failed")}</p>`;
    return;
  }
  if (signal.aborted || !$canvas.isConnected) return;
  const map = L.map($canvas, { scrollWheelZoom: false, maxZoom: MAX_ZOOM, zoomSnap: 0.5 });
  signal.addEventListener("abort", () => map.remove(), { once: true });
  map.on("focus", () => map.scrollWheelZoom.enable());
  map.on("blur", () => map.scrollWheelZoom.disable());
  L.tileLayer(TILE_URL, { maxZoom: MAX_ZOOM, attribution: ATTRIBUTION }).addTo(map);
  map.setView([DEFAULT_VIEW[0], DEFAULT_VIEW[1]], DEFAULT_VIEW[2]);
  map.on("click", (e) => {
    if (state.picking) placePick(e.latlng);
  });
  state.map = map;
  addMarkers(state.items);
  fitAll();
}

function pinIcon(status: CameraStatus) {
  return L!.divIcon({ className: `cam-pin cam-pin--${status}`, html: "<span></span>", iconSize: [22, 22] });
}

function popupHtml(c: CameraRequest): string {
  return `<div class="cam-pop">${cameraStatusHtml(c.status)}
    <p class="cam-pop__reason">${escapeHtml(c.reason)}</p>
    ${c.address ? `<p class="cam-pop__addr">${escapeHtml(c.address)}</p>` : ""}
    <p class="cam-pop__votes">${escapeHtml(t("cameras.votes", { n: c.votes }))}</p></div>`;
}

function addMarkers(items: CameraRequest[]): void {
  if (!state.map || !L) return;
  for (const c of items) {
    if (state.markers.has(c.id)) continue;
    const m = L.marker([c.lat, c.lon], {
      icon: pinIcon(c.status),
      title: `${cameraStatusLabel(c.status)}: ${c.address || c.reason}`,
      alt: c.address || c.reason,
      riseOnHover: true,
    })
      .bindPopup(popupHtml(c))
      .addTo(state.map);
    m.on("click", () => highlightItem(c.id));
    state.markers.set(c.id, m);
  }
}

function fitAll(): void {
  if (!state.map || !L || !state.items.length) return;
  const b = L.latLngBounds(state.items.map((c) => [c.lat, c.lon] as [number, number]));
  state.map.fitBounds(b, { padding: [32, 32], maxZoom: 13 });
}

function focusOnMap(id: string): void {
  const m = state.markers.get(id);
  if (!m || !state.map) return;
  state.map.setView(m.getLatLng(), Math.max(state.map.getZoom(), 15));
  m.openPopup();
  $main.querySelector(".cam-mapbox")!.scrollIntoView({ behavior: "smooth", block: "nearest" });
}

function highlightItem(id: string): void {
  $main.querySelectorAll(".cam-item.is-hl").forEach((x) => x.classList.remove("is-hl"));
  const $li = $main.querySelector(`.cam-item[data-id="${CSS.escape(id)}"]`);
  $li?.classList.add("is-hl");
}

// ---------------------------------------------------------------- выбор точки и форма

function setPicking(on: boolean): void {
  state.picking = on;
  $main.querySelector<HTMLElement>(".cam-pickbar")!.hidden = !on;
  $main.querySelector<HTMLElement>(".cam-form")!.hidden = !on;
  $main.querySelector<HTMLElement>(".cam-propose")!.hidden = on;
  $main.querySelector(".cam-map")!.classList.toggle("is-picking", on);
  if (on) {
    $main.querySelector<HTMLElement>(".cam-done")!.hidden = true;
    $main.querySelector<HTMLElement>(".cam-form__error")!.hidden = true;
    updatePointText();
    $main.querySelector(".cam-mapbox")!.scrollIntoView({ behavior: "smooth", block: "start" });
  } else {
    state.pick?.remove();
    state.pick = null;
    $main.querySelector<HTMLFormElement>(".cam-form")!.reset();
  }
}

function placePick(ll: LatLng): void {
  if (!state.map || !L) return;
  if (!state.pick) {
    state.pick = L.marker(ll, {
      icon: L.divIcon({ className: "cam-pin cam-pin--pick", html: "<span></span>", iconSize: [30, 30] }),
      draggable: true,
      zIndexOffset: 2000,
      title: t("cameras.pickMarker"),
    }).addTo(state.map);
    state.pick.on("dragend", updatePointText);
  } else {
    state.pick.setLatLng(ll);
  }
  updatePointText();
}

function updatePointText(): void {
  const $p = $main.querySelector<HTMLElement>(".cam-form__point")!;
  const ll = state.pick?.getLatLng();
  $p.classList.toggle("is-empty", !ll);
  $p.textContent = ll
    ? t("cameras.point", { lat: ll.lat.toFixed(5), lon: ll.lng.toFixed(5) })
    : t("cameras.noPoint");
}

function locate(): void {
  const $btn = $main.querySelector<HTMLButtonElement>(".cam-locate")!;
  if (!navigator.geolocation) return showFormError(t("cameras.locateFailed"));
  $btn.disabled = true;
  navigator.geolocation.getCurrentPosition(
    (pos) => {
      $btn.disabled = false;
      if (!state.map || !L) return;
      const ll = L.latLng(pos.coords.latitude, pos.coords.longitude);
      placePick(ll);
      state.map.setView(ll, PICK_ZOOM);
    },
    () => {
      $btn.disabled = false;
      showFormError(t("cameras.locateFailed"));
    },
    { enableHighAccuracy: true, timeout: 10_000 }
  );
}

function showFormError(msg: string): void {
  const $e = $main.querySelector<HTMLElement>(".cam-form__error")!;
  $e.hidden = !msg;
  $e.textContent = msg;
}

async function submit(): Promise<void> {
  const $form = $main.querySelector<HTMLFormElement>(".cam-form")!;
  const ll = state.pick?.getLatLng();
  const get = (n: string) => ($form.elements.namedItem(n) as HTMLInputElement).value.trim();
  if (!ll) return showFormError(t("cameras.noPoint"));
  if (!get("reason")) {
    ($form.elements.namedItem("reason") as HTMLElement).focus();
    return showFormError(t("cameras.needReason"));
  }
  showFormError("");
  const $send = $form.querySelector<HTMLButtonElement>(".cam-send")!;
  $send.disabled = true;
  $send.textContent = t("cameras.sending");
  try {
    const c = await submitCameraRequest({
      lat: ll.lat, lon: ((ll.lng + 540) % 360) - 180, // карту могли прокрутить через антимеридиан
      address: get("address"), reason: get("reason"), contact: get("contact"),
    });
    rememberVote(c.id); // за свою заявку голосовать не нужно
    state.items.unshift(c);
    state.total++;
    $main.querySelector(".cam-list")!.insertAdjacentHTML("afterbegin", itemHtml(c, votedIds()));
    updateCount();
    setPicking(false);
    addMarkers([c]);
    state.markers.get(c.id)?.openPopup();
    $main.querySelector<HTMLElement>(".cam-note")!.hidden = true;
    const $done = $main.querySelector<HTMLElement>(".cam-done")!;
    $done.hidden = false;
    $done.scrollIntoView({ behavior: "smooth", block: "nearest" });
  } catch (e) {
    showFormError(`${t("cameras.failed")}${e instanceof ApiError ? `: ${e.message}` : ""}`);
  } finally {
    $send.disabled = false;
    $send.textContent = t("cameras.send");
  }
}

// ---------------------------------------------------------------- список и голоса

function itemHtml(c: CameraRequest, voted: Set<string>): string {
  const did = voted.has(c.id);
  return `
    <li class="cam-item" data-id="${escapeHtml(c.id)}">
      <i class="cam-dot cam-dot--${c.status}" aria-hidden="true"></i>
      <div class="cam-item__body">
        <div class="cam-item__top">${cameraStatusHtml(c.status)}
          <time class="cam-item__time" datetime="${escapeHtml(c.createdAt)}">${escapeHtml(formatDate(c.createdAt))}</time></div>
        <p class="cam-item__reason">${escapeHtml(c.reason)}</p>
        <button class="cam-item__addr" type="button" data-show="${escapeHtml(c.id)}">${escapeHtml(
          c.address || `${c.lat.toFixed(4)}, ${c.lon.toFixed(4)}`
        )}</button>
      </div>
      <button class="cam-vote${did ? " is-voted" : ""}" type="button" data-vote="${escapeHtml(c.id)}"
        aria-pressed="${did}" ${did ? "disabled" : ""} title="${escapeHtml(t(did ? "cameras.voted" : "cameras.vote"))}">
        <span class="cam-vote__plus">+1</span><span class="cam-vote__n">${c.votes}</span>
      </button>
    </li>`;
}

function updateCount(): void {
  $main.querySelector(".cam-head__count")!.textContent = state.items.length
    ? t("admin.shown", { shown: state.items.length, total: state.total })
    : "";
}

async function load(signal: AbortSignal): Promise<void> {
  const $more = $main.querySelector<HTMLElement>(".more")!;
  const $btn = $more.querySelector<HTMLButtonElement>(".more__btn")!;
  const $note = $main.querySelector<HTMLElement>(".cam-note")!;
  $btn.disabled = true;
  $btn.textContent = t("more.loading");
  try {
    const p = await listCameraRequests({ offset: state.next ?? 0, limit: PAGE_SIZE }, signal);
    if (signal.aborted) return;
    const first = !state.items.length;
    state.items.push(...p.items);
    state.total = p.total;
    state.next = p.nextOffset;
    $main.querySelector(".cam-list")!.insertAdjacentHTML("beforeend", p.items.map((c) => itemHtml(c, votedIds())).join(""));
    addMarkers(p.items);
    if (first) fitAll();
    $note.hidden = state.items.length > 0;
    $note.className = "cam-note";
    $note.textContent = t("cameras.empty");
  } catch (e) {
    if (signal.aborted) return;
    $note.hidden = false;
    $note.className = "cam-note is-error";
    $note.textContent = `${t("error.loadFailed")}: ${e instanceof ApiError ? e.message : t("error.serverDown")}`;
  }
  updateCount();
  $more.hidden = state.next === null;
  $btn.disabled = false;
  $btn.textContent = t("more.load");
}

async function doVote(b: HTMLButtonElement): Promise<void> {
  const id = b.dataset.vote!;
  if (votedIds().has(id)) return;
  b.disabled = true;
  try {
    const c = await voteCameraRequest(id);
    rememberVote(id);
    b.classList.add("is-voted");
    b.setAttribute("aria-pressed", "true");
    b.title = t("cameras.voted");
    b.querySelector(".cam-vote__n")!.textContent = String(c.votes);
    const it = state.items.find((x) => x.id === id);
    if (it) {
      it.votes = c.votes;
      state.markers.get(id)?.setPopupContent(popupHtml(it));
    }
  } catch (e) {
    b.disabled = false;
    b.title = `${t("cameras.voteFailed")}${e instanceof ApiError ? `: ${e.message}` : ""}`;
    b.classList.add("is-error");
  }
}
