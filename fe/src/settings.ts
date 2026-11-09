// Настройки — значок-шестерёнка слева сверху. Панель открывается при наведении, по клику
// (для тач-экранов) и при фокусе с клавиатуры. Внутри — выбор языка EN/DE/CH/RU/UA.
import { getLang, LANGS, setLang, t, type Lang } from "./i18n/i18n";

const gear = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.6 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.6-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/></svg>`;

const CLOSE_DELAY_MS = 250; // успеть довести мышь от значка до панели

let seq = 0;

/** Рисует значок настроек с панелью в root. */
export function mountSettings(root: HTMLElement): void {
  const id = `settings-panel-${++seq}`;
  const current = getLang();
  const currentInfo = LANGS.find((l) => l.lang === current)!;
  root.innerHTML = `
    <div class="st">
      <button class="st__btn" type="button" aria-haspopup="true" aria-expanded="false" aria-controls="${id}"
        aria-label="${t("settings.title")}" title="${t("settings.title")}">
        ${gear}<span class="st__badge">${currentInfo.code}</span>
      </button>
      <div class="st__panel" id="${id}" role="group" aria-label="${t("settings.title")}" hidden>
        <div class="st__title">${t("settings.title")}</div>
        <div class="st__label" id="${id}-lang">${t("lang.label")}</div>
        <div class="st__langs" role="radiogroup" aria-labelledby="${id}-lang">
          ${LANGS.map(
            (l) => `
            <button class="st__lang" type="button" role="radio" lang="${l.lang}" data-lang="${l.lang}"
              aria-checked="${l.lang === current}" title="${l.name}">
              <span class="st__code">${l.code}</span><span class="st__name">${l.name}</span>
            </button>`
          ).join("")}
        </div>
      </div>
    </div>`;

  const $box = root.querySelector<HTMLElement>(".st")!;
  const $btn = root.querySelector<HTMLButtonElement>(".st__btn")!;
  const $panel = root.querySelector<HTMLElement>(".st__panel")!;
  let closeTimer: number | undefined;

  const setOpen = (open: boolean) => {
    clearTimeout(closeTimer);
    $panel.hidden = !open;
    $btn.setAttribute("aria-expanded", String(open));
  };
  const closeSoon = () => {
    clearTimeout(closeTimer);
    closeTimer = window.setTimeout(() => setOpen(false), CLOSE_DELAY_MS);
  };

  // Наведение (только для устройств с мышью — на таче mouseenter приходит вместе с кликом).
  const hoverable = matchMedia("(hover: hover)").matches;
  if (hoverable) {
    $box.addEventListener("mouseenter", () => setOpen(true));
    $box.addEventListener("mouseleave", () => {
      if (!$box.contains(document.activeElement)) closeSoon();
    });
  }
  // Клик — для тача; с мышью панель уже открыта наведением, клик её не закрывает.
  $btn.addEventListener("click", () => setOpen(hoverable ? true : $panel.hidden));
  // Клавиатура: фокус внутри — открыто, ушёл наружу — закрыто.
  $box.addEventListener("focusin", () => setOpen(true));
  $box.addEventListener("focusout", (e) => {
    if (!$box.contains(e.relatedTarget as Node)) closeSoon();
  });
  $box.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !$panel.hidden) {
      e.preventDefault();
      setOpen(false);
      $btn.focus();
      // focusin на кнопке откроет снова — закрываем после него.
      setTimeout(() => setOpen(false));
    }
  });
  document.addEventListener("mousedown", (e) => {
    if (!$box.contains(e.target as Node)) setOpen(false);
  });

  $panel.addEventListener("click", (e) => {
    const b = (e.target as Element).closest<HTMLButtonElement>(".st__lang");
    if (b) setLang(b.dataset.lang as Lang);
  });
}
