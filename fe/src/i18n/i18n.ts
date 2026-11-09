// Локализация интерфейса: языки, переводы (./locales), транслитерация адресов.
// Язык хранится в localStorage; смена языка перезагружает страницу (см. setLang),
// поэтому всё, что зависит от языка, вычисляется один раз при загрузке модуля.
import CyrillicToTranslit from "cyrillic-to-translit-js";
import { transliterate } from "transliteration";

import { de } from "./locales/de";
import { en } from "./locales/en";
import { ru } from "./locales/ru";
import { uk } from "./locales/uk";
import { zh } from "./locales/zh";

export type Lang = "en" | "de" | "zh" | "ru" | "uk";

type Plural = Partial<Record<Intl.LDMLPluralRule, string>> & { other: string };
export type Key = keyof typeof ru;
/** Словарь языка: те же ключи, что у русского; множественные — объектом форм. */
export type Dict = { [K in Key]: (typeof ru)[K] extends string ? string : Plural };

/** Языки в порядке меню. code — как показываем (CH — китайский), intl — локаль для Intl. */
export const LANGS: { lang: Lang; code: string; name: string; intl: string }[] = [
  { lang: "en", code: "EN", name: "English", intl: "en-GB" },
  { lang: "de", code: "DE", name: "Deutsch", intl: "de-DE" },
  { lang: "zh", code: "CH", name: "中文", intl: "zh-CN" },
  { lang: "ru", code: "RU", name: "Русский", intl: "ru-RU" },
  { lang: "uk", code: "UA", name: "Українська", intl: "uk-UA" },
];

const DICTS: Record<Lang, Dict> = { ru, uk, en, de, zh };
const STORAGE_KEY = "lang";

function detect(): Lang {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved && saved in DICTS) return saved as Lang;
  for (const l of navigator.languages ?? []) {
    const base = l.toLowerCase().split("-")[0];
    if (base in DICTS) return base as Lang;
  }
  return "ru";
}

const lang: Lang = detect();
const dict = DICTS[lang];
const info = LANGS.find((l) => l.lang === lang)!;
const pluralRules = new Intl.PluralRules(info.intl);
document.documentElement.lang = lang;

export const getLang = (): Lang => lang;
/** Локаль для Intl (даты, время, «3 дня назад»). */
export const intlLocale = (): string => info.intl;

/** Флаг: страница перезагружена сменой языка — можно восстановить состояние (запрос поиска). */
export const SWITCH_FLAG = "lang-switch";

/** Сохранить язык и перерисовать страницу (перезагрузкой — так обновятся все тексты и форматы). */
export function setLang(next: Lang): void {
  if (next === lang) return;
  localStorage.setItem(STORAGE_KEY, next);
  sessionStorage.setItem(SWITCH_FLAG, "1");
  location.reload();
}

const fill = (s: string, params: Record<string, string | number>) =>
  s.replace(/\{(\w+)\}/g, (m, k: string) => (k in params ? String(params[k]) : m));

/** Перевод строки; {name} подставляется из params ({app} — название приложения). */
export function t(key: Key, params: Record<string, string | number> = {}): string {
  const v = dict[key];
  return fill(typeof v === "string" ? v : v.other, { app: dict["app.title"] as string, ...params });
}

/** Перевод с множественным числом по правилам языка (Intl.PluralRules); {n} — само число. */
export function tp(key: Key, n: number, params: Record<string, string | number> = {}): string {
  const v = dict[key];
  const form = typeof v === "string" ? v : (v[pluralRules.select(n)] ?? v.other);
  return fill(form, { n, ...params });
}

// ---------------------------------------------------------------------------
// Адреса
// ---------------------------------------------------------------------------

const CYRILLIC = new Set<string>(["ru", "uk"]);
const toCyrillic = {
  ru: CyrillicToTranslit({ preset: "ru" }),
  uk: CyrillicToTranslit({ preset: "uk" }),
};

/**
 * Адрес на выбранном языке. Совпадает язык адреса (address_locale) с языком интерфейса — как есть,
 * иначе транслитерация в письменность языка интерфейса:
 * - en, de, zh → латиница (`transliteration`: кириллица и др. → латиница). Для zh тоже латиница:
 *   иероглифическую запись чужих топонимов транслитерацией не получить, а латиницу читают все.
 * - ru, uk → кириллица (`cyrillic-to-translit-js`, обратная транслитерация по пресету языка).
 *   Сначала приводим к ASCII (умлауты, ß) и заменяем буквы без пары в кириллице: w→v, q→k, x→ks.
 *   ru ↔ uk — одна письменность: адрес оставляем как есть.
 */
export function localizeAddress(address: string, addressLocale: string, to: Lang = lang): string {
  const from = (addressLocale || "ru").toLowerCase();
  if (!address || from === to) return address;
  if (CYRILLIC.has(to)) {
    if (CYRILLIC.has(from)) return address;
    const ascii = transliterate(address)
      .replace(/w/g, "v").replace(/W/g, "V")
      .replace(/q/g, "k").replace(/Q/g, "K")
      .replace(/x/g, "ks").replace(/X/g, "Ks");
    const out = toCyrillic[to as "ru" | "uk"].reverse(ascii);
    return to === "uk" ? out.replace(/ґ/g, "г").replace(/Ґ/g, "Г") : out;
  }
  return transliterate(address);
}
