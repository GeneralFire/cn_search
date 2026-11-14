// Админ-режим: с верным admin-token по всему интерфейсу появляются скрытые служебные панели
// (страница номера — «Управление», главная — полоса «Админ», см. adminpanels.ts).
//
// Токен — в sessionStorage этой вкладки (getAdminToken/setAdminToken из api). При старте, если токен
// сохранён, все запросы API сразу идут с заголовком admin-token (бэк отдаёт админу и скрытые фиксации),
// а rpc Me подтверждает токен: is_admin — класс is-admin на <html> и панели; иначе токен стирается.
// Без токена ни одного админского элемента в DOM нет.
//
// Вход: форма на /admin; пункт «Админ-режим» в панели настроек на логотипе (виден, только если токен
// сохранён: включить/выключить режим, не забывая токен); скрытый способ — ?admin=1 в адресе
// показывает поле ввода токена.
import "./adminmode.css";
import { checkAdmin, getAdminToken, setAdminHeader, setAdminToken } from "./api";
import { t } from "./i18n/i18n";
import { adminPath } from "./router";

/** Режим выключен вручную (токен сохранён) — флаг в sessionStorage. */
const MODE_OFF_KEY = "admin-mode-off";

/** Значок щита — у всех служебных панелей. */
export const shieldIcon = `<svg class="adm-shield" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="m9 12 2 2 4-4"/></svg>`;

let active = false;
const listeners = new Set<(on: boolean) => void>();

export const isAdminMode = (): boolean => active;

/** Подписка на включение/выключение режима; сразу вызывается с текущим состоянием. */
export function onAdminMode(fn: (on: boolean) => void): () => void {
  listeners.add(fn);
  fn(active);
  return () => listeners.delete(fn);
}

function setActive(on: boolean, token = ""): void {
  setAdminHeader(on ? token : "");
  document.documentElement.classList.toggle("is-admin", on);
  if (active === on) return;
  active = on;
  listeners.forEach((fn) => fn(on));
}

/** Включает режим с токеном, если бэк его подтвердил (rpc Me); false — токен неверный. */
export async function enterAdminMode(token: string): Promise<boolean> {
  if (!token || !(await checkAdmin(token))) return false;
  setAdminToken(token);
  sessionStorage.removeItem(MODE_OFF_KEY);
  setActive(true, token);
  return true;
}

/** Выход: режим выключен, токен забыт. */
export function exitAdminMode(): void {
  setAdminToken("");
  sessionStorage.removeItem(MODE_OFF_KEY);
  setActive(false);
}

/** Старт: сохранённый токен (и режим не выключен вручную) — заголовок сразу, подтверждение — rpc Me. */
async function init(): Promise<boolean> {
  const token = getAdminToken();
  if (!token || sessionStorage.getItem(MODE_OFF_KEY)) return false;
  setAdminHeader(token); // запросы страницы, ушедшие до ответа Me, уже видят скрытые фиксации
  try {
    if (await checkAdmin(token)) {
      setActive(true, token);
      return true;
    }
    setAdminToken(""); // токен больше не верен (сменили ADMIN_TOKEN)
  } catch {
    // бэк недоступен — режим не включаем, токен оставляем до следующей загрузки
  }
  setAdminHeader("");
  return false;
}

/** Готовность: true — режим включён (панели можно рисовать). */
export const adminReady: Promise<boolean> = init();

// ---------------------------------------------------------------- панель настроек

/** Пункт «Админ-режим» в панели настроек — только если токен сохранён; иначе пусто. */
export function adminSettingsHtml(): string {
  if (!getAdminToken()) return "";
  const on = !sessionStorage.getItem(MODE_OFF_KEY);
  return `
    <button class="st__home adm-st" type="button" role="switch" aria-checked="${on}" data-admin-toggle>
      ${shieldIcon}<span class="adm-st__label">${t("adminMode.toggle")}</span>
      <span class="adm-st__state">${t(on ? "adminMode.on" : "adminMode.off")}</span>
    </button>
    <a class="st__home" href="${adminPath()}" data-link>${t("admin.link")}</a>`;
}

/** Переключатель в панели настроек: смена режима перезагружает страницу (выдача — с/без скрытых). */
export function bindAdminSettings(panel: HTMLElement): void {
  panel.querySelector("[data-admin-toggle]")?.addEventListener("click", () => {
    if (sessionStorage.getItem(MODE_OFF_KEY)) sessionStorage.removeItem(MODE_OFF_KEY);
    else sessionStorage.setItem(MODE_OFF_KEY, "1");
    location.reload();
  });
}

// ---------------------------------------------------------------- скрытый вход ?admin=1

/** Форма токена поверх страницы, если в адресе ?admin=1 и режим ещё не включён. */
async function hiddenEntry(): Promise<void> {
  const url = new URL(location.href);
  if (url.searchParams.get("admin") !== "1" || (await adminReady)) return;
  const box = document.createElement("form");
  box.className = "adm-entry";
  box.autocomplete = "off";
  box.innerHTML = `
    <div class="adm-entry__head">${shieldIcon}<span>${t("adminMode.title")}</span>
      <button class="adm-entry__x" type="button" aria-label="${t("modal.close")}">×</button></div>
    <input class="adm-entry__input" type="password" required spellcheck="false" autocomplete="off"
      aria-label="${t("admin.tokenLabel")}" placeholder="${t("admin.tokenLabel")}" />
    <p class="adm-entry__error" role="alert" hidden></p>
    <button class="btn btn--primary" type="submit">${t("admin.login")}</button>`;
  document.body.append(box);
  const $input = box.querySelector<HTMLInputElement>("input")!;
  const $err = box.querySelector<HTMLElement>(".adm-entry__error")!;
  const close = () => {
    box.remove();
    url.searchParams.delete("admin");
    history.replaceState(history.state, "", url);
  };
  box.querySelector(".adm-entry__x")!.addEventListener("click", close);
  $input.focus();
  box.addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      if (await enterAdminMode($input.value.trim())) {
        close();
        location.reload(); // перерисовать выдачу со скрытыми фиксациями и панелями
        return;
      }
      $err.textContent = t("admin.denied");
    } catch (err) {
      $err.textContent = err instanceof Error ? err.message : String(err);
    }
    $err.hidden = false;
  });
}

void hiddenEntry();
