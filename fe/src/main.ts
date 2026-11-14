import "./style.css";
import "./didyoumean.css";
import "./theme";
import "./adminmode"; // до первых запросов: сохранённый admin-token попадает в заголовки
import { ApiError, search, type Car } from "./api";
import { openClarify } from "./clarify";
import { navigate, onRoute, plateHref } from "./router";
import { renderPlatePage } from "./plate";
import { renderAdminPage } from "./admin";
import { renderNotFoundPage } from "./notfound";
import { isAuthRoute, renderAuthPage } from "./auth";
import { leaveCamerasPage, renderCamerasPage } from "./cameras";
import { renderUploadPage } from "./upload";
import { cardHtml, escapeHtml, formatDate, icon, plateHtml } from "./ui";
import { t, tp } from "./i18n/i18n";
import { highlightPlate } from "./searchkey";
import { bindRetry, cardSkeletonHtml, emptyStateHtml, errorStateHtml } from "./states";
import { mountSettings } from "./settings";
import { initHoverPreview } from "./hoverpreview";
import { footerHtml } from "./footer";
import { mountHomeAdmin, refreshHomeAdmin } from "./adminpanels";
import {
  flagHtml, gridColumns, homeMarket, marketName, setTabInUrl, tabFromUrl, tabsHtml, visibleTabs, type Market,
} from "./home";
import { bindEmptyWatch, emptyWatchHtml, migrateGuestWatches, mountWatchStrip } from "./watch";
import { activeCount, filtersRequest, mountFilters } from "./filters";
import { registerPwa } from "./pwa";

registerPwa();

const PAGE_SIZE = 35; // кратно 5 и 7 — по числу колонок сетки
const SKELETON_CARDS = 7; // скелетон первой страницы ленты — один ряд при широком экране
const DEBOUNCE_MS = 250;

// ---------------------------------------------------------------------------
// Разметка
// ---------------------------------------------------------------------------


function field(id: string, label: string, ico: string, placeholder: string): string {
  return `
    <label class="field" for="${id}">
      <span class="field__icon">${ico}</span>
      <span class="field__body">
        <span class="field__label">${label}</span>
        <input class="field__input" id="${id}" type="search" autocomplete="off" spellcheck="false" maxlength="32" placeholder="${placeholder}" />
      </span>
      <button class="field__clear" type="button" data-clear="${id}" aria-label="${t("search.clear", { label })}" hidden>${icon.clear}</button>
    </label>`;
}

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <div class="view" id="view-search">
  <header class="hero">
    <div class="hero__inner">
      <h1 class="hero__title"><span class="hero__logo"></span><span>${t("app.title")}</span></h1>
      <p>${t("home.hint")}</p>
      <form class="searchbar" role="search" onsubmit="return false">
        ${field("q-plate", t("search.label"), icon.plate, t("search.placeholder"))}
        <span class="searchbar__go" aria-hidden="true">${icon.search}</span>
      </form>
    </div>
  </header>

  <main class="content">
    <div class="filters" id="filters"></div>
    <section class="wl" aria-labelledby="wl-title" hidden></section>
    <section class="fy" aria-labelledby="fy-title" hidden>
      <div class="fy__head">
        <h2 class="fy__title" id="fy-title">${flagHtml(homeMarket())}${t("homeTabs.forYou")} · <span class="fy__market">${escapeHtml(marketName(homeMarket()))}</span></h2>
        <span class="fy__sub">${t("homeTabs.forYouHint")}</span>
      </div>
      <div class="grid fy__grid"></div>
    </section>
    <nav class="tabs" role="tablist" aria-label="${t("homeTabs.aria")}" hidden></nav>
    <div class="content__head">
      <h2 class="content__title">${t("results.title")}</h2>
      <span class="content__count" aria-live="polite"></span>
    </div>
    <section class="grid feed" id="feed"></section>
    <div class="more" hidden>
      <button class="more__btn" type="button">${t("more.load")}</button>
      <span class="more__info"></span>
    </div>
  </main>
  <footer class="cam-homelink"><a href="/cameras" data-link>${t("cameras.homeLink")}</a></footer>
  </div>

  <div class="view" id="view-plate" hidden></div>
  <div class="view" id="view-admin" hidden></div>
  <div class="view" id="view-auth" hidden></div>
  <div class="view" id="view-upload" hidden></div>
  ${footerHtml()}
  <div class="view" id="view-cameras" hidden></div>
  <div class="view" id="view-notfound" hidden></div>

