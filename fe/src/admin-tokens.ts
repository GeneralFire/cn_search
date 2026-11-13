// Админка /admin: вкладки «Токены» (API-токены cst_…: создание, остаток, отзыв, журнал токена)
// и «Просмотры» (общий журнал открытий страниц номеров с фильтром по токену). Подключаются в реестр
// admintabs.ts; оболочку вкладок и вход рисует admin.ts.
import "./admin-tokens.css";
import {
  ApiError,
  createApiToken,
  listApiTokens,
  listPlateViews,
  revokeApiToken,
  type ApiToken,
  type PlateView,
} from "./api";
import { plateHref } from "./router";
import { escapeHtml, formatDate, plateHtml } from "./ui";
import { t } from "./i18n/i18n";
import { registerAdminTab } from "./admintabs";

type OnDenied = (e: unknown) => boolean;

const PAGE_SIZE = 30;
let seq = 0; // номер отрисовки: ответы устаревших запросов игнорируются

const errText = (e: unknown) => (e instanceof ApiError ? e.message : t("error.serverDown"));

// 403 от бэка — токен админки неверный: оболочка (admin.ts) возвращает к форме входа.
const deniedHandler =
  (onDenied: () => void): OnDenied =>
  (e) => {
    if (e instanceof ApiError && e.status === 403) {
      onDenied();
      return true;
    }
    return false;
  };

registerAdminTab({
  id: "tokens",
  label: "tokens.tabTokens",
  order: 40,
  render: ($main, onDenied) => {
    seq++;
    renderTokens($main, deniedHandler(onDenied));
  },
});
registerAdminTab({
  id: "views",
  label: "tokens.tabViews",
  order: 50,
  render: ($main, onDenied) => {
    seq++;
    renderViews($main, deniedHandler(onDenied), "");
  },
});

// ---------------------------------------------------------------- токены

function renderTokens($main: HTMLElement, onDenied: OnDenied): void {
  $main.innerHTML = `
    <form class="tk-create" autocomplete="off">
      <h2 class="tk-create__title">${t("tokens.createTitle")}</h2>
      <label class="tk-field"><span>${t("tokens.name")}</span>
        <input class="tk-input" name="name" required maxlength="100" placeholder="${t("tokens.namePlaceholder")}" /></label>
      <label class="tk-field tk-field--num"><span>${t("tokens.requests")}</span>
        <input class="tk-input" name="requests" type="number" min="1" max="1000000" value="100" required /></label>
      <button class="btn btn--primary" type="submit">${t("tokens.create")}</button>
      <p class="tk-error" role="alert" hidden></p>
    </form>
    <div class="tk-new" hidden></div>
    <div class="adm-head"><span class="adm-count" aria-live="polite"></span></div>
    <div class="adm-table-wrap">
      <table class="adm-table tk-table">
        <thead><tr>
          <th>${t("tokens.colName")}</th><th>${t("tokens.colLeft")}</th><th>${t("tokens.colViews")}</th>
          <th>${t("tokens.colCreated")}</th><th>${t("tokens.colLastUsed")}</th><th>${t("tokens.colStatus")}</th><th></th>
        </tr></thead>
        <tbody></tbody>
      </table>
    </div>
    <p class="adm-note" hidden></p>
    <div class="more" hidden><button class="more__btn" type="button">${t("more.load")}</button></div>
    <section class="tk-journal" hidden></section>`;

  const my = seq;
  const items: ApiToken[] = [];
  let next: number | null = null;
  const $body = $main.querySelector("tbody")!;
  const $note = $main.querySelector<HTMLElement>(".adm-note")!;
  const $more = $main.querySelector<HTMLElement>(".more")!;
  const $journal = $main.querySelector<HTMLElement>(".tk-journal")!;

  const load = async (reset: boolean) => {
    $note.hidden = true;
    try {
      const page = await listApiTokens(reset ? 0 : (next ?? 0), PAGE_SIZE);
      if (my !== seq) return;
      if (reset) items.length = 0;
      items.push(...page.items);
      next = page.nextOffset;
      $body.innerHTML = items.map(tokenRowHtml).join("");
      $main.querySelector(".adm-count")!.textContent = items.length
        ? t("admin.shown", { shown: items.length, total: page.total })
        : "";
      if (!items.length) {
        $note.hidden = false;
        $note.className = "adm-note";
        $note.textContent = t("tokens.empty");
      }
    } catch (e) {
      if (my !== seq || onDenied(e)) return;
      $note.hidden = false;
      $note.className = "adm-note is-error";
      $note.textContent = `${t("error.loadFailed")}: ${errText(e)}`;
    }
    $more.hidden = next === null;
  };

  const $form = $main.querySelector<HTMLFormElement>(".tk-create")!;
  $form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData($form);
    const $err = $form.querySelector<HTMLElement>(".tk-error")!;
    const $btn = $form.querySelector<HTMLButtonElement>("button[type=submit]")!;
    $btn.disabled = true;
    $err.hidden = true;
    try {
      const { token, info } = await createApiToken(String(fd.get("name")).trim(), Number(fd.get("requests")));
      showNewToken($main.querySelector<HTMLElement>(".tk-new")!, token, info);
      $form.reset();
      await load(true);
    } catch (e) {
      if (onDenied(e)) return;
      $err.hidden = false;
      $err.textContent = `${t("tokens.createFailed")}: ${errText(e)}`;
    }
    $btn.disabled = false;
  });

  $more.querySelector("button")!.addEventListener("click", () => void load(false));
  $body.addEventListener("click", async (e) => {
    const target = e.target as Element;
    const revoke = target.closest<HTMLButtonElement>("[data-revoke]");
    if (revoke) {
      e.stopPropagation();
      if (!confirm(t("tokens.revokeConfirm"))) return;
      revoke.disabled = true;
      try {
        const upd = await revokeApiToken(revoke.dataset.revoke!);
        const i = items.findIndex((x) => x.id === upd.id);
        if (i >= 0) items[i] = { ...items[i], revoked: true };
        $body.innerHTML = items.map(tokenRowHtml).join("");
      } catch (err) {
        if (onDenied(err)) return;
        revoke.disabled = false;
        revoke.title = errText(err);
      }
      return;
    }
    const row = target.closest<HTMLElement>("tr[data-token]");
    if (!row) return;
    $body.querySelectorAll("tr.is-selected").forEach((r) => r.classList.remove("is-selected"));
    row.classList.add("is-selected");
    const tok = items.find((x) => x.id === row.dataset.token);
    $journal.hidden = false;
    renderViews($journal, onDenied, row.dataset.token!, tok?.name ?? "");
    $journal.scrollIntoView({ behavior: "smooth", block: "start" });
  });

  void load(true);
}

