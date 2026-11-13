// Карта на странице номера: все фиксации номера точками (OpenStreetMap + Leaflet).
// Маркеры пронумерованы по времени (1 — самая ранняя), линия — маршрут в том же порядке,
// открытая фиксация выделена. Клик по маркеру — переход на фиксацию; наведение на пункт
// ленты «История фиксаций» (li[data-id]) подсвечивает маркер, наведение на маркер — пункт ленты.
// Leaflet грузится лениво (dynamic import) — на главную он не попадает.
import "./map.css";
import type { Map as LeafletMap, Marker } from "leaflet";
import type { Car } from "./api";
import { t } from "./i18n/i18n";
import { navigate, plateHref } from "./router";
import { escapeHtml, formatShortDate, formatTime } from "./ui";

/** Тайлы OpenStreetMap: по их tile usage policy атрибуция обязательна. */
export const TILE_URL = "https://tile.openstreetmap.org/{z}/{x}/{y}.png";
export const ATTRIBUTION = `&copy; <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noopener">OpenStreetMap</a> contributors`;
/** Максимальный зум тайлов OSM. */
export const MAX_ZOOM = 19;
/** Зум для единственной точки и верхняя граница для fitBounds — уровень квартала. */
const POINT_ZOOM = 15;

type Located = Car & { lat: number; lon: number };

const hasCoords = (c: Car): c is Located => c.lat != null && c.lon != null;

/** Разметка блока карты; у номера нет ни одной фиксации с координатами — пустая строка (блок скрыт). */
export function plateMapHtml(items: Car[]): string {
  const located = items.filter(hasCoords).length;
  if (!located) return "";
  const note =
    located < items.length
      ? `<p class="pp-map__note">${escapeHtml(t("map.partial", { shown: located, total: items.length }))}</p>`
      : "";
  return `
    <section class="pp-map" aria-labelledby="pp-map-title">
      <h2 id="pp-map-title" class="pp-map__title">${t("map.title")}</h2>
      <div class="pp-map__canvas" role="region" aria-label="${escapeHtml(t("map.aria"))}"></div>
      ${note}
    </section>`;
}

/** Рисует карту в блоке из plateMapHtml внутри $root. signal — жизнь страницы: при его отмене
 *  (переход на другую фиксацию или главную) карта уничтожается. */
export async function mountPlateMap($root: HTMLElement, items: Car[], currentId: string, signal: AbortSignal): Promise<void> {
  const $canvas = $root.querySelector<HTMLElement>(".pp-map__canvas");
  // По времени: items от новых к старым, на карте нумерация с самой ранней.
  const points = items.filter(hasCoords).reverse();
  if (!$canvas || !points.length) return;

  let L: typeof import("leaflet");
  try {
    const [mod] = await Promise.all([import("leaflet"), import("leaflet/dist/leaflet.css")]);
    L = mod.default ?? mod;
  } catch {
    if (!signal.aborted) $canvas.innerHTML = `<p class="pp-map__error">${t("map.failed")}</p>`;
    return;
  }
  if (signal.aborted || !$canvas.isConnected) return;

  // Колесо мыши не перехватывает прокрутку страницы, пока карта не в фокусе (клик или Tab).
  const map: LeafletMap = L.map($canvas, { scrollWheelZoom: false, maxZoom: MAX_ZOOM, zoomSnap: 0.5 });
  signal.addEventListener("abort", () => map.remove(), { once: true });
  map.on("focus", () => map.scrollWheelZoom.enable());
  map.on("blur", () => map.scrollWheelZoom.disable());

  L.tileLayer(TILE_URL, { maxZoom: MAX_ZOOM, attribution: ATTRIBUTION }).addTo(map);

  const latLngs = points.map((c) => L.latLng(c.lat, c.lon));
  if (points.length > 1) {
    L.polyline(latLngs, { className: "pp-map__route", interactive: false }).addTo(map);
  }

  const markers = new Map<string, Marker>();
  points.forEach((c, i) => {
    const current = c.id === currentId;
    const marker = L.marker(latLngs[i], {
      icon: L.divIcon({
        className: `pp-map__pin${current ? " is-current" : ""}`,
        html: `<span>${i + 1}</span>`,
        iconSize: current ? [34, 34] : [26, 26],
      }),
      title: t("map.marker", { n: i + 1, date: `${formatShortDate(c.createdAt)}, ${formatTime(c.createdAt)}`, address: c.address }),
      alt: c.address,
      riseOnHover: true,
      zIndexOffset: current ? 1000 : 0, // открытая фиксация — поверх соседних
    }).addTo(map);
    if (!current) marker.on("click", () => navigate(plateHref(c.id)));
    marker.on("mouseover", () => highlightItem($root, c.id, true));
    marker.on("mouseout", () => highlightItem($root, c.id, false));
    markers.set(c.id, marker);
  });

  if (points.length === 1) map.setView(latLngs[0], POINT_ZOOM);
  else map.fitBounds(L.latLngBounds(latLngs), { padding: [32, 32], maxZoom: POINT_ZOOM });

  // Пункт ленты → маркер.
  $root.querySelectorAll<HTMLElement>(".tl__item[data-id]").forEach(($li) => {
    const marker = markers.get($li.dataset.id!);
    if (!marker) return;
    const on = () => highlightMarker(marker, true);
    const off = () => highlightMarker(marker, false);
    $li.addEventListener("mouseenter", on);
    $li.addEventListener("mouseleave", off);
    $li.addEventListener("focusin", on);
    $li.addEventListener("focusout", off);
  });
}

function highlightMarker(marker: Marker, on: boolean): void {
  marker.getElement()?.classList.toggle("is-hover", on);
  marker.setZIndexOffset(on ? 2000 : marker.getElement()?.classList.contains("is-current") ? 1000 : 0);
}

function highlightItem($root: HTMLElement, id: string, on: boolean): void {
  $root.querySelector(`.tl__item[data-id="${CSS.escape(id)}"]`)?.classList.toggle("is-hover", on);
}
