// Общие куски разметки и форматирования для страниц поиска и номера.

export const icon = {
  back: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M15 18l-6-6 6-6"/></svg>`,
  clock: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>`,
  search: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>`,
  plate: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="2" y="7" width="20" height="10" rx="2"/><path d="M16 7v10"/></svg>`,
  clear: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18"/></svg>`,
  pin: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M12 21s-7-6.2-7-11.5A7 7 0 0 1 19 9.5C19 14.8 12 21 12 21z"/><circle cx="12" cy="9.5" r="2.5"/></svg>`,
  cal: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/></svg>`,
  car: `<svg viewBox="0 0 64 32" fill="currentColor" aria-hidden="true"><path d="M10 22h-4a2 2 0 0 1-2-2v-4c0-1.5 1-2.6 2.4-3l6.6-1.8 6-5.4A6 6 0 0 1 23 4.5h14a6 6 0 0 1 4.3 1.8L47 12l9 1.6c1.8.3 3 1.9 3 3.6V20a2 2 0 0 1-2 2h-3"/><circle cx="16" cy="23" r="5"/><circle cx="48" cy="23" r="5"/></svg>`,
};

export function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!
  );
}

const dateFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric", month: "long", year: "numeric", hour: "2-digit", minute: "2-digit",
});
export const formatDate = (iso: string) => (iso ? dateFmt.format(new Date(iso)) : "");

export function plateHtml(plate: string, size: "" | "xl" = ""): string {
  const [num = "", reg = ""] = plate.split(" ");
  return `<span class="plate${size ? ` plate--${size}` : ""}" aria-label="Госномер ${escapeHtml(plate)}"><span class="plate__num">${escapeHtml(num)}</span><span class="plate__reg">${escapeHtml(reg)}<span class="plate__rus">RUS<i class="plate__flag"></i></span></span></span>`;
}

const dayMonthFmt = new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "short" });
const dayMonthYearFmt = new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "short", year: "numeric" });
/** «9 нояб.» в текущем году, «20 окт. 2025 г.» — в другом. */
export function formatShortDate(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  return (d.getFullYear() === new Date().getFullYear() ? dayMonthFmt : dayMonthYearFmt).format(d);
}

const timeFmt = new Intl.DateTimeFormat("ru-RU", { hour: "2-digit", minute: "2-digit" });
export const formatTime = (iso: string) => (iso ? timeFmt.format(new Date(iso)) : "");

const rtf = new Intl.RelativeTimeFormat("ru", { numeric: "auto" });
/** «вчера», «3 дня назад», «2 месяца назад». */
export function formatAgo(iso: string, now = Date.now()): string {
  const sec = (new Date(iso).getTime() - now) / 1000;
  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ["year", 31536000], ["month", 2592000], ["week", 604800], ["day", 86400], ["hour", 3600], ["minute", 60],
  ];
  for (const [unit, size] of units) {
    if (Math.abs(sec) >= size) return rtf.format(Math.round(sec / size), unit);
  }
  return "только что";
}

export const plural = (n: number, one: string, few: string, many: string) => {
  const m10 = n % 10, m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return few;
  return many;
};

/** Миниатюра, при наведении — полное фото (data-full, см. hoverpreview.ts). */
export function photoAttrs(c: { photoUrl?: string; thumbUrl?: string }): { src: string; full: string } {
  return { src: c.thumbUrl || c.photoUrl || "", full: c.photoUrl || "" };
}

/** Логотип — квадратный «мотоциклетный» номер: ПОИСК / АВТО + RUS. */
export const logoHtml = `
  <span class="logo" aria-hidden="true">
    <span class="logo__top">ПОИСК</span>
    <span class="logo__bottom">АВТО<span class="logo__rus">RUS<i class="plate__flag"></i></span></span>
  </span>`;