function showNewToken($box: HTMLElement, token: string, info: ApiToken): void {
  $box.hidden = false;
  $box.innerHTML = `
    <div class="tk-new__head">${t("tokens.created", { name: escapeHtml(info.name), n: info.requestsTotal })}</div>
    <div class="tk-new__row">
      <code class="tk-new__token">${escapeHtml(token)}</code>
      <button class="btn btn--primary tk-new__copy" type="button">${t("tokens.copy")}</button>
    </div>
    <p class="tk-new__hint">${t("tokens.onceHint")}</p>`;
  const $copy = $box.querySelector<HTMLButtonElement>(".tk-new__copy")!;
  $copy.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(token);
    } catch {
      // Нет доступа к буферу (не https) — выделяем текст, чтобы скопировать вручную.
      getSelection()?.selectAllChildren($box.querySelector(".tk-new__token")!);
    }
    $copy.textContent = t("tokens.copied");
  });
}

function tokenRowHtml(tk: ApiToken): string {
  const pct = tk.requestsTotal ? Math.round((tk.requestsLeft / tk.requestsTotal) * 100) : 0;
  const level = tk.requestsLeft === 0 ? "empty" : pct <= 20 ? "low" : "ok";
  return `
    <tr data-token="${escapeHtml(tk.id)}" class="tk-row${tk.revoked ? " is-revoked" : ""}" title="${t("tokens.openJournal")}">
      <td class="tk-name">${escapeHtml(tk.name)}</td>
      <td class="tk-left">
        <div class="tk-bar tk-bar--${level}" role="progressbar" aria-valuemin="0" aria-valuemax="${tk.requestsTotal}"
          aria-valuenow="${tk.requestsLeft}"><span style="width:${pct}%"></span></div>
        <span class="tk-left__num">${tk.requestsLeft} / ${tk.requestsTotal}</span>
      </td>
      <td>${tk.viewsCount}</td>
      <td class="adm-time">${escapeHtml(formatDate(tk.createdAt))}</td>
      <td class="adm-time">${tk.lastUsedAt ? escapeHtml(formatDate(tk.lastUsedAt)) : "—"}</td>
      <td><span class="tk-status tk-status--${tk.revoked ? "revoked" : level}">${t(
        tk.revoked ? "tokens.statusRevoked" : tk.requestsLeft === 0 ? "tokens.statusExhausted" : "tokens.statusActive"
      )}</span></td>
      <td>${tk.revoked ? "" : `<button class="btn btn--ghost" type="button" data-revoke="${escapeHtml(tk.id)}">${t("tokens.revoke")}</button>`}</td>
    </tr>`;
}

