import { plateHref } from "./router";
import "./plate-formats.css";

// Общие куски разметки и форматирования для страниц поиска и номера.
import { getLang, intlLocale, t } from "./i18n/i18n";

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

// Даты и время — по языку интерфейса (он меняется только с перезагрузкой страницы).
const LOCALE = intlLocale();

const dateFmt = new Intl.DateTimeFormat(LOCALE, {
  day: "numeric", month: "long", year: "numeric", hour: "2-digit", minute: "2-digit",
});
export const formatDate = (iso: string) => (iso ? dateFmt.format(new Date(iso)) : "");

/** Что нужно, чтобы нарисовать номер. type: "ru" | "eu" | "us" | "ua" | "cn" | "" (неизвестен). */
export interface PlateLike {
  plate: string;
  plateType?: string;
  plateRegion?: string;
}

const US_STATES: Record<string, string> = {
  CA: "California", NY: "New York", TX: "Texas", FL: "Florida", WA: "Washington", IL: "Illinois", NV: "Nevada",
};

/** Номерной знак в формате его страны: RU, EU (синяя полоса со звёздами), US (штат сверху),
 *  UA (синяя полоса с флагом и «UA»), CN (синий знак с белыми символами), иначе — нейтральный. */
export function plateHtml(c: PlateLike | string, size: "" | "xl" = ""): string {
  const { plate, plateType = "ru", plateRegion = "" } = typeof c === "string" ? { plate: c } : c;
  const label = `aria-label="${escapeHtml(t("plate.aria", { plate }))}"`;
  const xl = size ? ` plate--${size}` : "";
  switch (plateType) {
    case "ru": {
      const [num = "", reg = ""] = plate.split(" ");
      if (!reg) break; // не похоже на RU — рисуем нейтрально
      return `<span class="plate${xl}" ${label}><span class="plate__num">${escapeHtml(num)}</span><span class="plate__reg">${escapeHtml(reg)}<span class="plate__rus">RUS<i class="plate__flag"></i></span></span></span>`;
    }
    case "eu":
      return `<span class="plate-eu${xl}" ${label}><span class="plate-eu__band"><i class="plate-eu__stars"></i>${escapeHtml(plateRegion)}</span><span class="plate-eu__num">${escapeHtml(plate)}</span></span>`;
    case "us":
      return `<span class="plate-us${xl}" ${label}><span class="plate-us__state">${escapeHtml(US_STATES[plateRegion] ?? plateRegion)}</span><span class="plate-us__num">${escapeHtml(plate)}</span></span>`;
    case "ua":
      return `<span class="plate-ua${xl}" ${label}><span class="plate-ua__band"><i class="flag flag--uk plate-ua__flag"></i>UA</span><span class="plate-ua__num">${escapeHtml(plate)}</span></span>`;
    case "cn":
      return `<span class="plate-cn${xl}" ${label}><span class="plate-cn__num">${escapeHtml(plate)}</span></span>`;
  }
  return `<span class="plate-any${xl}" ${label}>${escapeHtml(plate)}</span>`;
}

const dayMonthFmt = new Intl.DateTimeFormat(LOCALE, { day: "numeric", month: "short" });
const dayMonthYearFmt = new Intl.DateTimeFormat(LOCALE, { day: "numeric", month: "short", year: "numeric" });
/** «9 нояб.» в текущем году, «20 окт. 2025 г.» — в другом. */
export function formatShortDate(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  return (d.getFullYear() === new Date().getFullYear() ? dayMonthFmt : dayMonthYearFmt).format(d);
}

const timeFmt = new Intl.DateTimeFormat(LOCALE, { hour: "2-digit", minute: "2-digit" });
export const formatTime = (iso: string) => (iso ? timeFmt.format(new Date(iso)) : "");

const rtf = new Intl.RelativeTimeFormat(LOCALE, { numeric: "auto" });
/** «вчера», «3 дня назад», «2 месяца назад». */
export function formatAgo(iso: string, now = Date.now()): string {
  const sec = (new Date(iso).getTime() - now) / 1000;
  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ["year", 31536000], ["month", 2592000], ["week", 604800], ["day", 86400], ["hour", 3600], ["minute", 60],
  ];
  for (const [unit, size] of units) {
    if (Math.abs(sec) >= size) return rtf.format(Math.round(sec / size), unit);
  }
  return t("time.justNow");
}

