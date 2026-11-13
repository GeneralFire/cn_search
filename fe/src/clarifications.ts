// Уточнения на странице номера: плашка под шапкой («Возможно, это А123ВС 77 · 3 предложения · принято»,
// ссылка «искать этот номер») и значок у фиксаций с уточнениями в ленте истории.
import "./clarifications.css";
import type { PlateClarification } from "./api";
import { escapeHtml, formatShortDate, formatTime, plateHtml } from "./ui";
import { clarificationControlsHtml, clarificationStatusHtml } from "./adminpanels";
import { isAdminMode } from "./adminmode";
import { t, tp } from "./i18n/i18n";

const markIcon = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" aria-hidden="true"><path d="M4 20h4L19 9l-4-4L4 16v4z"/></svg>`;

/** Плашка уточнений; нет уточнений — пустая строка. currentType — формат текущего номера: им рисуется
 *  уточнение без формата (тестовые и старые уточнения записаны без него). */
export function clarificationsHtml(list: PlateClarification[], currentType: string): string {
  if (!list.length) return "";
  return `
    <section class="pc" aria-label="${t("pc.title")}">
      ${list.map((c) => rowHtml(c, currentType)).join("")}
    </section>`;
}

function rowHtml(c: PlateClarification, currentType: string): string {
  const plate = plateHtml({ plate: c.plate, plateType: c.plateType || currentType, plateRegion: c.plateRegion });
  return `
    <div class="pc__row${c.accepted ? " is-accepted" : ""}">
      <span class="pc__maybe">${t("pc.maybe")}</span>
      <span class="pc__plate">${plate}</span>
      <span class="pc__count">· ${escapeHtml(tp("pc.offers", c.count))}</span>
      <span class="pc__badge">${t(c.accepted ? "pc.accepted" : "pc.notAccepted")}</span>
      <a class="pc__search" href="/?q=${encodeURIComponent(c.plate)}" data-pc-search="${escapeHtml(c.plate)}">${t("pc.search")}</a>
      ${adminItemsHtml(c)}
    </div>`;
}

/** Админ-режим: уточнения группы по одному — дата, статус, «Принять / Отклонить» (adminpanels.ts;
 *  обработчик — bindClarificationControls в plate.ts). Вне режима — пусто. */
function adminItemsHtml(c: PlateClarification): string {
  if (!isAdminMode() || !c.items.length) return "";
  return `
    <ul class="pc__adm">
      ${c.items
        .map(
          (it) => `<li>
            <span class="pc__adm-when">${escapeHtml(formatShortDate(it.createdAt))}, ${escapeHtml(formatTime(it.createdAt))}</span>
            <span data-adm-clar-status="${escapeHtml(it.id)}">${clarificationStatusHtml(it.status)}</span>
            ${clarificationControlsHtml(it)}
          </li>`
        )
        .join("")}
    </ul>`;
}

/** Значок у фиксации ленты истории, если к ней есть уточнения; иначе пустая строка. */
export function clarifiedMarkHtml(sightingId: string, list: PlateClarification[]): string {
  const plates = list.filter((c) => c.sightingIds.includes(sightingId)).map((c) => c.plate);
  if (!plates.length) return "";
  const title = escapeHtml(t("pc.mark", { plates: plates.join(", ") }));
  return `<span class="pc-mark" title="${title}" aria-label="${title}">${markIcon}</span>`;
}
