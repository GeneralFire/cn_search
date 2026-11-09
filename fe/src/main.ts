import "./style.css";
import { ApiError, search, type Car } from "./api";
import { navigate, onRoute, plateHref } from "./router";
import { renderPlatePage } from "./plate";
import { escapeHtml, formatDate, icon, photoAttrs, plateHtml, plural } from "./ui";
import { initHoverPreview } from "./hoverpreview";

const PAGE_SIZE = 35; // кратно 5 и 7 — по числу колонок сетки
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
      <button class="field__clear" type="button" data-clear="${id}" aria-label="Очистить: ${label}" hidden>${icon.clear}</button>
    </label>`;
}

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <div class="view" id="view-search">
  <header class="hero">
    <div class="hero__inner">
      <h1>Поиск по авто</h1>
      <p>Введите госномер или его начало: «У54», «А123ВС 777», регион «77»</p>
      <form class="searchbar" role="search" onsubmit="return false">
        ${field("q-plate", "Номер", icon.plate, "А123ВС 777")}
        <span class="searchbar__go" aria-hidden="true">${icon.search}</span>
      </form>
    </div>
  </header>

  <main class="content">
    <div class="content__head">
      <h2>Результаты</h2>
      <span class="content__count" aria-live="polite"></span>
    </div>
    <section class="grid"></section>
    <div class="more" hidden>
      <button class="more__btn" type="button">Загрузить ещё</button>
      <span class="more__info"></span>
    </div>
  </main>
  </div>

  <div class="view" id="view-plate" hidden></div>

  <dialog class="modal" aria-labelledby="modal-title">
    <form method="dialog" class="modal__inner">
      <header class="modal__head">
        <h3 id="modal-title">Уточнить данные</h3>
        <button class="modal__x" value="close" aria-label="Закрыть">${icon.clear}</button>
      </header>
      <div class="modal__body"></div>
      <footer class="modal__foot">
        <button class="btn btn--primary" value="close">Понятно</button>
      </footer>
    </form>
  </dialog>
`;

const $plate = app.querySelector<HTMLInputElement>("#q-plate")!;
const $grid = app.querySelector<HTMLElement>(".grid")!;
const $count = app.querySelector<HTMLElement>(".content__count")!;
const $more = app.querySelector<HTMLElement>(".more")!;
const $moreBtn = app.querySelector<HTMLButtonElement>(".more__btn")!;
const $moreInfo = app.querySelector<HTMLElement>(".more__info")!;
const $modal = app.querySelector<HTMLDialogElement>(".modal")!;
const $modalBody = app.querySelector<HTMLElement>(".modal__body")!;

function cardHtml(c: Car, i: number): string {
  const city = c.address.split(",")[0] ?? "";
  const { src, full } = photoAttrs(c);
  const photo = src ? `<img src="${escapeHtml(src)}" alt="" loading="lazy" />` : icon.car;
  return `
    <article class="card" style="animation-delay:${Math.min(i, 14) * 30}ms">
      <a class="card__link" href="${plateHref(c.id)}" data-link aria-label="${escapeHtml(c.plate)}: все фиксации"></a>
      <a class="card__photo" href="${plateHref(c.id)}" data-link tabindex="-1"${full ? ` data-full="${escapeHtml(full)}"` : ""}>${photo}<span class="card__badge">${escapeHtml(city)}</span></a>
      <div class="card__body">
        <p class="card__address">${icon.pin}${escapeHtml(c.address)}</p>
        <div class="card__row">${plateHtml(c.plate)}</div>
        <div class="card__foot">
          <span class="card__date">${icon.cal}<time datetime="${escapeHtml(c.createdAt)}">${formatDate(c.createdAt)}</time></span>
          <button class="btn btn--ghost" type="button" data-clarify="${escapeHtml(c.id)}">Уточнить</button>
        </div>
      </div>
    </article>`;
}


// ---------------------------------------------------------------------------
// Состояние и загрузка страниц
// ---------------------------------------------------------------------------

const state = {
  generation: 0, // растёт при каждом новом запросе — ответы старых запросов отбрасываются
  loading: false,
  total: 0,
  nextOffset: 0 as number | null, // null — всё загружено
  failed: false,
  cars: new Map<string, Car>(), // загруженные машины — для модалки
  ctrl: null as AbortController | null,
};

function render(): void {
  const shown = state.cars.size;
  $count.textContent = state.total
    ? `Найдено ${state.total} ${plural(state.total, "машина", "машины", "машин")}`
    : "";

  const hasMore = state.nextOffset !== null;
  $more.hidden = !hasMore && !state.failed;
  $moreBtn.disabled = state.loading;
  $moreBtn.classList.toggle("is-loading", state.loading);
  $moreBtn.textContent = state.loading ? "Загружаем…" : state.failed ? "Повторить" : "Загрузить ещё";
  $moreInfo.textContent = state.failed
    ? "Не удалось загрузить — проверьте соединение"
    : state.total
      ? `Показано ${shown} из ${state.total}`
      : "";
}

