// Фильтры выдачи под строкой поиска: период («Сегодня / 7 дней / 30 дней / Всё»), страна (рынок номера)
// и «Только с координатами». Состояние — в адресе (?since=7d&market=de&geo=1, рядом с ?q= и ?tab=),
// через replaceState, как запрос и вкладка: F5 и «Назад» со страницы номера возвращают те же фильтры.
// К выдаче поиска применяются все; на главной без запроса — период и координаты к ленте вкладки
// (страну там выбирают вкладки, блок «для вас» не фильтруется).
import "./filters.css";
import type { CarsearchSearchRequest } from "./api/generated";
import { flagHtml, marketName, type Market } from "./home";
import { t, type Key } from "./i18n/i18n";
import { escapeHtml } from "./ui";

export type Period = "today" | "7d" | "30d" | "";

export interface Filters {
  period: Period; // "" — всё время
  market: Market | ""; // "" — все страны
  geo: boolean; // только фиксации с координатами
}

const PERIODS: Period[] = ["today", "7d", "30d", ""];
const MARKETS: Market[] = ["ru", "ua", "de", "us", "cn", "other"];

export const noFilters = (): Filters => ({ period: "", market: "", geo: false });

/** Фильтры из адреса; неизвестные значения — как «без фильтра». */
export function filtersFromUrl(): Filters {
  const p = new URLSearchParams(location.search);
  const since = p.get("since") ?? "";
  const market = p.get("market") ?? "";
  return {
    period: (PERIODS as string[]).includes(since) ? (since as Period) : "",
    market: (MARKETS as string[]).includes(market) ? (market as Market) : "",
    geo: p.get("geo") === "1",
  };
}

/** Записать фильтры в адрес (без фильтра — без параметра), не добавляя запись в историю. */
export function setFiltersInUrl(f: Filters): void {
  if (location.pathname !== "/") return;
  const url = new URL(location.href);
  const set = (k: string, v: string) => (v ? url.searchParams.set(k, v) : url.searchParams.delete(k));
  set("since", f.period);
  set("market", f.market);
  set("geo", f.geo ? "1" : "");
  if (url.href !== location.href) history.replaceState(history.state, "", url);
}

export const sameFilters = (a: Filters, b: Filters) => a.period === b.period && a.market === b.market && a.geo === b.geo;

/** Сколько фильтров задано; home — главная без запроса (страна там не применяется). */
export const activeCount = (f: Filters, home = false) =>
  (f.period ? 1 : 0) + (f.market && !home ? 1 : 0) + (f.geo ? 1 : 0);

/** Начало периода: «сегодня» — с местной полуночи, иначе N суток назад от текущего момента. */
function sinceOf(p: Period, now = new Date()): string | undefined {
  if (!p) return undefined;
  if (p === "today") return new Date(now.getFullYear(), now.getMonth(), now.getDate()).toISOString();
  const days = p === "7d" ? 7 : 30;
  return new Date(now.getTime() - days * 86_400_000).toISOString();
}

/** Поля SearchRequest для фильтров; home — без страны (её задаёт вкладка). */
export function filtersRequest(f: Filters, home: boolean): Partial<CarsearchSearchRequest> {
  const req: Partial<CarsearchSearchRequest> = {};
  const since = sinceOf(f.period);
  if (since) req.since = since;
  if (f.geo) req.hasCoords = true;
  if (f.market && !home) req.market = f.market;
  return req;
}

function chip(group: string, value: string, label: string, on: boolean, title = label): string {
  return `<button class="flt__chip" type="button" data-${group}="${value}" aria-pressed="${on}"
    title="${escapeHtml(title)}">${label}</button>`;
}

/** Разметка панели; home — главная без запроса: выбор страны скрыт. */
function panelHtml(f: Filters, home: boolean, open: boolean): string {
  const n = activeCount(f, home);
  const periods = PERIODS.map((p) => chip("period", p, escapeHtml(t(`filters.period.${p || "all"}` as Key)), f.period === p)).join("");
  const markets = [
    chip("market", "", escapeHtml(t("filters.market.all")), f.market === ""),
    ...MARKETS.map((m) =>
      chip("market", m, m === "other" ? escapeHtml(marketName(m)) : `${flagHtml(m)}<span>${m.toUpperCase()}</span>`,
        f.market === m, marketName(m))),
  ].join("");
  return `
    <button class="flt__toggle" type="button" aria-expanded="${open}" aria-controls="flt-body">
      ${escapeHtml(t("filters.title"))}${n ? ` <span class="flt__badge">${n}</span>` : ""}
    </button>
    <div class="flt__body" id="flt-body">
      <div class="flt__group" role="group" aria-label="${escapeHtml(t("filters.period"))}">${periods}</div>
      ${home ? "" : `<div class="flt__group" role="group" aria-label="${escapeHtml(t("filters.market"))}">${markets}</div>`}
      <label class="flt__geo"><input type="checkbox" data-geo ${f.geo ? "checked" : ""} />
        <span class="flt__switch" aria-hidden="true"></span>${escapeHtml(t("filters.geo"))}</label>
      ${n ? `<button class="flt__reset" type="button" data-reset>${escapeHtml(t("filters.reset"))}</button>` : ""}
    </div>`;
}

/** Панель фильтров в el: onChange вызывается после каждого изменения (фильтры уже в адресе). */
export function mountFilters(el: HTMLElement, onChange: (f: Filters) => void) {
  let f = filtersFromUrl();
  let home = true;
  let open = false; // раскрыта ли панель на узком экране
  el.classList.add("flt");

  const render = () => {
    el.innerHTML = panelHtml(f, home, open);
    el.classList.toggle("is-open", open);
  };
  const apply = (next: Filters) => {
    f = next;
    setFiltersInUrl(f);
    render();
    onChange(f);
  };

  el.addEventListener("click", (e) => {
    const target = e.target as Element;
    const b = target.closest<HTMLButtonElement>("button");
    if (!b) return;
    if (b.classList.contains("flt__toggle")) {
      open = !open;
      render();
    } else if (b.dataset.period !== undefined) apply({ ...f, period: b.dataset.period as Period });
    else if (b.dataset.market !== undefined) apply({ ...f, market: b.dataset.market as Market | "" });
    else if (b.dataset.reset !== undefined) apply(noFilters());
  });
  el.addEventListener("change", (e) => {
    const input = e.target as HTMLInputElement;
    if (input.dataset.geo !== undefined) apply({ ...f, geo: input.checked });
  });

  render();
  return {
    get: () => f,
    /** Режим главной (без запроса) — скрыть выбор страны. */
    setHome(h: boolean) {
      if (h === home) return;
      home = h;
      render();
    },
    /** Подхватить фильтры из адреса (переход по истории); true — изменились. */
    syncFromUrl(): boolean {
      const next = filtersFromUrl();
      if (sameFilters(next, f)) return false;
      f = next;
      render();
      return true;
    },
  };
}