`;

const $plate = app.querySelector<HTMLInputElement>("#q-plate")!;
const $grid = app.querySelector<HTMLElement>(".feed")!;
const $title = app.querySelector<HTMLElement>(".content__title")!;
const $fy = app.querySelector<HTMLElement>(".fy")!;
const $fyGrid = app.querySelector<HTMLElement>(".fy__grid")!;
const $tabs = app.querySelector<HTMLElement>(".tabs")!;
const $count = app.querySelector<HTMLElement>(".content__count")!;
const $more = app.querySelector<HTMLElement>(".more")!;
const $moreBtn = app.querySelector<HTMLButtonElement>(".more__btn")!;
const $moreInfo = app.querySelector<HTMLElement>(".more__info")!;

mountSettings(app.querySelector<HTMLElement>(".hero__logo")!);

// Полоса «Отслеживаемые» (watch.ts); гостевые подписки после входа переносятся на бэк.
const $wl = app.querySelector<HTMLElement>(".wl")!;
const refreshWatches = mountWatchStrip($wl);
void migrateGuestWatches().then(refreshWatches);

// Фильтры выдачи (filters.ts): любое изменение — новая выдача.
const filters = mountFilters(app.querySelector<HTMLElement>("#filters")!, () => {
  clearTimeout(debounce);
  newSearch();
});



// ---------------------------------------------------------------------------
// Состояние и загрузка страниц
// ---------------------------------------------------------------------------

const state = {
  generation: 0, // растёт при каждом новом запросе — ответы старых запросов отбрасываются
  loading: false,
  total: 0,
  nextOffset: 0 as number | null, // null — всё загружено
  failed: false,
  cars: new Map<string, Car>(), // машины ленты
  ctrl: null as AbortController | null,
  tab: tabFromUrl(), // вкладка главной (рынок ленты без запроса)
  skip: 0, // ленту вкладки «для вас» начинаем после карточек блока — они уже показаны выше
};

/** Все загруженные машины (лента и «для вас») — для модалки «Уточнить». */
const known = new Map<string, Car>();

/** Есть запрос — обычная выдача поиска; нет — главная: «для вас», вкладки и лента вкладки. */
const hasQuery = () => $plate.value.trim() !== "";

function render(): void {
  const shown = state.cars.size + state.skip;
  const home = !hasQuery();
  // «Для вас» скрыт, если пуст; при его ошибке — виден, пока лента не упала тоже (одна ошибка на экран).
  $wl.classList.toggle("is-off", !home); // полоса — только на главной без запроса
  $fy.hidden = !home || (forYou.cars.length === 0 && !forYou.loading && !(forYou.failed && !state.failed));
  $tabs.hidden = !home;
  $title.textContent = home ? marketName(state.tab) : t("results.title");
  $count.textContent = state.total
    ? tp("results.found", state.total)
    : "";

  const hasMore = state.nextOffset !== null;
  // Первая страница не пришла — «Повторить» уже в блоке ошибки ленты, второй кнопки не нужно.
  $more.hidden = (!hasMore && !state.failed) || (state.failed && state.cars.size === 0);
  $moreBtn.disabled = state.loading;
  $moreBtn.classList.toggle("is-loading", state.loading);
  $moreBtn.textContent = t(state.loading ? "more.loading" : state.failed ? "more.retry" : "more.load");
  $moreInfo.textContent = state.failed
    ? t("more.failed")
    : state.total
      ? t("more.shown", { shown, total: state.total })
      : "";
}

async function loadMore(): Promise<void> {
  if (state.loading || state.nextOffset === null) return;

  const gen = state.generation;
  const ctrl = (state.ctrl = new AbortController());
  state.loading = true;
  state.failed = false;
  if (state.cars.size === 0) $grid.innerHTML = cardSkeletonHtml(SKELETON_CARDS);
  render();

  try {
    const home = !hasQuery();
    if (home && state.tab === homeMarket() && activeCount(filters.get(), true) === 0) {
      // Лента вкладки языка продолжает блок «для вас»: ждём его и пропускаем его карточки.
      const skip = await forYou.ready;
      if (gen !== state.generation) return;
      if (state.nextOffset === 0) state.nextOffset = state.skip = skip;
    }
    const page = await search(
      home
        ? { market: state.tab, ...filtersRequest(filters.get(), true), offset: state.nextOffset!, limit: PAGE_SIZE }
        : { plate: $plate.value, ...filtersRequest(filters.get(), false), offset: state.nextOffset!, limit: PAGE_SIZE },
      ctrl.signal
    );
    if (gen !== state.generation) return; // пока грузили, запрос поменялся

    const startIndex = state.cars.size;
    if (startIndex === 0) $grid.innerHTML = ""; // убрать скелетон
    page.items.forEach((c) => (state.cars.set(c.id, c), known.set(c.id, c)));
    $grid.insertAdjacentHTML("beforeend", page.items.map((c, i) => cardHtml(c, i)).join(""));
    if (!home) highlightCards(startIndex, page.items);
    state.total = page.total;
    state.nextOffset = page.nextOffset;

    if (startIndex === 0 && page.items.length === 0) renderEmpty(page.suggestions);
  } catch (e) {
    if (ctrl.signal.aborted || gen !== state.generation) return;
    state.failed = true;
    if (state.cars.size === 0) {
      // Первая страница не пришла — блок ошибки с «Повторить» (и автоповтором при возврате сети).
      $grid.innerHTML = errorStateHtml(e);
      bindRetry($grid, retryFeed);
    }
  } finally {
    if (gen === state.generation) {
      state.loading = false;
      render();
      // Если блок «Загрузить ещё» всё ещё на экране — догружаем дальше.
      if (!state.failed) rearmObserver();
    }
  }
}

/** Подсветить совпадение с запросом в номерах только что добавленных карточек выдачи
 *  (похожие символы — как на бэке, searchkey.ts). */
function highlightCards(from: number, cars: Car[]): void {
  const cards = $grid.querySelectorAll<HTMLElement>(".card .card__plate");
  cars.forEach((c, i) => {
    const el = cards[from + i];
    if (el) highlightPlate(el, c.plate, $plate.value, "dym-mark");
  });
}

/** Блок «Возможно, вы искали» — номера, близкие к запросу (подсказки бэка при пустой выдаче). */
function suggestionsHtml(cars: Car[]): string {
  if (!cars.length) return "";
  return `<div class="dym"><span class="dym__title">${t("dym.title")}</span>${cars
    .map(
      (c) => `<button class="dym__plate" type="button" data-suggest="${escapeHtml(c.plate)}"
        aria-label="${escapeHtml(t("dym.search", { plate: c.plate }))}">${plateHtml(c)}</button>`
    )
    .join("")}</div>`;
}

/** Пустая выдача при заданных фильтрах — подсказка, что их можно сбросить. */
const filtersHint = (home: boolean): string =>
  activeCount(filters.get(), home) ? `<span class="flt__empty">${escapeHtml(t("filters.emptyHint"))}</span>` : "";

function renderEmpty(suggestions: Car[] = []): void {
  if (hasQuery()) {
    $grid.innerHTML = emptyStateHtml(t("empty.title"), t("empty.hint"), filtersHint(false) + suggestionsHtml(suggestions) + emptyWatchHtml($plate.value));
    void bindEmptyWatch($grid); // «подписаться на появление» (watch.ts)
  } else if (state.skip > 0) {
    $grid.innerHTML = `<p class="feed__note">${t("homeTabs.allAbove")}</p>`;
  } else {
    $grid.innerHTML = emptyStateHtml(t("homeTabs.emptyTitle"), t("homeTabs.emptyHint"), filtersHint(true));
  }
}

/** «Повторить» в блоке ошибки (и автоповтор при возврате сети): блок «для вас», если он не загрузился,
 *  грузится заново вместе с лентой — иначе первые карточки ленты повторили бы его. */
function retryFeed(): void {
  if (!hasQuery() && forYou.cars.length === 0) forYou.started = false;
  newSearch();
}

function newSearch(): void {
  state.ctrl?.abort();
  state.generation++;
  state.loading = false;
  state.failed = false;
  state.total = 0;
  state.nextOffset = 0;
  state.skip = 0;
  state.cars.clear();
  $grid.innerHTML = "";
  filters.setHome(!hasQuery());
  if (!hasQuery()) {
    loadForYou();
    setTabInUrl(state.tab);
  }
  render();
  void loadMore();
}

// ---------------------------------------------------------------------------
// Главная: «для вас» и вкладки
// ---------------------------------------------------------------------------

/** Блок «для вас»: первые 2 ряда номеров рынка языка; ready — сколько карточек показано. */
const forYou = {
  started: false,
  loading: false,
  failed: false,
  cars: [] as Car[],
  ready: Promise.resolve(0),
};

/** Грузится один раз: число карточек — 2 ряда при ширине сетки на момент загрузки. */
function loadForYou(): void {
  if (forYou.started) return;
  forYou.started = true;
  forYou.failed = false;
  forYou.loading = true;
  $fy.hidden = false; // чтобы сетка получила ширину и колонки
  $fyGrid.innerHTML = cardSkeletonHtml(Math.min(2 * gridColumns($fyGrid), 14));
  const limit = Math.min(2 * gridColumns($fyGrid), 100);
  forYou.ready = search({ market: homeMarket(), offset: 0, limit })
    .then((page) => {
      forYou.cars = page.items;
      page.items.forEach((c) => known.set(c.id, c));
      $fyGrid.innerHTML = page.items.map((c, i) => cardHtml(c, i)).join("");
      return page.items.length;
    })
    .catch((e) => {
      // Не загрузился — лента вкладки начинается с начала; в блоке — ошибка с «Повторить»
      // (видна, только если сама лента загрузилась, — иначе ошибку показывает лента).
      forYou.failed = true;
      $fyGrid.innerHTML = errorStateHtml(e, { compact: true });
      bindRetry($fyGrid, retryFeed);
      return 0;
    })
    .finally(() => {
      forYou.loading = false;
      render();
    });
}

function renderTabs(): void {
  $tabs.innerHTML = tabsHtml(state.tab, "feed");
  $grid.setAttribute("role", "tabpanel");
  $grid.setAttribute("aria-labelledby", `tab-${state.tab}`);
}

function selectTab(m: Market, focus = false): void {
  if (m === state.tab) return;
  state.tab = m;
  renderTabs();
  if (focus) $tabs.querySelector<HTMLElement>(`[data-tab="${m}"]`)?.focus();
  $tabs.querySelector(`[data-tab="${m}"]`)?.scrollIntoView({ block: "nearest", inline: "nearest" });
  newSearch();
}

$tabs.addEventListener("click", (e) => {
  const b = (e.target as Element).closest<HTMLButtonElement>("[data-tab]");
  if (b) selectTab(b.dataset.tab as Market);
});
// Стрелки, Home/End — по вкладкам (шаблон WAI-ARIA tabs).
$tabs.addEventListener("keydown", (e) => {
  const tabs = visibleTabs();
  const i = tabs.findIndex((x) => x.market === state.tab);
  const next = { ArrowRight: i + 1, ArrowLeft: i - 1, Home: 0, End: tabs.length - 1 }[e.key];
  if (next === undefined) return;
  e.preventDefault();
  selectTab(tabs[(next + tabs.length) % tabs.length].market, true);
});
renderTabs();

// Бесконечная прокрутка: когда блок «Загрузить ещё» подходит к экрану — грузим следующую страницу.
const observer = new IntersectionObserver(
  (entries) => {
    if (entries.some((e) => e.isIntersecting) && !state.failed) void loadMore();
  },
  { rootMargin: "600px 0px" }
);
function rearmObserver(): void {
  // Переподписка заставляет наблюдатель заново проверить видимость:
  // на высоком экране блок может остаться видимым и после догрузки.
  observer.unobserve($more);
  observer.observe($more);
}
observer.observe($more);

// Кнопка «Уточнить» на карточке — модалка уточнения (clarify.ts).
app.querySelector(".content")!.addEventListener("click", (e) => {
  const btn = (e.target as Element).closest<HTMLButtonElement>("[data-clarify]");
  const car = btn && known.get(btn.dataset.clarify ?? "");
  if (car) openClarify(car);
});

// ---------------------------------------------------------------------------
// Поиск
// ---------------------------------------------------------------------------

/** Запрос из адреса (?q=) — с ним открывается выдача (F5, ссылка, «Назад», смена языка). */
const queryFromUrl = (): string => (location.pathname === "/" && new URLSearchParams(location.search).get("q")) || "";

/** Записать запрос в адрес (пусто — без параметра), не добавляя запись в историю. */
function setQueryInUrl(q: string): void {
  if (location.pathname !== "/") return;
  const url = new URL(location.href);
  if (q.trim()) url.searchParams.set("q", q);
  else url.searchParams.delete("q");
  if (url.href !== location.href) history.replaceState(history.state, "", url);
}

/** Поставить запрос в поле поиска (кнопка очистки — по нему). */
function setQuery(q: string): void {
  $plate.value = q;
  app.querySelector<HTMLButtonElement>(`[data-clear="${$plate.id}"]`)!.hidden = !q;
}

let debounce: number | undefined;
function onInput(): void {
  app.querySelector<HTMLButtonElement>(`[data-clear="${$plate.id}"]`)!.hidden = !$plate.value;
  setQueryInUrl($plate.value);
  clearTimeout(debounce);
  debounce = window.setTimeout(newSearch, DEBOUNCE_MS);
}

$plate.addEventListener("input", onInput);

// «Возможно, вы искали»: клик по номеру — поиск по нему.
$grid.addEventListener("click", (e) => {
  const b = (e.target as Element).closest<HTMLButtonElement>("[data-suggest]");
  if (!b) return;
  clearTimeout(debounce);
  setQuery(b.dataset.suggest ?? "");
  setQueryInUrl($plate.value);
  newSearch();
  window.scrollTo({ top: 0 });
});

app.querySelectorAll<HTMLButtonElement>("[data-clear]").forEach((btn) =>
  btn.addEventListener("click", (e) => {
    e.preventDefault(); // кнопка внутри <label> — не фокусировать поле дважды
    const input = app.querySelector<HTMLInputElement>(`#${btn.dataset.clear}`)!;
    input.value = "";
    onInput();
    input.focus();
  })
);

