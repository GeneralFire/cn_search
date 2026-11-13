// Аккаунт: вход по токену (cookie cs_token), регистрация, тарифы и демо-оплата.
// Логина и пароля нет: при регистрации бэк выдаёт токен "cs_…" один раз, пользователь вводит его в поле
// «Токен» (панель настроек на логотипе или /account) — после проверки через rpc Me токен кладётся в cookie.
// Страницы: /signup, /pricing, /pay/mock/<id> (демо-касса, деньги не списываются), /account.
// admin-token (sessionStorage, админ-панель) здесь не трогаем.
import "./auth.css";
import {
  ApiError,
  confirmMockPayment,
  createPayment,
  me,
  myPayments,
  plans,
  register,
  type Me,
  type Payment,
  type PlanInfo,
} from "./api";
import { intlLocale, t, type Key } from "./i18n/i18n";
import { navigate, type Route } from "./router";
import { mountSettings } from "./settings";
import { escapeHtml, formatDate } from "./ui";

// ---------------------------------------------------------------- cookie

const COOKIE = "cs_token";
const YEAR_S = 365 * 24 * 3600;

const cookieAttrs = (maxAge: number) =>
  `; path=/; max-age=${maxAge}; SameSite=Lax${location.protocol === "https:" ? "; Secure" : ""}`;

export function getToken(): string {
  const m = document.cookie.match(/(?:^|;\s*)cs_token=([^;]*)/);
  return m ? decodeURIComponent(m[1]) : "";
}

function setToken(token: string): void {
  document.cookie = `${COOKIE}=${encodeURIComponent(token)}${cookieAttrs(YEAR_S)}`;
}

/** Выход: удаляет cookie и перезагружает страницу (обновятся панель, значок PRO, страница аккаунта). */
export function logout(): void {
  document.cookie = `${COOKIE}=${cookieAttrs(0)}`;
  location.reload();
}

// ---------------------------------------------------------------- кто я (один запрос на загрузку страницы)

let mePromise: Promise<Me> | null = null;
const anon: Me = { isAdmin: false, user: null };

function loadMe(): Promise<Me> {
  if (!getToken()) return Promise.resolve(anon);
  mePromise ??= me().catch(() => anon);
  return mePromise;
}

/** Проверяет токен через Me; верный — ставит cookie. Иначе — текст ошибки (cookie не ставится). */
async function login(token: string): Promise<string> {
  token = token.trim();
  if (!token) return t("auth.tokenEmpty");
  try {
    const r = await me(token);
    if (!r.user) return t("auth.tokenWrong");
  } catch (e) {
    return e instanceof ApiError ? e.message : t("error.serverDown");
  }
  setToken(token);
  return "";
}

// ---------------------------------------------------------------- форматирование

const money = (cents: number, currency: string) =>
  new Intl.NumberFormat(intlLocale(), { style: "currency", currency, maximumFractionDigits: cents % 100 ? 2 : 0 }).format(
    cents / 100
  );

const proBadge = () => `<span class="pro-badge" title="${t("auth.proTitle")}">PRO</span>`;

// ---------------------------------------------------------------- форма входа (панель настроек и /account)

function loginFormHtml(): string {
  return `
    <form class="au-login" autocomplete="off">
      <label class="au-login__field">
        <span class="au-login__label">${t("auth.token")}</span>
        <input class="au-input" name="token" type="password" required spellcheck="false" autocomplete="off" placeholder="cs_…" />
      </label>
      <p class="au-error" role="alert" hidden></p>
      <div class="au-login__row">
        <button class="btn btn--primary" type="submit">${t("auth.login")}</button>
        <a class="au-link" href="/signup" data-link>${t("auth.signupLink")}</a>
      </div>
    </form>`;
}

function bindLoginForm(root: HTMLElement): void {
  const form = root.querySelector<HTMLFormElement>(".au-login")!;
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const btn = form.querySelector<HTMLButtonElement>("button")!;
    const err = form.querySelector<HTMLElement>(".au-error")!;
    btn.disabled = true;
    const msg = await login(form.querySelector<HTMLInputElement>("input")!.value);
    btn.disabled = false;
    err.hidden = !msg;
    err.textContent = msg;
    if (!msg) location.reload();
  });
}

// ---------------------------------------------------------------- секция «Аккаунт» в панели настроек

