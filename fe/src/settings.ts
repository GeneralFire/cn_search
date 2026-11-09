// Настройки на логотипе: наведение (и фокус с клавиатуры) открывает панель с выбором языка
// EN/DE/CH/RU/UA, клик по логотипу ведёт на главную. На тач-экранах без наведения первый тап
// открывает панель — в ней есть ссылка «На главную».
import { getLang, LANGS, setLang, t, type Lang } from "./i18n/i18n";
import { logoHtml } from "./ui";


const CLOSE_DELAY_MS = 250; // успеть довести мышь от значка до панели

let seq = 0;

/** Рисует логотип-ссылку на главную с панелью настроек в root. */
export function mountSettings(root: HTMLElement): void {
  const id = `settings-panel-${++seq}`;
  const current = getLang();
  root.innerHTML = `
    <div class="st">
      <a class="st__btn" href="/" data-link aria-haspopup="true" aria-expanded="false" aria-controls="${id}"
        aria-label="${t("nav.home")}">
        ${logoHtml()}
      </a>
      <div class="st__panel" id="${id}" role="group" aria-label="${t("settings.title")}" hidden>
        <div class="st__title">${t("settings.title")}</div>
        <a class="st__home" href="/" data-link>${t("nav.homeShort")}</a>
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
  const $btn = root.querySelector<HTMLAnchorElement>(".st__btn")!;
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
  // Клик по логотипу — на главную (data-link, роутер). На таче без наведения первый тап открывает
  // панель (в ней «На главную»), второй — переходит.
  $btn.addEventListener("click", (e) => {
    if (!hoverable && $panel.hidden) {
      e.preventDefault();
      e.stopPropagation();
      setOpen(true);
      return;
    }
    setOpen(false);
  });
  $panel.querySelector(".st__home")!.addEventListener("click", () => setOpen(false));
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
