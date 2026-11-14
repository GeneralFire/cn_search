// Кнопка «Скачать» на странице номера: выпадающее меню CSV / GeoJSON по уже загруженным фиксациям.
import "./export.css";
import type { Car } from "./api";
import { t } from "./i18n/i18n";

const downloadIcon = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M12 15V3M7 10l5 5 5-5"/><path d="M5 17v2a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-2"/></svg>`;

/** Колонки CSV — в этом порядке. */
const CSV_HEADER = ["id", "plate", "created_at", "address", "lat", "lon", "photo_url"];

/** Кнопка с меню для шапки страницы номера (рядом с «Поделиться»). Обработчики вешает bindExport. */
export function exportButtonHtml(items: Car[] = []): string {
  // Аноним: координаты скрыты — GeoJSON недоступен (пункт неактивен с подсказкой «войдите»), CSV без lat/lon.
  const locked = items.some((c) => c.coordsHidden);
  const geo = locked
    ? `<button class="ex__item" type="button" role="menuitem" disabled aria-disabled="true" title="${t("coords.exportHint")}">${t("export.geojson")}</button>`
    : `<button class="ex__item" type="button" role="menuitem" data-export-format="geojson">${t("export.geojson")}</button>`;
  return `<div class="ex" data-export>
      <button class="btn btn--ghost pp-share ex__toggle" type="button" aria-haspopup="menu" aria-expanded="false" data-export-toggle>${downloadIcon}<span>${t("export.button")}</span></button>
      <div class="ex__menu" role="menu">
        <button class="ex__item" type="button" role="menuitem" data-export-format="csv">${t("export.csv")}</button>
        ${geo}
      </div>
    </div>`;
}

/** Вешает обработчики на меню внутри root. items — те же фиксации, что рисует история номера. */
export function bindExport(root: HTMLElement, items: Car[]): void {
  const $wrap = root.querySelector<HTMLElement>("[data-export]");
  const $toggle = $wrap?.querySelector<HTMLButtonElement>("[data-export-toggle]");
  if (!$wrap || !$toggle) return;

  const setOpen = (open: boolean) => {
    $wrap.classList.toggle("is-open", open);
    $toggle.setAttribute("aria-expanded", String(open));
  };
  $toggle.addEventListener("click", () => setOpen(!$wrap.classList.contains("is-open")));
  // Меню закрывается при уходе фокуса за пределы кнопки и меню, по Escape — с возвратом фокуса на кнопку.
  $wrap.addEventListener("focusout", (e) => {
    if (!$wrap.contains(e.relatedTarget as Node | null)) setOpen(false);
  });
  $wrap.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      setOpen(false);
      $toggle.focus();
    }
  });
  $wrap.querySelectorAll<HTMLButtonElement>("[data-export-format]").forEach(($item) =>
    $item.addEventListener("click", () => {
      setOpen(false);
      exportPlate(items, $item.dataset.exportFormat === "geojson" ? "geojson" : "csv");
    })
  );
}

function exportPlate(items: Car[], format: "csv" | "geojson"): void {
  const base = `plate-${(items[0]?.plate ?? "").replace(/\s+/g, "")}`;
  if (format === "csv") download(`${base}.csv`, toCsv(items), "text/csv;charset=utf-8");
  else download(`${base}.geojson`, toGeoJson(items), "application/geo+json;charset=utf-8");
}

/** CSV: все значения в кавычках, " внутри удваивается; BOM — чтобы Excel открыл UTF-8 (кириллицу). */
function toCsv(items: Car[]): string {
  const cell = (v: string | number | null | undefined) => `"${String(v ?? "").replace(/"/g, '""')}"`;
  // Аноним: колонок lat/lon нет совсем.
  const noCoords = items.some((c) => c.coordsHidden);
  const header = noCoords ? CSV_HEADER.filter((h) => h !== "lat" && h !== "lon") : CSV_HEADER;
  const rows = items.map((c) =>
    (noCoords ? [c.id, c.plate, c.createdAt, c.address, c.photoUrl] : [c.id, c.plate, c.createdAt, c.address, c.lat, c.lon, c.photoUrl])
      .map(cell)
      .join(",")
  );
  return "\uFEFF" + [header.join(","), ...rows].join("\r\n") + "\r\n";
}

/** GeoJSON: точка [lon, lat] на каждую фиксацию с координатами; без координат — пропускаем. */
function toGeoJson(items: Car[]): string {
  const features = items
    .filter((c): c is Car & { lat: number; lon: number } => typeof c.lat === "number" && typeof c.lon === "number")
    .map((c) => ({
      type: "Feature",
      geometry: { type: "Point", coordinates: [c.lon, c.lat] },
      properties: { id: c.id, plate: c.plate, created_at: c.createdAt, address: c.address, photo_url: c.photoUrl || null },
    }));
  return JSON.stringify({ type: "FeatureCollection", features }, null, 2);
}

/** Скачивание через Blob и временную ссылку <a download>. */
function download(name: string, content: string, type: string): void {
  const url = URL.createObjectURL(new Blob([content], { type }));
  const $a = document.createElement("a");
  $a.href = url;
  $a.download = name;
  document.body.append($a);
  $a.click();
  $a.remove();
  // Сразу отзывать URL нельзя: некоторые браузеры ещё не успели начать загрузку.
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}
