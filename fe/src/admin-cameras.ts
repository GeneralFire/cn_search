// Вкладка «Заявки на камеры» админ-панели /admin: таблица заявок с контактом автора (ListCameraRequests
// с admin-token), фильтр по статусу, «Загрузить ещё» и смена статуса (select → SetCameraRequestStatus;
// на карте /cameras маркер получает цвет нового статуса).
import "./cameras.css";
import {
  ApiError,
  CAMERA_STATUSES,
  listCameraRequests,
  setCameraRequestStatus,
  type CameraRequest,
  type CameraStatus,
} from "./api";
import { registerAdminTab } from "./admintabs";
import { cameraStatusHtml, cameraStatusLabel } from "./cameras";
import { escapeHtml, formatDate } from "./ui";
import { t } from "./i18n/i18n";

const PAGE_SIZE = 50;

type Filter = CameraStatus | "";
let filter: Filter = "new";
let seq = 0;

function rowHtml(c: CameraRequest): string {
  const osm = `https://www.openstreetmap.org/?mlat=${c.lat}&mlon=${c.lon}#map=17/${c.lat}/${c.lon}`;
  return `
    <tr data-id="${escapeHtml(c.id)}">
      <td class="adm-cam-coords"><a href="${osm}" target="_blank" rel="noopener">${c.lat.toFixed(5)}, ${c.lon.toFixed(5)}</a></td>
      <td class="adm-addr">${escapeHtml(c.address) || "—"}</td>
      <td class="adm-cam-reason">${escapeHtml(c.reason)}</td>
      <td class="adm-cam-contact">${escapeHtml(c.contact) || "—"}</td>
      <td>${c.votes}</td>
      <td class="adm-time"><time datetime="${escapeHtml(c.createdAt)}">${escapeHtml(formatDate(c.createdAt))}</time></td>
      <td class="adm-cam-status">${cameraStatusHtml(c.status)}</td>
      <td><select class="adm-cam-select" data-cam-status="${escapeHtml(c.id)}" aria-label="${t("admin.colStatus")}">
        ${CAMERA_STATUSES.map((s) => `<option value="${s}"${s === c.status ? " selected" : ""}>${escapeHtml(cameraStatusLabel(s))}</option>`).join("")}
      </select></td>
    </tr>`;
}

async function render(main: HTMLElement, onDenied: () => void): Promise<void> {
  const filters: [Filter, string][] = [...CAMERA_STATUSES.map((s): [Filter, string] => [s, cameraStatusLabel(s)]), ["", t("admin.filterAll")]];
  main.innerHTML = `
    <div class="adm-head">
      <div class="adm-filter" role="radiogroup" aria-label="${t("admin.filter")}">
        ${filters.map(([f, label]) => `<label class="seg"><input type="radio" name="adm-cam-filter" value="${f}"${
          f === filter ? " checked" : ""} /><span>${escapeHtml(label)}</span></label>`).join("")}
      </div>
      <span class="adm-count" aria-live="polite"></span>
    </div>
    <div class="adm-table-wrap">
      <table class="adm-table">
        <thead><tr>
          <th>${t("cameras.admin.colPoint")}</th><th>${t("admin.colAddress")}</th><th>${t("cameras.admin.colReason")}</th>
          <th>${t("cameras.admin.colContact")}</th><th>${t("cameras.admin.colVotes")}</th><th>${t("admin.colTime")}</th>
          <th>${t("admin.colStatus")}</th><th></th>
        </tr></thead>
        <tbody></tbody>
      </table>
    </div>
    <p class="adm-note" hidden></p>
    <div class="more" hidden><button class="more__btn" type="button">${t("more.load")}</button></div>`;

  const $body = main.querySelector("tbody")!;
  const $note = main.querySelector<HTMLElement>(".adm-note")!;
  const $more = main.querySelector<HTMLElement>(".more")!;
  const $btn = $more.querySelector<HTMLButtonElement>(".more__btn")!;
  let shown = 0;
  let next: number | null = null;

  const denied = (e: unknown) => e instanceof ApiError && e.status === 403 && (onDenied(), true);

  async function load(reset: boolean): Promise<void> {
    const my = ++seq;
    if (reset) {
      shown = 0;
      next = null;
      $body.innerHTML = `<tr class="adm-sk"><td colspan="8"><div class="skeleton"></div></td></tr>`.repeat(3);
    }
    $btn.disabled = true;
    $note.hidden = true;
    try {
      const page = await listCameraRequests({ status: filter, offset: next ?? 0, limit: PAGE_SIZE, admin: true });
      if (my !== seq) return;
      if (reset) $body.innerHTML = "";
      $body.insertAdjacentHTML("beforeend", page.items.map(rowHtml).join(""));
      shown += page.items.length;
      next = page.nextOffset;
      main.querySelector(".adm-count")!.textContent = shown ? t("admin.shown", { shown, total: page.total }) : "";
      if (!shown) {
        $note.hidden = false;
        $note.className = "adm-note";
        $note.textContent = t("cameras.admin.empty");
      }
    } catch (e) {
      if (my !== seq || denied(e)) return;
      if (reset) $body.innerHTML = "";
      $note.hidden = false;
      $note.className = "adm-note is-error";
      $note.textContent = `${t("error.loadFailed")}: ${e instanceof ApiError ? e.message : t("error.serverDown")}`;
    }
    $more.hidden = next === null;
    $btn.disabled = false;
  }

  main.querySelector(".adm-filter")!.addEventListener("change", (e) => {
    filter = (e.target as HTMLInputElement).value as Filter;
    void load(true);
  });
  $btn.addEventListener("click", () => void load(false));
  $body.addEventListener("change", async (e) => {
    const $sel = (e.target as Element).closest<HTMLSelectElement>("[data-cam-status]");
    if (!$sel) return;
    $sel.disabled = true;
    $sel.classList.remove("is-error");
    try {
      const c = await setCameraRequestStatus($sel.dataset.camStatus!, $sel.value as CameraStatus);
      $sel.closest("tr")!.querySelector(".adm-cam-status")!.innerHTML = cameraStatusHtml(c.status);
    } catch (err) {
      if (denied(err)) return;
      $sel.classList.add("is-error");
      $sel.title = `${t("cameras.admin.statusFailed")}: ${err instanceof ApiError ? err.message : t("error.serverDown")}`;
    }
    $sel.disabled = false;
  });
  await load(true);
}

registerAdminTab({ id: "cameras", label: "cameras.admin.tab", order: 40, render });
