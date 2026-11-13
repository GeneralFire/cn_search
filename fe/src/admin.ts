// Админ-панель /admin: вкладки (реестр — admintabs.ts): «Жалобы» (здесь), «Уточнения» (adminclar.ts),
// заявки на камеры подключаются своей вкладкой. Вход — токен admin-token, проверяется rpc Me и включает
// админ-режим (adminmode.ts); без токена или с неверным (бэк ответил 403) — форма ввода.
// Жалобы: таблица с фильтром по статусу, «Загрузить ещё», кнопка «Решено»; ?sighting=<id> — только
// жалобы на эту фиксацию (ссылка из панели «Управление»); клик по номеру — страница фиксации.
import "./admin.css";
import { ApiError, getAdminToken, listComplaints, resolveComplaint, type Complaint, type ComplaintStatus } from "./api";
import { enterAdminMode, exitAdminMode, shieldIcon } from "./adminmode";
import { adminTabs, registerAdminTab } from "./admintabs";
import "./adminclar";
import { plateHref } from "./router";
import { escapeHtml, formatDate, photoAttrs, plateHtml } from "./ui";
import { t } from "./i18n/i18n";
import { mountSettings } from "./settings";
import { photoHtml } from "./nophoto";

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
        <h1 class="adm-bar__title">${shieldIcon}${t("adminMode.panelTitle")}</h1>
        <button class="btn btn--ghost adm-bar__logout" type="button" hidden>${t("admin.logout")}</button>
      </header>
      <nav class="adm-tabs" role="tablist" hidden></nav>
      <main class="adm"></main>`;
    mountSettings(root.querySelector<HTMLElement>(".adm-bar__settings")!);
    root.querySelector(".adm-bar__logout")!.addEventListener("click", () => {
      exitAdminMode();
      showLogin();
    });
  }
  $main = root.querySelector<HTMLElement>(".adm")!;
  document.title = t("admin.pageTitle");
  window.scrollTo({ top: 0 });
  if (getAdminToken()) showTab();
  else showLogin();
}

function setLoggedIn(on: boolean): void {
  $root.querySelector<HTMLElement>(".adm-bar__logout")!.hidden = !on;
  $root.querySelector<HTMLElement>(".adm-tabs")!.hidden = !on;
}

// ---------------------------------------------------------------- вкладки

const tabFromUrl = () => {
  const id = new URLSearchParams(location.search).get("tab");
  return adminTabs.find((x) => x.id === id) ?? adminTabs[0];
};

/** Рисует ряд вкладок и активную вкладку (из ?tab=). */
function showTab(): void {
  state.seq++;
  setLoggedIn(true);
  const active = tabFromUrl();
  const $tabs = $root.querySelector<HTMLElement>(".adm-tabs")!;
  $tabs.innerHTML = adminTabs
    .map((x) => `<button class="adm-tabs__tab" type="button" role="tab" data-tab="${x.id}" aria-selected="${x === active}">${t(x.label)}</button>`)
    .join("");
  $tabs.onclick = (e) => {
    const b = (e.target as Element).closest<HTMLButtonElement>("[data-tab]");
    if (!b) return;
    const url = new URL(location.href);
    url.search = b.dataset.tab === adminTabs[0].id ? "" : `?tab=${b.dataset.tab}`;
    history.replaceState(history.state, "", url);
    showTab();
  };
  void active.render($main, () => {
    exitAdminMode();
    showLogin(t("admin.denied"));
  });
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
  $form.addEventListener("submit", async (e) => {
    e.preventDefault();
    let ok = false;
    try {
      ok = await enterAdminMode($input.value.trim());
    } catch (err) {
      showLogin(err instanceof ApiError ? err.message : t("error.serverDown"));
      return;
    }
    if (ok) showTab();
    else showLogin(t("admin.denied"));
  });
}

/** Ошибка админского запроса: 403 — токен неверный, снова форма входа; иначе — сообщение. */
function isDenied(e: unknown): boolean {
  if (e instanceof ApiError && e.status === 403) {
    exitAdminMode();
    showLogin(t("admin.denied"));
    return true;
  }
  return false;
}

// ---------------------------------------------------------------- список

registerAdminTab({ id: "complaints", label: "adminMode.tabComplaints", order: 10, render: () => showList() });

/** Фильтр по фиксации из адреса (?sighting=<id>); "" — все жалобы. */
const sightingFilter = () => new URLSearchParams(location.search).get("sighting") ?? "";

async function showList(): Promise<void> {
  const sighting = sightingFilter();
  $main.innerHTML = `
    ${sighting ? `<p class="adm-filter-chip">${t("adminMode.bySighting")} <code>${escapeHtml(sighting.slice(0, 8))}</code>
      <button class="adm-link" type="button" data-reset-sighting>${t("adminMode.resetFilter")}</button></p>` : ""}
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
  $main.querySelector("[data-reset-sighting]")?.addEventListener("click", () => {
    history.replaceState(history.state, "", "/admin");
    void showList();
  });
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
    const page = await listComplaints(state.filter, reset ? 0 : (state.next ?? 0), PAGE_SIZE, sightingFilter());
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
  const thumb = photoHtml(src);
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
