// Админ-панель /admin: жалобы на фото. Вход — токен admin-token (sessionStorage, см. api/index.ts);
// без токена или с неверным (бэк ответил 403) — форма ввода. Таблица жалоб с фильтром по статусу,
// «Загрузить ещё» и кнопкой «Решено»; клик по номеру — страница фиксации.
import "./admin.css";
import {
  ApiError,
  getAdminToken,
  listComplaints,
  resolveComplaint,
  setAdminToken,
  type Complaint,
  type ComplaintStatus,
} from "./api";
import { plateHref } from "./router";
import { escapeHtml, formatDate, icon, photoAttrs, plateHtml } from "./ui";
import { t } from "./i18n/i18n";
import { mountSettings } from "./settings";

const PAGE_SIZE = 30;

type Filter = ComplaintStatus | "";
const FILTERS: [Filter, Parameters<typeof t>[0]][] = [
  ["new", "admin.filterNew"],
  ["resolved", "admin.filterResolved"],
  ["", "admin.filterAll"],
];

const state = {
  filter: "new" as Filter,
  items: [] as Complaint[],
  total: 0,
  next: null as number | null,
  seq: 0, // номер загрузки: ответы устаревших запросов (сменили фильтр) игнорируются
};

let $root: HTMLElement;
let $main: HTMLElement;

/** Рисует админ-панель в root (шапка создаётся один раз). */
export function renderAdminPage(root: HTMLElement): void {
  $root = root;
  if (!root.querySelector(".adm-bar")) {
    root.innerHTML = `
      <header class="adm-bar">
        <div class="adm-bar__settings"></div>
        <h1 class="adm-bar__title">${t("admin.title")}</h1>
        <button class="btn btn--ghost adm-bar__logout" type="button" hidden>${t("admin.logout")}</button>
      </header>
      <main class="adm"></main>`;
    mountSettings(root.querySelector<HTMLElement>(".adm-bar__settings")!);
    root.querySelector(".adm-bar__logout")!.addEventListener("click", () => {
      setAdminToken("");
      showLogin();
    });
  }
  $main = root.querySelector<HTMLElement>(".adm")!;
  document.title = t("admin.pageTitle");
  window.scrollTo({ top: 0 });
  if (getAdminToken()) void showList();
  else showLogin();
}

function setLoggedIn(on: boolean): void {
  $root.querySelector<HTMLElement>(".adm-bar__logout")!.hidden = !on;
}

// ---------------------------------------------------------------- вход

function showLogin(error = ""): void {
  state.seq++;
  setLoggedIn(false);
  $main.innerHTML = `
    <form class="adm-login" autocomplete="off">
      <label class="adm-login__field">
        <span class="adm-login__label">${t("admin.tokenLabel")}</span>
        <input class="adm-login__input" type="password" name="token" required autocomplete="off" spellcheck="false" />
      </label>
      <p class="adm-login__hint">${t("admin.tokenHint")}</p>
      <p class="adm-login__error" role="alert"${error ? "" : " hidden"}>${escapeHtml(error)}</p>
      <button class="btn btn--primary" type="submit">${t("admin.login")}</button>
    </form>`;
  const $form = $main.querySelector<HTMLFormElement>(".adm-login")!;
  const $input = $form.querySelector<HTMLInputElement>(".adm-login__input")!;
  $input.focus();
  $form.addEventListener("submit", (e) => {
    e.preventDefault();
    setAdminToken($input.value.trim());
    void showList();
  });
}

/** Ошибка админского запроса: 403 — токен неверный, снова форма входа; иначе — сообщение. */
function isDenied(e: unknown): boolean {
  if (e instanceof ApiError && e.status === 403) {
    setAdminToken("");
    showLogin(t("admin.denied"));
    return true;
  }
  return false;
}

// ---------------------------------------------------------------- список

async function showList(): Promise<void> {
  setLoggedIn(true);
  $main.innerHTML = `
    <div class="adm-head">
      <div class="adm-filter" role="radiogroup" aria-label="${t("admin.filter")}">
        ${FILTERS.map(
          ([f, key]) => `<label class="seg"><input type="radio" name="adm-filter" value="${f}"${
            f === state.filter ? " checked" : ""
          } /><span>${t(key)}</span></label>`
        ).join("")}
      </div>
      <span class="adm-count" aria-live="polite"></span>
    </div>
    <div class="adm-table-wrap">
      <table class="adm-table">
        <thead><tr>
          <th>${t("admin.colPhoto")}</th><th>${t("admin.colPlate")}</th><th>${t("admin.colAddress")}</th>
          <th>${t("admin.colReason")}</th><th>${t("admin.colComment")}</th><th>${t("admin.colTime")}</th>
          <th>${t("admin.colStatus")}</th><th></th>
        </tr></thead>
        <tbody></tbody>
      </table>
    </div>
    <p class="adm-note" hidden></p>
    <div class="more" hidden><button class="more__btn" type="button">${t("more.load")}</button></div>`;

  $main.querySelector(".adm-filter")!.addEventListener("change", (e) => {
    state.filter = (e.target as HTMLInputElement).value as Filter;
    void load(true);
  });
  $main.querySelector(".more__btn")!.addEventListener("click", () => void load(false));
  $main.querySelector("tbody")!.addEventListener("click", (e) => {
    const b = (e.target as Element).closest<HTMLButtonElement>("[data-resolve]");
    if (b) void resolve(b);
  });
  await load(true);
}

