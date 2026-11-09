// Главная без запроса: блок «для вас» (первые 2 ряда номеров «рынка» языка интерфейса) и вкладки стран
// по языкам + «Прочее». Рынок — фильтр market в rpc Search (страна по формату номера, см. proto).
// Выбранная вкладка — в адресе (?tab=de), через replaceState: «Назад» не листает вкладки,
// а возврат со страницы номера приходит на ту же вкладку.
import "./home.css";
import { getLang, LANGS, t, type Key, type Lang } from "./i18n/i18n";
import { escapeHtml } from "./ui";

export type Market = "us" | "de" | "cn" | "ru" | "ua" | "other";

/** Рынок номеров для языка интерфейса. */
const LANG_MARKET: Record<Lang, Market> = { en: "us", de: "de", zh: "cn", ru: "ru", uk: "ua" };

/** Вкладки: языки в порядке меню (EN/DE/CH/RU/UA), затем «Прочее». flag — флаг страны рынка. */
export const TABS: { market: Market; code: string; flag: string }[] = [
  ...LANGS.map((l) => ({ market: LANG_MARKET[l.lang], code: l.code, flag: LANG_MARKET[l.lang] })),
  { market: "other", code: "", flag: "" },
];

/** Рынок языка интерфейса — блок «для вас» и вкладка по умолчанию. */
export const homeMarket = (): Market => LANG_MARKET[getLang()];

/** Вкладки ленты без рынка языка интерфейса — его номера уже в блоке «для вас» выше. */
export const visibleTabs = () => TABS.filter((x) => x.market !== homeMarket());

/** Вкладка по умолчанию — первая из видимых. */
export const defaultTab = (): Market => visibleTabs()[0].market;

export const marketName = (m: Market): string => t(`homeTabs.market.${m}` as Key);

export function flagHtml(m: Market): string {
  return m === "other" ? "" : `<i class="flag flag--m-${m}" aria-hidden="true"></i>`;
}

const isMarket = (v: string | null): v is Market => TABS.some((x) => x.market === v);

/** Вкладка из адреса; нет или чужая — рынок языка. */
export function tabFromUrl(): Market {
  const v = new URLSearchParams(location.search).get("tab");
  return isMarket(v) && v !== homeMarket() ? v : defaultTab();
}

/** Записать вкладку в адрес (вкладка по умолчанию — без параметра), не добавляя запись в историю. */
export function setTabInUrl(m: Market): void {
  if (location.pathname !== "/") return; // только на главной
  const url = new URL(location.href);
  if (m === defaultTab()) url.searchParams.delete("tab");
  else url.searchParams.set("tab", m);
  if (url.href !== location.href) history.replaceState(history.state, "", url);
}

/** Ряд вкладок (role="tablist"); активная — aria-selected. Панель — лента под вкладками (panelId). */
export function tabsHtml(active: Market, panelId: string): string {
  return visibleTabs().map(({ market, code }) => {
    const on = market === active;
    const name = marketName(market);
    const title = market === "other" ? t("homeTabs.otherHint") : name;
    const label = code
      ? `${flagHtml(market)}<span class="tabs__code">${code}</span><span class="tabs__name">${escapeHtml(name)}</span>`
      : `<span class="tabs__other">${escapeHtml(name)}</span>`;
    return `<button class="tabs__tab" type="button" role="tab" id="tab-${market}" data-tab="${market}"
      aria-selected="${on}" aria-controls="${panelId}" tabindex="${on ? 0 : -1}" title="${escapeHtml(title)}">${label}</button>`;
  }).join("");
}

/** Сколько колонок сейчас у CSS-сетки (auto-fill даёт дорожки и без карточек). */
export function gridColumns(grid: HTMLElement): number {
  const cols = getComputedStyle(grid).gridTemplateColumns.split(" ").filter((s) => s && s !== "none");
  return Math.max(1, cols.length);
}
