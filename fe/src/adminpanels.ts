// Скрытые служебные панели админ-режима (adminmode.ts): без режима в DOM ничего не попадает.
//  - страница номера: карточка «Управление» — скрыть/показать каждую фиксацию, жалобы по ней,
//    уточнения к открытой фиксации с «Принять/Отклонить»;
//  - главная: полоса «Админ» со счётчиками новых жалоб и непринятых уточнений, ссылки на /admin.
// Вставляются одной строкой из plate.ts / main.ts.
import {
  ApiError,
  hideSighting,
  listClarifications,
  listComplaints,
  setClarificationStatus,
  type Car,
  type Clarification,
  type ClarificationStatus,
} from "./api";
import { adminReady, isAdminMode, shieldIcon } from "./adminmode";
import { t } from "./i18n/i18n";
import { escapeHtml, formatShortDate, formatTime, plateHtml } from "./ui";

const errText = (e: unknown) => (e instanceof ApiError ? e.message : t("error.serverDown"));

// ---------------------------------------------------------------- уточнения: точка расширения

/** Кнопки «Принять / Отклонить» для уточнения. Точка расширения: где бы ни рисовались уточнения
 *  (страница номера, /admin), разметка отсюда + bindClarificationControls(root) делают их рабочими.
 *  Вне админ-режима — пусто. */
export function clarificationControlsHtml(c: Pick<Clarification, "id" | "status">): string {
  if (!isAdminMode()) return "";
  const btn = (st: ClarificationStatus, key: "adminMode.accept" | "adminMode.reject") =>
    `<button class="btn btn--ghost adm-clar__btn adm-clar__btn--${st}" type="button" data-adm-clar="${escapeHtml(c.id)}"
      data-status="${st}"${c.status === st ? " disabled" : ""}>${t(key)}</button>`;
  return `<span class="adm-clar__ctrl">${btn("accepted", "adminMode.accept")}${btn("not_accepted", "adminMode.reject")}</span>`;
}

export function clarificationStatusHtml(st: ClarificationStatus): string {
  return `<span class="adm-chip adm-chip--${st}">${t(st === "accepted" ? "adminMode.accepted" : "adminMode.notAccepted")}</span>`;
}

/** Клики по кнопкам clarificationControlsHtml внутри root: статус меняется на бэке, кнопки и чип
 *  [data-adm-clar-status="<id>"] (если есть) обновляются. onChange — после успешной смены. */
export function bindClarificationControls(root: HTMLElement, onChange?: (c: Clarification) => void): void {
  root.addEventListener("click", async (e) => {
    const b = (e.target as Element).closest<HTMLButtonElement>("[data-adm-clar]");
    if (!b || !root.contains(b)) return;
    const id = b.dataset.admClar!;
    const btns = root.querySelectorAll<HTMLButtonElement>(`[data-adm-clar="${CSS.escape(id)}"]`);
    btns.forEach((x) => (x.disabled = true));
    try {
      const c = await setClarificationStatus(id, b.dataset.status as ClarificationStatus);
      btns.forEach((x) => (x.disabled = x.dataset.status === c.status));
      root.querySelectorAll(`[data-adm-clar-status="${CSS.escape(id)}"]`).forEach((el) => (el.innerHTML = clarificationStatusHtml(c.status)));
      onChange?.(c);
    } catch (err) {
      btns.forEach((x) => (x.disabled = false));
      b.title = `${t("adminMode.actionFailed")}: ${errText(err)}`;
      b.classList.add("is-error");
    }
  });
}

// ---------------------------------------------------------------- страница номера

function sightingRow(c: Car, current: boolean): string {
  return `
    <li class="adm-row${c.hidden ? " is-hidden" : ""}${current ? " is-current" : ""}" data-sighting="${escapeHtml(c.id)}">
      <span class="adm-row__when">${escapeHtml(formatShortDate(c.createdAt))}, ${escapeHtml(formatTime(c.createdAt))}</span>
      <span class="adm-row__where">${escapeHtml(c.address)}</span>
      <span class="adm-chip adm-chip--hidden"${c.hidden ? "" : " hidden"}>${t("adminMode.hidden")}</span>
      <span class="adm-row__actions">
        <button class="btn btn--ghost" type="button" data-adm-hide="${c.hidden ? "0" : "1"}">
          ${t(c.hidden ? "adminMode.show" : "adminMode.hide")}</button>
        <a class="adm-link" href="/admin?sighting=${encodeURIComponent(c.id)}" data-link>${t("adminMode.complaints")}</a>
      </span>
    </li>`;
}

function clarRow(c: Clarification): string {
  return `
    <li class="adm-row">
      ${plateHtml({ plate: c.plate, plateType: c.plateType, plateRegion: c.plateRegion })}
      <span class="adm-row__when">${escapeHtml(formatShortDate(c.createdAt))}</span>
      <span data-adm-clar-status="${escapeHtml(c.id)}">${clarificationStatusHtml(c.status)}</span>
      <span class="adm-row__actions">${clarificationControlsHtml(c)}</span>
    </li>`;
}

