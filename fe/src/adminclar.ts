// Вкладка «Уточнения» админ-панели /admin: уточнения номеров с фильтром по статусу (по умолчанию —
// непринятые), «Загрузить ещё», «Принять / Отклонить» (кнопки — точка расширения adminpanels.ts).
import { ApiError, listClarifications, type Clarification, type ClarificationStatus } from "./api";
import { showErrorIn } from "./states";
import { registerAdminTab } from "./admintabs";
import { adminReady, isAdminMode } from "./adminmode";
import { bindClarificationControls, clarificationControlsHtml, clarificationStatusHtml } from "./adminpanels";
import { plateHref } from "./router";
import { escapeHtml, formatDate, plateHtml } from "./ui";
import { t } from "./i18n/i18n";

const PAGE_SIZE = 30;

type Filter = ClarificationStatus | "";
const FILTERS: [Filter, Parameters<typeof t>[0]][] = [
  ["not_accepted", "adminMode.notAccepted"],
  ["accepted", "adminMode.accepted"],
  ["", "admin.filterAll"],
];

let filter: Filter = "not_accepted";
let seq = 0;

function rowHtml(c: Clarification): string {
  const s = c.sighting;
  return `
    <tr>
      <td>${s ? `<a class="adm-plate" href="${plateHref(s.id)}" data-link>${plateHtml(s)}</a>` : ""}</td>
      <td>${plateHtml({ plate: c.plate, plateType: c.plateType, plateRegion: c.plateRegion })}</td>
      <td class="adm-addr">${escapeHtml(s?.address ?? "")}</td>
      <td class="adm-time"><time datetime="${escapeHtml(c.createdAt)}">${escapeHtml(formatDate(c.createdAt))}</time></td>
      <td data-adm-clar-status="${escapeHtml(c.id)}">${clarificationStatusHtml(c.status)}</td>
      <td>${clarificationControlsHtml(c)}</td>
    </tr>`;
}

async function render(main: HTMLElement, onDenied: () => void): Promise<void> {
  // Кнопки «Принять / Отклонить» рисуются только во включённом режиме: ждём подтверждения токена.
  await adminReady;
  if (!isAdminMode()) return onDenied();
  main.innerHTML = `
    <div class="adm-head">
      <div class="adm-filter" role="radiogroup" aria-label="${t("admin.filter")}">
        ${FILTERS.map(([f, key]) => `<label class="seg"><input type="radio" name="adm-clar-filter" value="${f}"${
          f === filter ? " checked" : ""} /><span>${t(key)}</span></label>`).join("")}
      </div>
      <span class="adm-count" aria-live="polite"></span>
    </div>
    <div class="adm-table-wrap">
      <table class="adm-table">
        <thead><tr>
          <th>${t("adminMode.colCurrent")}</th><th>${t("adminMode.colProposed")}</th><th>${t("admin.colAddress")}</th>
          <th>${t("admin.colTime")}</th><th>${t("admin.colStatus")}</th><th></th>
        </tr></thead>
        <tbody></tbody>
      </table>
    </div>
    <p class="adm-note" hidden></p>
    <div class="more" hidden><button class="more__btn" type="button">${t("more.load")}</button></div>`;

  const $body = main.querySelector("tbody")!;
  const $note = main.querySelector<HTMLElement>(".adm-note")!;
  const $more = main.querySelector<HTMLElement>(".more")!;
  let next: number | null = null;
  let shown = 0;

  const load = async (reset: boolean) => {
    const my = ++seq;
    if (reset) {
      $body.innerHTML = `<tr class="adm-sk"><td colspan="6"><div class="skeleton"></div></td></tr>`;
      next = null;
      shown = 0;
    }
    $note.hidden = true;
    try {
      const page = await listClarifications(filter, reset ? 0 : (next ?? 0), PAGE_SIZE);
      if (my !== seq) return;
      if (reset) $body.innerHTML = "";
      $body.insertAdjacentHTML("beforeend", page.items.map(rowHtml).join(""));
      shown += page.items.length;
      next = page.nextOffset;
      main.querySelector(".adm-count")!.textContent = shown ? t("admin.shown", { shown, total: page.total }) : "";
      if (!shown) {
        $note.hidden = false;
        $note.className = "adm-note";
        $note.textContent = t("adminMode.noClarifications");
      }
    } catch (e) {
      if (my !== seq) return;
      if (e instanceof ApiError && e.status === 403) return onDenied();
      if (reset) $body.innerHTML = "";
      showErrorIn($note, e, () => void load(reset));
    }
    $more.hidden = next === null;
  };

  main.querySelector(".adm-filter")!.addEventListener("change", (e) => {
    filter = (e.target as HTMLInputElement).value as Filter;
    void load(true);
  });
  $more.querySelector("button")!.addEventListener("click", () => void load(false));
  bindClarificationControls(main.querySelector<HTMLElement>(".adm-table-wrap")!);
  await load(true);
}

registerAdminTab({ id: "clarifications", label: "adminMode.tabClarifications", order: 30, render });