/** Секция «Аккаунт» панели на логотипе: форма входа или e-mail/ID, план и ссылки. close — закрыть панель. */
export function mountAccountSection(root: HTMLElement, close: () => void): void {
  root.innerHTML = `<div class="st__label">${t("auth.section")}</div><div class="st-acc__body"></div>`;
  const body = root.querySelector<HTMLElement>(".st-acc__body")!;
  void loadMe().then(({ user }) => {
    if (!user) {
      body.innerHTML = loginFormHtml();
      bindLoginForm(body);
    } else {
      body.innerHTML = `
        <div class="st-acc__who">
          <span class="st-acc__name">${escapeHtml(user.email || `ID ${user.id.slice(0, 8)}`)}</span>
          ${user.plan === "pro" ? proBadge() : `<span class="free-badge">Free</span>`}
        </div>
        <div class="st-acc__links">
          <a class="au-link" href="/account" data-link>${t("auth.account")}</a>
          <a class="au-link" href="/pricing" data-link>${t("auth.pricing")}</a>
          <button class="au-link au-link--btn" type="button" data-logout>${t("auth.logout")}</button>
        </div>`;
    }
    body.querySelectorAll("a[data-link]").forEach((a) => a.addEventListener("click", close));
    body.querySelector("[data-logout]")?.addEventListener("click", logout);
  });
}

/** Значок PRO рядом с логотипом, когда план — Pro. */
export function mountProBadge(box: HTMLElement): void {
  void loadMe().then(({ user }) => {
    if (user?.plan === "pro") box.querySelector(".st__btn")?.insertAdjacentHTML("afterend", proBadge());
  });
}

// ---------------------------------------------------------------- страницы

export const isAuthRoute = (r: Route): r is Extract<Route, { name: "signup" | "pricing" | "account" | "pay" }> =>
  r.name === "signup" || r.name === "pricing" || r.name === "account" || r.name === "pay";

const TITLES: Record<"signup" | "pricing" | "account" | "pay", Key> = {
  signup: "auth.signupTitle",
  pricing: "auth.pricingTitle",
  account: "auth.accountTitle",
  pay: "auth.payTitle",
};

let seq = 0; // номер отрисовки: ответы для уже ушедшей страницы игнорируются

export function renderAuthPage(root: HTMLElement, route: Extract<Route, { name: "signup" | "pricing" | "account" | "pay" }>): void {
  const my = ++seq;
  root.innerHTML = `
    <header class="au-bar">
      <div class="au-bar__settings"></div>
      <h1 class="au-bar__title">${t(TITLES[route.name])}</h1>
    </header>
    <main class="au"><p class="au-muted">${t("auth.loading")}</p></main>`;
  mountSettings(root.querySelector<HTMLElement>(".au-bar__settings")!);
  document.title = `${t(TITLES[route.name])} — ${t("app.title")}`;
  window.scrollTo({ top: 0 });
  const main = root.querySelector<HTMLElement>(".au")!;
  const alive = () => my === seq;
  const fail = (e: unknown) => {
    if (alive()) main.innerHTML = `<p class="au-error">${escapeHtml(e instanceof ApiError ? e.message : t("error.serverDown"))}</p>`;
  };
  const pages = { signup: renderSignup, pricing: renderPricing, account: renderAccount, pay: renderPay };
  pages[route.name](main, alive, "id" in route ? route.id : "").catch(fail);
}

/** Не вошёл — карточка с формой входа вместо страницы. */
function needLogin(main: HTMLElement): void {
  main.innerHTML = `<section class="au-card au-card--narrow"><p>${t("auth.needLogin")}</p>${loginFormHtml()}</section>`;
  bindLoginForm(main);
}

// /signup: e-mail (необязательно) → токен один раз, «Скопировать», «Войти с этим токеном».
async function renderSignup(main: HTMLElement): Promise<void> {
  main.innerHTML = `
    <form class="au-card au-card--narrow au-signup">
      <p class="au-muted">${t("auth.signupHint")}</p>
      <label class="au-login__field">
        <span class="au-login__label">${t("auth.email")}</span>
        <input class="au-input" name="email" type="email" maxlength="254" autocomplete="email" placeholder="name@example.com" />
      </label>
      <p class="au-error" role="alert" hidden></p>
      <button class="btn btn--primary" type="submit">${t("auth.signupBtn")}</button>
    </form>`;
  const form = main.querySelector<HTMLFormElement>("form")!;
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const btn = form.querySelector<HTMLButtonElement>("button")!;
    const err = form.querySelector<HTMLElement>(".au-error")!;
    btn.disabled = true;
    try {
      const { token } = await register(form.querySelector<HTMLInputElement>("input")!.value.trim());
      showToken(main, token);
    } catch (ex) {
      err.hidden = false;
      err.textContent = ex instanceof ApiError ? ex.message : t("error.serverDown");
      btn.disabled = false;
    }
  });
}