/** Метка скрытой фиксации в ленте страницы (plate.ts рисует .tl__item[data-id]). */
function markTimeline($main: HTMLElement, id: string, hidden: boolean): void {
  $main.querySelector(`.tl__item[data-id="${CSS.escape(id)}"]`)?.classList.toggle("adm-is-hidden", hidden);
}

/** Карточка «Управление» на странице номера (после блока с номером). Только в админ-режиме. */
export async function mountPlateAdmin($main: HTMLElement, items: Car[], currentId: string, signal: AbortSignal): Promise<void> {
  if (!(await adminReady) || signal.aborted) return;
  const shownId = items.some((c) => c.id === currentId) ? currentId : items[0].id;
  const $panel = document.createElement("section");
  $panel.className = "adm-panel";
  $panel.setAttribute("aria-label", t("adminMode.manage"));
  $panel.innerHTML = `
    <header class="adm-panel__head">${shieldIcon}<h2>${t("adminMode.manage")}</h2>
      <span class="adm-panel__tag">${t("adminMode.title")}</span></header>
    <h3 class="adm-panel__sub">${t("adminMode.sightings")}</h3>
    <ul class="adm-rows">${items.map((c) => sightingRow(c, c.id === shownId)).join("")}</ul>
    <h3 class="adm-panel__sub">${t("adminMode.clarifications")}</h3>
    <ul class="adm-rows adm-rows--clar" aria-busy="true"><li class="adm-row adm-row--note">${t("more.loading")}</li></ul>`;
  ($main.querySelector(".pp-hero") ?? $main.firstElementChild)?.after($panel);
  items.forEach((c) => c.hidden && markTimeline($main, c.id, true));

  $panel.addEventListener("click", async (e) => {
    const b = (e.target as Element).closest<HTMLButtonElement>("[data-adm-hide]");
    if (!b) return;
    const $row = b.closest<HTMLElement>("[data-sighting]")!;
    b.disabled = true;
    try {
      const c = await hideSighting($row.dataset.sighting!, b.dataset.admHide === "1");
      const item = items.find((x) => x.id === c.id);
      if (item) item.hidden = c.hidden;
      $row.outerHTML = sightingRow(c, $row.classList.contains("is-current"));
      markTimeline($main, c.id, !!c.hidden);
    } catch (err) {
      b.disabled = false;
      b.title = `${t("adminMode.actionFailed")}: ${errText(err)}`;
      b.classList.add("is-error");
    }
  });
  bindClarificationControls($panel);

  const $clar = $panel.querySelector<HTMLElement>(".adm-rows--clar")!;
  try {
    const page = await listClarifications("", 0, 100, shownId);
    if (signal.aborted) return;
    $clar.innerHTML = page.items.length
      ? page.items.map(clarRow).join("")
      : `<li class="adm-row adm-row--note">${t("adminMode.noClarifications")}</li>`;
  } catch (err) {
    $clar.innerHTML = `<li class="adm-row adm-row--note is-error">${escapeHtml(errText(err))}</li>`;
  }
  $clar.removeAttribute("aria-busy");
}

// ---------------------------------------------------------------- главная

let $strip: HTMLElement | null = null;

/** Полоса «Админ» в начале container (главная). Только в админ-режиме. */
export async function mountHomeAdmin(container: HTMLElement): Promise<void> {
  if (!(await adminReady)) return;
  $strip = document.createElement("aside");
  $strip.className = "adm-strip";
  $strip.setAttribute("aria-label", t("adminMode.title"));
  $strip.innerHTML = `
    ${shieldIcon}<strong class="adm-strip__title">${t("adminMode.title")}</strong>
    <a class="adm-strip__item" href="/admin" data-link>${t("adminMode.newComplaints")}
      <span class="adm-strip__count" data-count="complaints">…</span></a>
    <a class="adm-strip__item" href="/admin?tab=clarifications" data-link>${t("adminMode.newClarifications")}
      <span class="adm-strip__count" data-count="clarifications">…</span></a>`;
  container.prepend($strip);
  void refreshHomeAdmin();
}

/** Обновить счётчики полосы (при возврате на главную). */
export async function refreshHomeAdmin(): Promise<void> {
  if (!$strip) return;
  const set = (name: string, n: number | null) => {
    const el = $strip!.querySelector<HTMLElement>(`[data-count="${name}"]`)!;
    el.textContent = n === null ? "—" : String(n);
    el.classList.toggle("is-zero", n === 0);
  };
  await Promise.all([
    listComplaints("new", 0, 1).then((p) => set("complaints", p.total), () => set("complaints", null)),
    listClarifications("not_accepted", 0, 1).then((p) => set("clarifications", p.total), () => set("clarifications", null)),
  ]);
}
