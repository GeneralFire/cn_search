// Настройки на логотипе: наведение (и фокус с клавиатуры) открывает панель с выбором языка
// EN/DE/CH/RU/UA, клик по логотипу ведёт на главную. На тач-экранах без наведения первый тап
// открывает панель — в ней есть ссылка «На главную».
import { getLang, LANGS, setLang, t, type Lang } from "./i18n/i18n";
import { logoHtml } from "./ui";
import { adminSettingsHtml, bindAdminSettings } from "./adminmode";
import { getTheme, setTheme, THEMES, type Theme } from "./theme";
import { mountAccountSection, mountProBadge } from "./auth";
import { mountApiTokenSection } from "./apitoken-settings";

// ---------- Секция «Тема» (переключатель auto/light/dark) ----------

const THEME_ICONS: Record<Theme, string> = {
  light: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></svg>`,
  auto: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="M12 3a9 9 0 0 1 0 18z" fill="currentColor"/></svg>`,
  dark: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" aria-hidden="true"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/></svg>`,
};

function themeSectionHtml(id: string): string {
  const current = getTheme();
  return `
        <div class="st__label st__label--theme" id="${id}-theme">${t("theme.label")}</div>
        <div class="th" role="radiogroup" aria-labelledby="${id}-theme">
          ${THEMES.map(
            (th) => `
            <button class="th__opt" type="button" role="radio" data-theme-opt="${th}" aria-checked="${th === current}"
              title="${th === "auto" ? t("theme.autoHint") : t(`theme.${th}`)}">
              ${THEME_ICONS[th]}<span>${t(`theme.${th}`)}</span>
            </button>`
          ).join("")}
        </div>`;
}

function bindThemeSection(panel: HTMLElement): void {
  panel.addEventListener("click", (e) => {
    const b = (e.target as Element).closest<HTMLButtonElement>(".th__opt");
    if (!b) return;
    setTheme(b.dataset.themeOpt as Theme);
    panel.querySelectorAll<HTMLButtonElement>(".th__opt").forEach((o) =>
      o.setAttribute("aria-checked", String(o === b))
    );
  });
}



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
        <a class="st__home" href="/upload" data-link>${t("upload.link")}</a>
        ${adminSettingsHtml()}
        <a class="st__home" href="/cameras" data-link>${t("cameras.link")}</a>
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
        ${themeSectionHtml(id)}
        <div class="st-acc"></div>
        <div class="stk"></div>
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
  $panel.querySelectorAll(".st__home").forEach((a) => a.addEventListener("click", () => setOpen(false)));
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
  bindThemeSection($panel);
  bindAdminSettings($panel);
  // Секция «Аккаунт» и значок PRO у логотипа — auth.ts.
  mountAccountSection($panel.querySelector<HTMLElement>(".st-acc")!, () => setOpen(false));
  mountProBadge($box);
  mountApiTokenSection($panel.querySelector<HTMLElement>(".stk")!);
}