function showToken(main: HTMLElement, token: string): void {
  main.innerHTML = `
    <section class="au-card au-card--narrow">
      <h2 class="au-h2">${t("auth.tokenReady")}</h2>
      <div class="au-token">
        <input class="au-input au-token__value" readonly value="${escapeHtml(token)}" spellcheck="false" aria-label="${t("auth.token")}" />
        <button class="btn btn--ghost" type="button" data-copy>${t("auth.copy")}</button>
      </div>
      <p class="au-warn" role="note">${t("auth.tokenWarn")}</p>
      <button class="btn btn--primary" type="button" data-use>${t("auth.useToken")}</button>
    </section>`;
  const input = main.querySelector<HTMLInputElement>(".au-token__value")!;
  const copy = main.querySelector<HTMLButtonElement>("[data-copy]")!;
  copy.addEventListener("click", async () => {
    input.select();
    try {
      await navigator.clipboard.writeText(token);
    } catch {
      document.execCommand("copy"); // http в локальной сети — Clipboard API недоступен
    }
    copy.textContent = t("auth.copied");
  });
  main.querySelector("[data-use]")!.addEventListener("click", () => {
    setToken(token);
    location.assign("/account");
  });
}

const FEATURE_KEYS: Record<string, Key> = {
  search: "auth.feature.search",
  uploads_limited: "auth.feature.uploadsLimited",
  pro_badge: "auth.feature.proBadge",
  unlimited_uploads: "auth.feature.unlimitedUploads",
  notifications: "auth.feature.notifications",
};

// /pricing: карточки тарифов, «Перейти на Pro» → CreatePayment → демо-касса.
async function renderPricing(main: HTMLElement, alive: () => boolean): Promise<void> {
  const [list, { user }] = await Promise.all([plans(), loadMe()]);
  if (!alive()) return;
  const card = (p: PlanInfo) => {
    const current = (user?.plan ?? "free") === p.plan && !!user;
    const price = p.amountCents ? money(p.amountCents, p.currency) : t("auth.priceFree");
    const action =
      p.plan !== "pro"
        ? ""
        : user
          ? `<button class="btn btn--primary au-plan__btn" type="button" data-buy>${t(user.plan === "pro" ? "auth.extendPro" : "auth.goPro")}</button>`
          : `<a class="btn btn--primary au-plan__btn" href="/signup" data-link>${t("auth.signupToBuy")}</a>`;
    return `
      <article class="au-card au-plan${p.plan === "pro" ? " au-plan--pro" : ""}">
        <h2 class="au-plan__title">${escapeHtml(p.title)} ${p.plan === "pro" ? proBadge() : ""}</h2>
        <div class="au-plan__price">${escapeHtml(price)}${p.periodDays ? `<span class="au-muted"> / ${t("auth.month")}</span>` : ""}</div>
        <ul class="au-plan__features">
          ${p.features
            .map((f) => `<li>${escapeHtml(FEATURE_KEYS[f] ? t(FEATURE_KEYS[f], { n: p.uploadsPerDay }) : f)}</li>`)
            .join("")}
        </ul>
        ${current ? `<p class="au-plan__current">${t("auth.currentPlan")}</p>` : ""}
        ${action}
      </article>`;
  };
  main.innerHTML = `<div class="au-plans">${list.map(card).join("")}</div>
    <p class="au-error" role="alert" hidden></p>`;
  main.querySelector<HTMLButtonElement>("[data-buy]")?.addEventListener("click", async (e) => {
    const btn = e.currentTarget as HTMLButtonElement;
    btn.disabled = true;
    try {
      navigate(await createPayment());
    } catch (ex) {
      const err = main.querySelector<HTMLElement>(".au-error")!;
      err.hidden = false;
      err.textContent = ex instanceof ApiError ? ex.message : t("error.serverDown");
      btn.disabled = false;
    }
  });
}

const STATUS_KEYS: Record<Payment["status"], Key> = {
  pending: "auth.statusPending",
  succeeded: "auth.statusSucceeded",
  canceled: "auth.statusCanceled",
};