async function loadMore(): Promise<void> {
  if (state.loading || state.nextOffset === null) return;

  const gen = state.generation;
  const ctrl = (state.ctrl = new AbortController());
  state.loading = true;
  state.failed = false;
  render();

  try {
    const page = await search(
      { plate: $plate.value, offset: state.nextOffset, limit: PAGE_SIZE },
      ctrl.signal
    );
    if (gen !== state.generation) return; // пока грузили, запрос поменялся

    const startIndex = state.cars.size;
    page.items.forEach((c) => state.cars.set(c.id, c));
    $grid.insertAdjacentHTML("beforeend", page.items.map((c, i) => cardHtml(c, i)).join(""));
    state.total = page.total;
    state.nextOffset = page.nextOffset;

    if (startIndex === 0 && page.items.length === 0) renderEmpty();
  } catch (e) {
    if (ctrl.signal.aborted || gen !== state.generation) return;
    state.failed = true;
    if (state.cars.size === 0) {
      $grid.innerHTML = `<div class="empty"><strong>Не удалось загрузить</strong>${escapeHtml(
        e instanceof ApiError ? e.message : "Сервер недоступен"
      )}</div>`;
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

function renderEmpty(): void {
  $grid.innerHTML = `<div class="empty"><strong>Ничего не найдено</strong>Проверьте номер — ищется по началу серии и региона</div>`;
}

function newSearch(): void {
  state.ctrl?.abort();
  state.generation++;
  state.loading = false;
  state.failed = false;
  state.total = 0;
  state.nextOffset = 0;
  state.cars.clear();
  $grid.innerHTML = "";
  void loadMore();
}

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

// ---------------------------------------------------------------------------
// Модалка «Уточнить» — заглушка
// ---------------------------------------------------------------------------

function openClarify(car: Car): void {
  $modalBody.innerHTML = `
    <div class="modal__car">
      ${plateHtml(car.plate)}
      <div>
        <div class="modal__date">${escapeHtml(formatDate(car.createdAt))}</div>
        <div class="modal__addr">${escapeHtml(car.address)}</div>
      </div>
    </div>
    <p class="modal__stub">Здесь будет форма уточнения данных по машине. Пока это заглушка.</p>`;
  $modal.showModal();
}

$grid.addEventListener("click", (e) => {
  const btn = (e.target as Element).closest<HTMLButtonElement>("[data-clarify]");
  const car = btn && state.cars.get(btn.dataset.clarify ?? "");
  if (car) openClarify(car);
});

// Клик по подложке закрывает модалку.
$modal.addEventListener("click", (e) => {
  if (e.target === $modal) $modal.close();
});

// ---------------------------------------------------------------------------
// Поиск
// ---------------------------------------------------------------------------

let debounce: number | undefined;
function onInput(): void {
  app.querySelector<HTMLButtonElement>(`[data-clear="${$plate.id}"]`)!.hidden = !$plate.value;
  clearTimeout(debounce);
  debounce = window.setTimeout(newSearch, DEBOUNCE_MS);
}

$plate.addEventListener("input", onInput);

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

const $viewSearch = app.querySelector<HTMLElement>("#view-search")!;
const $viewPlate = app.querySelector<HTMLElement>("#view-plate")!;
let searchStarted = false;
let searchScroll = 0;

onRoute((route) => {
  if (route.name === "plate") {
    if (!$viewSearch.hidden) searchScroll = window.scrollY;
    $viewSearch.hidden = true;
    $viewPlate.hidden = false;
    void renderPlatePage($viewPlate, route.id, {
      open: (id) => navigate(plateHref(id)),
      // Enter в поиске шапки — полная выдача на главной с этим запросом.
      submit: (query) => {
        $plate.value = query;
        app.querySelector<HTMLButtonElement>(`[data-clear="${$plate.id}"]`)!.hidden = !query;
        searchStarted = true;
        searchScroll = 0;
        newSearch();
        navigate("/");
      },
    });
    return;
  }
  $viewPlate.hidden = true;
  $viewPlate.querySelector(".pp")?.replaceChildren(); // шапку с поиском оставляем
  $viewSearch.hidden = false;
  document.title = "Поиск по авто";
  if (!searchStarted) {
    searchStarted = true;
    newSearch();
  }
  requestAnimationFrame(() => window.scrollTo({ top: searchScroll }));
});