// ---------------------------------------------------------------- журнал просмотров

/** Журнал в $box: tokenId "" — общий (вкладка «Просмотры», с фильтром по токену), иначе — журнал токена. */
function renderViews($box: HTMLElement, onDenied: OnDenied, tokenId: string, tokenName = ""): void {
  const general = !tokenId;
  $box.innerHTML = `
    <div class="adm-head">
      ${
        general
          ? `<label class="tk-filter"><span>${t("tokens.filterToken")}</span>
              <select class="tk-input tk-filter__select"><option value="">${t("tokens.filterAll")}</option></select></label>`
          : `<h2 class="tk-journal__title">${t("tokens.journalOf", { name: escapeHtml(tokenName) })}</h2>`
      }
      <span class="adm-count" aria-live="polite"></span>
    </div>
    <div class="adm-table-wrap">
      <table class="adm-table tk-views">
        <thead><tr>
          <th>${t("admin.colTime")}</th><th>${t("admin.colPlate")}</th><th>${t("tokens.colToken")}</th>
          <th>${t("tokens.colConsumed")}</th><th>${t("tokens.colUserAgent")}</th>
        </tr></thead>
        <tbody></tbody>
      </table>
    </div>
    <p class="adm-note" hidden></p>
    <div class="more" hidden><button class="more__btn" type="button">${t("more.load")}</button></div>`;

  const my = seq;
  let filter = tokenId;
  let next: number | null = null;
  let shown = 0;
  let loadSeq = 0;
  const $body = $box.querySelector("tbody")!;
  const $note = $box.querySelector<HTMLElement>(".adm-note")!;
  const $more = $box.querySelector<HTMLElement>(".more")!;

  const load = async (reset: boolean) => {
    const mine = ++loadSeq;
    $note.hidden = true;
    try {
      const page = await listPlateViews({ tokenId: filter }, reset ? 0 : (next ?? 0), PAGE_SIZE);
      if (my !== seq || mine !== loadSeq) return;
      if (reset) {
        $body.innerHTML = "";
        shown = 0;
      }
      $body.insertAdjacentHTML("beforeend", page.items.map(viewRowHtml).join(""));
      shown += page.items.length;
      next = page.nextOffset;
      $box.querySelector(".adm-count")!.textContent = shown ? t("admin.shown", { shown, total: page.total }) : "";
      if (!shown) {
        $note.hidden = false;
        $note.className = "adm-note";
        $note.textContent = t("tokens.viewsEmpty");
      }
    } catch (e) {
      if (my !== seq || onDenied(e)) return;
      $note.hidden = false;
      $note.className = "adm-note is-error";
      $note.textContent = `${t("error.loadFailed")}: ${errText(e)}`;
    }
    $more.hidden = next === null;
  };
  $more.querySelector("button")!.addEventListener("click", () => void load(false));

  if (general) {
    const $sel = $box.querySelector<HTMLSelectElement>(".tk-filter__select")!;
    $sel.addEventListener("change", () => {
      filter = $sel.value;
      void load(true);
    });
    // Список токенов для фильтра (первые 100).
    listApiTokens(0, 100)
      .then((p) => {
        if (my !== seq) return;
        $sel.insertAdjacentHTML(
          "beforeend",
          p.items.map((tk) => `<option value="${escapeHtml(tk.id)}">${escapeHtml(tk.name)}</option>`).join("")
        );
      })
      .catch(() => {});
  }
  void load(true);
}

function viewRowHtml(v: PlateView): string {
  const s = v.sighting;
  const who = v.isAdmin
    ? `<span class="tk-who tk-who--admin">${t("tokens.admin")}</span>`
    : v.tokenId
      ? `<span class="tk-who">${escapeHtml(v.tokenName)}</span>`
      : `<span class="tk-who tk-who--anon">${t("tokens.noToken")}</span>`;
  return `
    <tr>
      <td class="adm-time"><time datetime="${escapeHtml(v.createdAt)}">${escapeHtml(formatDate(v.createdAt))}</time></td>
      <td>${s ? `<a class="adm-plate" href="${plateHref(s.id)}" data-link>${plateHtml(s)}</a>` : ""}</td>
      <td>${who}</td>
      <td><span class="tk-consumed tk-consumed--${v.consumed}">${t(v.consumed ? "tokens.consumedYes" : "tokens.consumedNo")}</span></td>
      <td class="tk-ua" title="${escapeHtml(v.userAgent)}">${escapeHtml(v.userAgent) || "—"}</td>
    </tr>`;
}