// /pay/mock/<id>: демо-касса. Сумму берём из истории платежей пользователя (свой платёж).
async function renderPay(main: HTMLElement, alive: () => boolean, id: string): Promise<void> {
  const { user } = await loadMe();
  if (!alive()) return;
  if (!user) return needLogin(main);
  const p = (await myPayments()).find((x) => x.id === id);
  if (!alive()) return;
  if (!p) {
    main.innerHTML = `<section class="au-card au-card--narrow"><p class="au-error">${t("auth.payNotFound")}</p>
      <a class="au-link" href="/account" data-link>${t("auth.toAccount")}</a></section>`;
    return;
  }
  const result = (q: Payment) => `
    <p class="au-pay__result ${q.status === "succeeded" ? "is-ok" : "is-cancel"}">${t(STATUS_KEYS[q.status])}</p>
    <a class="btn btn--primary" href="/account" data-link>${t("auth.toAccount")}</a>`;
  main.innerHTML = `
    <section class="au-card au-card--narrow au-pay">
      <div class="au-pay__demo">${t("auth.payDemo")}</div>
      <div class="au-pay__row"><span class="au-muted">${t("auth.payFor")}</span><strong>Pro · 30 ${t("auth.days")}</strong></div>
      <div class="au-pay__row"><span class="au-muted">${t("auth.amount")}</span><strong class="au-pay__sum">${escapeHtml(money(p.amountCents, p.currency))}</strong></div>
      <div class="au-pay__actions">
        ${
          p.status === "pending"
            ? `<button class="btn btn--ghost" type="button" data-pay="0">${t("auth.payCancel")}</button>
               <button class="btn btn--primary" type="button" data-pay="1">${t("auth.payBtn")}</button>`
            : result(p)
        }
      </div>
      <p class="au-error" role="alert" hidden></p>
    </section>`;
  main.querySelectorAll<HTMLButtonElement>("[data-pay]").forEach((b) =>
    b.addEventListener("click", async () => {
      main.querySelectorAll<HTMLButtonElement>("[data-pay]").forEach((x) => (x.disabled = true));
      try {
        const q = await confirmMockPayment(id, b.dataset.pay === "1");
        main.querySelector(".au-pay__actions")!.innerHTML = result(q);
        mePromise = null; // план поменялся
      } catch (ex) {
        const err = main.querySelector<HTMLElement>(".au-error")!;
        err.hidden = false;
        err.textContent = ex instanceof ApiError ? ex.message : t("error.serverDown");
      }
    })
  );
}

// /account: план, срок, история платежей, «Выйти».
async function renderAccount(main: HTMLElement, alive: () => boolean): Promise<void> {
  const { user } = await loadMe();
  if (!alive()) return;
  if (!user) return needLogin(main);
  const items = await myPayments();
  if (!alive()) return;
  const pro = user.plan === "pro";
  main.innerHTML = `
    <section class="au-card">
      <dl class="au-dl">
        <dt>${t("auth.email")}</dt><dd>${escapeHtml(user.email || "—")}</dd>
        <dt>ID</dt><dd class="au-mono">${escapeHtml(user.id)}</dd>
        <dt>${t("auth.plan")}</dt><dd>${pro ? proBadge() : `<span class="free-badge">Free</span>`}</dd>
        ${pro && user.proUntil ? `<dt>${t("auth.proUntil")}</dt><dd>${escapeHtml(formatDate(user.proUntil))}</dd>` : ""}
      </dl>
      <div class="au-actions">
        <a class="btn btn--primary" href="/pricing" data-link>${t(pro ? "auth.extendPro" : "auth.goPro")}</a>
        <button class="btn btn--ghost" type="button" data-logout>${t("auth.logout")}</button>
      </div>
    </section>
    <section class="au-card">
      <h2 class="au-h2">${t("auth.payments")}</h2>
      ${
        items.length
          ? `<div class="au-table-wrap"><table class="au-table">
              <thead><tr><th>${t("auth.date")}</th><th>${t("auth.plan")}</th><th>${t("auth.amount")}</th><th>${t("auth.status")}</th></tr></thead>
              <tbody>${items
                .map(
                  (p) => `<tr>
                    <td>${escapeHtml(formatDate(p.createdAt))}</td>
                    <td>${p.plan === "pro" ? "Pro" : "Free"}</td>
                    <td>${escapeHtml(money(p.amountCents, p.currency))}</td>
                    <td><span class="au-status au-status--${p.status}">${t(STATUS_KEYS[p.status])}</span>${
                      p.status === "pending" ? ` <a class="au-link" href="/pay/mock/${encodeURIComponent(p.id)}" data-link>${t("auth.payBtn")}</a>` : ""
                    }</td></tr>`
                )
                .join("")}</tbody></table></div>`
          : `<p class="au-muted">${t("auth.noPayments")}</p>`
      }
    </section>`;
  main.querySelector("[data-logout]")!.addEventListener("click", logout);
}