/** Миниатюра, при наведении — полное фото (data-full, см. hoverpreview.ts). */
export function photoAttrs(c: { photoUrl?: string; thumbUrl?: string }): { src: string; full: string } {
  return { src: c.thumbUrl || c.photoUrl || "", full: c.photoUrl || "" };
}

/** Логотип — квадратный «мотоциклетный» номер: ПОИСК / НОМЕР + флаг (подписи — на языке интерфейса). */
/** Логотип — квадратный «мотоциклетный» номер (ПОИСК / НОМЕР) с флагом выбранного языка. */
export function logoHtml(): string {
  return `
  <span class="logo" aria-hidden="true">
    <span class="logo__top">${t("logo.top")}</span>
    <span class="logo__bottom">${t("logo.bottom")}<i class="flag flag--${getLang()}"></i></span>
  </span>`;
}

/** Компактная карточка фиксации (блок «Свежие фиксации» на странице номера): вся — одна ссылка на номер. */
export function miniCardHtml(
  c: PlateLike & { id: string; address: string; createdAt: string; photoUrl?: string; thumbUrl?: string },
  i: number
): string {
  const { src } = photoAttrs(c);
  const photo = src ? `<img src="${escapeHtml(src)}" alt="" loading="lazy" />` : icon.car;
  return `
    <a class="mini" href="${plateHref(c.id)}" data-link style="animation-delay:${Math.min(i, 14) * 25}ms"
       aria-label="${escapeHtml(t("card.allSightings", { plate: c.plate }))}">
      <span class="mini__photo">${photo}</span>
      <span class="mini__body">
        <span class="mini__plate">${plateHtml(c)}</span>
        <span class="mini__where" title="${escapeHtml(c.address)}">${escapeHtml(c.address)}</span>
        <time class="mini__when" datetime="${escapeHtml(c.createdAt)}">${escapeHtml(formatShortDate(c.createdAt))}, ${escapeHtml(formatTime(c.createdAt))}</time>
      </span>
    </a>`;
}

/** Заглушка компактной карточки на время загрузки. */
export const miniSkeletonHtml = `
  <div class="mini mini--skeleton" aria-hidden="true">
    <span class="mini__photo skeleton"></span>
    <span class="mini__body"><span class="skeleton mini__sk-plate"></span><span class="skeleton mini__sk-line"></span><span class="skeleton mini__sk-line mini__sk-line--short"></span></span>
  </div>`;

/** Карточка фиксации: вся — ссылка на страницу номера; clarify — кнопка «Уточнить» (модалка на главной). */
export function cardHtml(
  c: PlateLike & { id: string; address: string; createdAt: string; photoUrl?: string; thumbUrl?: string },
  i: number,
  opts: { clarify?: boolean } = { clarify: true }
): string {
  const city = c.address.split(",")[0] ?? "";
  const { src } = photoAttrs(c); // в карточках — только миниатюра, без превью по наведению
  const photo = src ? `<img src="${escapeHtml(src)}" alt="" loading="lazy" />` : icon.car;
  return `
    <article class="card" style="animation-delay:${Math.min(i, 14) * 30}ms">
      <a class="card__link" href="${plateHref(c.id)}" data-link aria-label="${escapeHtml(t("card.allSightings", { plate: c.plate }))}"></a>
      <a class="card__photo" href="${plateHref(c.id)}" data-link tabindex="-1">${photo}<span class="card__badge">${escapeHtml(city)}</span></a>
      <div class="card__body">
        <p class="card__address">${icon.pin}${escapeHtml(c.address)}</p>
        <div class="card__row">${plateHtml(c)}</div>
        <div class="card__foot">
          <span class="card__date">${icon.cal}<time datetime="${escapeHtml(c.createdAt)}">${formatDate(c.createdAt)}</time></span>
          ${opts.clarify ? `<button class="btn btn--ghost" type="button" data-clarify="${escapeHtml(c.id)}">${t("card.clarify")}</button>` : ""}
        </div>
      </div>
    </article>`;
}
