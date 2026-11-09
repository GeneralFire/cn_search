// Тема оформления: auto (по системе) / light / dark. Выбор — в localStorage («theme»), применяется
// атрибутом data-theme на <html> (палитры — theme.css). До загрузки скриптов тему ставит inline-скрипт
// в index.html, чтобы не было вспышки светлого фона.
import "./theme.css";

export type Theme = "auto" | "light" | "dark";
export const THEMES: Theme[] = ["light", "auto", "dark"];

const STORAGE_KEY = "theme";
const darkMq = matchMedia("(prefers-color-scheme: dark)");
const root = document.documentElement;

function load(): Theme {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (v === "light" || v === "dark") return v;
  } catch {
    /* localStorage недоступен — auto */
  }
  return "auto";
}

let theme: Theme = load();

export const getTheme = (): Theme => theme;
/** Фактическая тема: для auto — по системной настройке. */
export const resolvedTheme = (): "light" | "dark" =>
  theme === "auto" ? (darkMq.matches ? "dark" : "light") : theme;

function apply(): void {
  root.dataset.theme = theme;
  // color-scheme из inline-скрипта больше не нужен — дальше его задаёт theme.css.
  root.style.removeProperty("color-scheme");
  window.dispatchEvent(new CustomEvent("themechange", { detail: resolvedTheme() }));
}

export function setTheme(next: Theme): void {
  theme = next;
  try {
    if (next === "auto") localStorage.removeItem(STORAGE_KEY);
    else localStorage.setItem(STORAGE_KEY, next);
  } catch {
    /* не сохранили — тема действует до перезагрузки */
  }
  apply();
}

// auto: палитру переключает @media в theme.css, а слушателям сообщаем о смене на лету.
darkMq.addEventListener("change", () => {
  if (theme === "auto") apply();
});

apply();