async function load(reset: boolean): Promise<void> {
  const my = ++state.seq;
  const $more = $main.querySelector<HTMLElement>(".more")!;
  const $btn = $more.querySelector<HTMLButtonElement>(".more__btn")!;
  const $note = $main.querySelector<HTMLElement>(".adm-note")!;
  if (reset) {
    state.items = [];
    state.next = null;
    $main.querySelector("tbody")!.innerHTML = skeletonRows(3);
  }
  $btn.disabled = true;
  $btn.textContent = t("more.loading");
  $note.hidden = true;
  try {
    const page = await listComplaints(state.filter, reset ? 0 : (state.next ?? 0), PAGE_SIZE);
    if (my !== state.seq) return;
    state.items.push(...page.items);
    state.total = page.total;
    state.next = page.nextOffset;
    const $body = $main.querySelector("tbody")!;
    if (reset) $body.innerHTML = "";
    $body.insertAdjacentHTML("beforeend", page.items.map(rowHtml).join(""));
    if (!state.items.length) {
      $note.hidden = false;
      $note.className = "adm-note";
      $note.textContent = t("admin.empty");
    }
  } catch (e) {
    if (my !== state.seq || isDenied(e)) return;
    if (reset) $main.querySelector("tbody")!.innerHTML = "";
    $note.hidden = false;
    $note.className = "adm-note is-error";
    $note.textContent = `${t("error.loadFailed")}: ${e instanceof ApiError ? e.message : t("error.serverDown")}`;
  }
  $main.querySelector(".adm-count")!.textContent = state.items.length
    ? t("admin.shown", { shown: state.items.length, total: state.total })
    : "";
  $more.hidden = state.next === null;
  $btn.disabled = false;
  $btn.textContent = t("more.load");
}

function statusHtml(s: ComplaintStatus): string {
  return `<span class="adm-status adm-status--${s}">${t(s === "new" ? "admin.statusNew" : "admin.statusResolved")}</span>`;
}

function rowHtml(c: Complaint): string {
  const s = c.sighting;
  const href = s ? plateHref(s.id) : "";
  const { src } = s ? photoAttrs(s) : { src: "" };
  const thumb = src ? `<img src="${escapeHtml(src)}" alt="" loading="lazy" />` : icon.car;
  return `
    <tr data-id="${escapeHtml(c.id)}">
      <td>${href ? `<a class="adm-thumb" href="${href}" data-link tabindex="-1">${thumb}</a>` : `<span class="adm-thumb">${thumb}</span>`}</td>
      <td>${s ? `<a class="adm-plate" href="${href}" data-link>${plateHtml(s)}</a>` : ""}</td>
      <td class="adm-addr">${escapeHtml(s?.address ?? "")}</td>
      <td><span class="adm-reason adm-reason--${c.reason}">${t(`complaint.reason.${c.reason}`)}</span></td>
      <td class="adm-comment">${escapeHtml(c.comment) || "—"}</td>
      <td class="adm-time"><time datetime="${escapeHtml(c.createdAt)}">${escapeHtml(formatDate(c.createdAt))}</time></td>
      <td class="adm-status-cell">${statusHtml(c.status)}</td>
      <td>${c.status === "new" ? `<button class="btn btn--ghost" type="button" data-resolve="${escapeHtml(c.id)}">${t("admin.resolve")}</button>` : ""}</td>
    </tr>`;
}

function skeletonRows(n: number): string {
  return `<tr class="adm-sk"><td colspan="8"><div class="skeleton"></div></td></tr>`.repeat(n);
}

async function resolve(b: HTMLButtonElement): Promise<void> {
  b.disabled = true;
  try {
    await resolveComplaint(b.dataset.resolve!);
    const c = state.items.find((x) => x.id === b.dataset.resolve);
    if (c) c.status = "resolved";
    const $row = b.closest("tr")!;
    $row.querySelector(".adm-status-cell")!.innerHTML = statusHtml("resolved");
    $row.classList.add("is-resolved");
    b.remove();
  } catch (e) {
    if (isDenied(e)) return;
    b.disabled = false;
    b.title = `${t("admin.resolveFailed")}: ${e instanceof ApiError ? e.message : t("error.serverDown")}`;
    b.classList.add("is-error");
  }
}