$moreBtn.addEventListener("click", () => {
  state.failed = false;
  void loadMore();
});

// ---------------------------------------------------------------------------
// Маршруты: поиск ↔ страница номера. Поиск не пересоздаётся — только скрывается,
// поэтому при возврате остаются запрос, загруженные карточки и позиция прокрутки.
// ---------------------------------------------------------------------------

initHoverPreview();

// Запрос — в адресе (?q=): F5 и смена языка (перезагрузка) открывают ту же выдачу.
setQuery(queryFromUrl());

const $viewSearch = app.querySelector<HTMLElement>("#view-search")!;
const $viewPlate = app.querySelector<HTMLElement>("#view-plate")!;
let searchStarted = false;
let searchScroll = 0;

const $viewAdmin = app.querySelector<HTMLElement>("#view-admin")!;
void mountHomeAdmin(app.querySelector<HTMLElement>("#view-search .content")!); // админ-режим: полоса «Админ»

const $viewAuth = app.querySelector<HTMLElement>("#view-auth")!;
const $viewCameras = app.querySelector<HTMLElement>("#view-cameras")!;
const $viewUpload = app.querySelector<HTMLElement>("#view-upload")!;
const $viewNotFound = app.querySelector<HTMLElement>("#view-notfound")!;

onRoute((route) => {
  $viewAuth.hidden = !isAuthRoute(route);
  if (isAuthRoute(route)) {
    if (!$viewSearch.hidden) searchScroll = window.scrollY;
    $viewSearch.hidden = $viewPlate.hidden = $viewAdmin.hidden = $viewCameras.hidden = $viewNotFound.hidden = true;
    leaveCamerasPage();
    renderAuthPage($viewAuth, route);
    return;
  }
  $viewAdmin.hidden = route.name !== "admin";
  $viewCameras.hidden = route.name !== "cameras";
  $viewNotFound.hidden = route.name !== "notfound";
  if (route.name === "cameras") {
    if (!$viewSearch.hidden) searchScroll = window.scrollY;
    $viewSearch.hidden = $viewPlate.hidden = true;
    renderCamerasPage($viewCameras);
    return;
  }
  leaveCamerasPage();
  $viewUpload.hidden = route.name !== "upload";
  if (route.name === "upload") {
    if (!$viewSearch.hidden) searchScroll = window.scrollY;
    $viewSearch.hidden = $viewPlate.hidden = true;
    renderUploadPage($viewUpload);
    return;
  }
  if (route.name === "notfound") {
    if (!$viewSearch.hidden) searchScroll = window.scrollY;
    $viewSearch.hidden = $viewPlate.hidden = true;
    renderNotFoundPage($viewNotFound);
    return;
  }
  if (route.name === "admin") {
    if (!$viewSearch.hidden) searchScroll = window.scrollY;
    $viewSearch.hidden = $viewPlate.hidden = true;
    renderAdminPage($viewAdmin);
    return;
  }
  if (route.name === "plate") {
    if (!$viewSearch.hidden) searchScroll = window.scrollY;
    $viewSearch.hidden = true;
    $viewPlate.hidden = false;
    void renderPlatePage($viewPlate, route.id, {
      open: (id) => navigate(plateHref(id)),
      // Enter в поиске шапки — полная выдача на главной с этим запросом.
      submit: (query) => {
        setQuery(query);
        searchStarted = true;
        searchScroll = 0;
        newSearch();
        navigate(query.trim() ? `/?q=${encodeURIComponent(query)}` : "/");
      },
    });
    return;
  }
  $viewPlate.hidden = true;
  $viewPlate.querySelector(".pp")?.replaceChildren(); // шапку с поиском оставляем
  $viewSearch.hidden = false;
  document.title = t("app.title");
  void refreshHomeAdmin();
  void refreshWatches(); // счётчики новых могли измениться
  if (!searchStarted) {
    searchStarted = true;
    newSearch();
  } else if (filters.syncFromUrl()) {
    // Адрес с другими фильтрами (переход по истории) — новая выдача с ними.
    clearTimeout(debounce);
    setQuery(queryFromUrl());
    if (!hasQuery()) state.tab = tabFromUrl();
    renderTabs();
    searchScroll = 0;
    newSearch();
  } else if (queryFromUrl() !== $plate.value) {
    // Адрес с другим запросом (переход по истории или ссылкой на главную) — новая выдача.
    clearTimeout(debounce);
    setQuery(queryFromUrl());
    if (!hasQuery()) state.tab = tabFromUrl();
    renderTabs();
    searchScroll = 0;
    newSearch();
  } else if (!hasQuery() && tabFromUrl() !== state.tab) {
    selectTab(tabFromUrl()); // адрес с другой вкладкой (переход по истории)
  }
  requestAnimationFrame(() => window.scrollTo({ top: searchScroll }));
});
