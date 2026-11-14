// Админка /admin: вкладка «Журнал» — действия админов (скрытие, разбор уточнений и жалоб, токены…):
// время, действие, объект, детали «было → стало», request_id. Фильтр по действию, «Загрузить ещё».
// Подключается в реестр admintabs.ts; оболочку вкладок и вход рисует admin.ts.
import "./admin-audit.css";
import { ApiError, listAdminAudit, type AuditEntry } from "./api";
import { plateHref } from "./router";
import { showErrorIn } from "./states";
import { escapeHtml, formatDate } from "./ui";
import { t, type Key } from "./i18n/i18n";
import { registerAdminTab } from "./admintabs";

const PAGE_SIZE = 30;
const ACTIONS = [
  "hide_sighting",
  "show_sighting",
  "restore_photo",
  "set_clarification_status",
  "resolve_complaint",
  "set_camera_status",
  "create_api_token",
  "revoke_api_token",
];
let seq = 0; // номер отрисовки: ответы устаревших запросов игнорируются

registerAdminTab({
  id: "audit",
  label: "audit.tab",
  order: 60,
  render: ($main, onDenied) => {
    seq++;
    renderAudit($main, onDenied);
  },
});

// Ключ i18n, если он есть; иначе — как пришло с бэка (новое действие, не добавленное в локали).
const label = (prefix: string, v: string): string => {
  const key = `${prefix}.${v}` as Key;
  const s = t(key);
  return s === key ? v : s;
};

function renderAudit($main: HTMLElement, onDenied: () => void): void {
  $main.innerHTML = `
    <div class="adm-head">
      <label class="au-filter"><span>${t("audit.filter")}</span>
        <select class="au-select">
          <option value="">${t("audit.all")}</option>
          ${ACTIONS.map((a) => `<option value="${a}">${escapeHtml(label("audit.action", a))}</option>`).join("")}
        </select></label>
      <span class="adm-count" aria-live="polite"></span>
    </div>
    <div class="adm-table-wrap">
      <table class="adm-table au-table">
        <thead><tr>
          <th>${t("audit.colTime")}</th><th>${t("audit.colAction")}</th><th>${t("audit.colTarget")}</th>
          <th>${t("audit.colDetails")}</th>
        </tr></thead>
        <tbody></tbody>
      </table>
    </div>
    <p class="adm-note" hidden></p>
    <div class="more" hidden><button class="more__btn" type="button">${t("more.load")}</button></div>`;

  const my = seq;
  let action = "";
  let next: number | null = null;
  let shown = 0;
  let loadSeq = 0;
  const $body = $main.querySelector("tbody")!;
  const $note = $main.querySelector<HTMLElement>(".adm-note")!;
  const $more = $main.querySelector<HTMLElement>(".more")!;
  const $sel = $main.querySelector<HTMLSelectElement>(".au-select")!;

  const load = async (reset: boolean) => {
    const mine = ++loadSeq;
    $note.hidden = true;
    try {
      const page = await listAdminAudit({ action }, reset ? 0 : (next ?? 0), PAGE_SIZE);
      if (my !== seq || mine !== loadSeq) return;
      if (reset) {
        $body.innerHTML = "";
        shown = 0;
      }
      $body.insertAdjacentHTML("beforeend", page.items.map(rowHtml).join(""));
      shown += page.items.length;
      next = page.nextOffset;
      $main.querySelector(".adm-count")!.textContent = shown ? t("admin.shown", { shown, total: page.total }) : "";
      if (!shown) {
        $note.hidden = false;
        $note.className = "adm-note";
        $note.textContent = t("audit.empty");
      }
    } catch (e) {
      if (my !== seq || mine !== loadSeq) return;
      if (e instanceof ApiError && e.status === 403) return onDenied();
      showErrorIn($note, e, () => void load(reset));
    }
    $more.hidden = next === null;
  };

  $sel.addEventListener("change", () => {
    action = $sel.value;
    void load(true);
  });
  $more.querySelector("button")!.addEventListener("click", () => void load(false));
  void load(true);
}

function targetHtml(e: AuditEntry): string {
  const type = escapeHtml(label("audit.target", e.targetType));
  if (e.targetType === "sighting") {
    return `${type}: <a href="${plateHref(e.targetId)}" data-link>${escapeHtml(e.targetId.slice(0, 8))}</a>`;
  }
  if (e.targetType === "api_token") {
    return `${type}: <b>${escapeHtml(e.targetName || e.targetId.slice(0, 8))}</b>`;
  }
  return `${type}: <span class="au-id">${escapeHtml(e.targetId.slice(0, 8))}</span>`;
}

const valueText = (v: unknown): string =>
  typeof v === "boolean" ? t(v ? "audit.yes" : "audit.no") : v === null || v === undefined ? "—" : String(v);

// Что изменилось: у существующих объектов — только изменившиеся поля «было → стало»;
// у созданных (from = null) — поля нового объекта.
function detailsHtml(e: AuditEntry): string {
  const to = e.to ?? {};
  const from = e.from ?? {};
  const keys = Object.keys(to).filter((k) => !e.from || from[k] !== to[k]);
  if (!keys.length) return "—";
  return keys
    .map((k) => {
      const name = escapeHtml(label("audit.f", k));
      const now = `<b>${escapeHtml(valueText(to[k]))}</b>`;
      return e.from
        ? `<span class="au-chg">${name}: ${escapeHtml(valueText(from[k]))} → ${now}</span>`
        : `<span class="au-chg">${name}: ${now}</span>`;
    })
    .join("");
}

function rowHtml(e: AuditEntry): string {
  return `
    <tr>
      <td class="adm-time"><time datetime="${escapeHtml(e.createdAt)}">${escapeHtml(formatDate(e.createdAt))}</time></td>
      <td>${escapeHtml(label("audit.action", e.action))}<div class="au-meta">${escapeHtml(e.actor)}</div></td>
      <td>${targetHtml(e)}</td>
      <td>${detailsHtml(e)}${
        e.requestId ? `<div class="au-meta au-req" title="${t("audit.req")}">${escapeHtml(e.requestId)}</div>` : ""
      }</td>
    </tr>`;
}
