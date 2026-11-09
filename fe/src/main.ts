import "./style.css";
import { ApiError, search, type Car } from "./api";

const PAGE_SIZE = 35; // кратно 5 и 7 — по числу колонок сетки
const DEBOUNCE_MS = 250;

// ---------------------------------------------------------------------------
// Разметка
// ---------------------------------------------------------------------------

const icon = {
  search: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>`,
  plate: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="2" y="7" width="20" height="10" rx="2"/><path d="M16 7v10"/></svg>`,
  clear: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18"/></svg>`,
  pin: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M12 21s-7-6.2-7-11.5A7 7 0 0 1 19 9.5C19 14.8 12 21 12 21z"/><circle cx="12" cy="9.5" r="2.5"/></svg>`,
  cal: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/></svg>`,
  car: `<svg viewBox="0 0 64 32" fill="currentColor" aria-hidden="true"><path d="M10 22h-4a2 2 0 0 1-2-2v-4c0-1.5 1-2.6 2.4-3l6.6-1.8 6-5.4A6 6 0 0 1 23 4.5h14a6 6 0 0 1 4.3 1.8L47 12l9 1.6c1.8.3 3 1.9 3 3.6V20a2 2 0 0 1-2 2h-3"/><circle cx="16" cy="23" r="5"/><circle cx="48" cy="23" r="5"/></svg>`,
};

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
  <header class="hero">
    <div class="hero__inner">
      <h1>Поиск по авто</h1>
      <p>Введите госномер целиком или его часть: серию «А123ВС» или регион «777»</p>
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

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!
  );
}

const dateFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric", month: "long", year: "numeric", hour: "2-digit", minute: "2-digit",
});
const formatDate = (iso: string) => (iso ? dateFmt.format(new Date(iso)) : "");

function plateHtml(plate: string): string {
  const [num = "", reg = ""] = plate.split(" ");
  return `<span class="plate" aria-label="Госномер ${escapeHtml(plate)}"><span class="plate__num">${escapeHtml(num)}</span><span class="plate__reg">${escapeHtml(reg)}<span class="plate__rus">RUS<i class="plate__flag"></i></span></span></span>`;
}

function cardHtml(c: Car, i: number): string {
  const city = c.address.split(",")[0] ?? "";
  const photo = c.photoUrl ? `<img src="${escapeHtml(c.photoUrl)}" alt="" loading="lazy" />` : icon.car;
  return `
    <article class="card" style="animation-delay:${Math.min(i, 14) * 30}ms">
      <div class="card__photo">${photo}<span class="card__badge">${escapeHtml(city)}</span></div>
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

const plural = (n: number, one: string, few: string, many: string) => {
  const m10 = n % 10, m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return few;
  return many;
};

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
  $grid.innerHTML = `<div class="empty"><strong>Ничего не найдено</strong>Проверьте номер — ищется по серии целиком и региону</div>`;
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

newSearch();
