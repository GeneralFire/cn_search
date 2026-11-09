import "./style.css";
import { ApiError, clarify, search, type Car } from "./api";
import { navigate, onRoute, plateHref } from "./router";
import { renderPlatePage } from "./plate";
import { cardHtml, escapeHtml, formatDate, icon, plateHtml } from "./ui";
import { SWITCH_FLAG, t, tp } from "./i18n/i18n";
import { mountSettings } from "./settings";
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
    <div class="content__head">
      <h2>${t("results.title")}</h2>
      <span class="content__count" aria-live="polite"></span>
    </div>
    <section class="grid"></section>
    <div class="more" hidden>
      <button class="more__btn" type="button">${t("more.load")}</button>
      <span class="more__info"></span>
    </div>
  </main>
  </div>

  <div class="view" id="view-plate" hidden></div>

  <dialog class="modal" aria-labelledby="modal-title">
    <form method="dialog" class="modal__inner">
      <header class="modal__head">
        <h3 id="modal-title">${t("modal.title")}</h3>
        <button class="modal__x" value="close" formnovalidate aria-label="${t("modal.close")}">${icon.clear}</button>
      </header>
      <div class="modal__body"></div>
      <footer class="modal__foot">
        <button class="btn btn--primary" value="close">${t("modal.ok")}</button>
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
const $modalFoot = app.querySelector<HTMLElement>(".modal__foot")!;
const $modalForm = app.querySelector<HTMLFormElement>(".modal__inner")!;

mountSettings(app.querySelector<HTMLElement>(".hero__logo")!);



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
    ? tp("results.found", state.total)
    : "";

  const hasMore = state.nextOffset !== null;
  $more.hidden = !hasMore && !state.failed;
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
      $grid.innerHTML = `<div class="empty"><strong>${t("error.loadFailed")}</strong>${escapeHtml(
        e instanceof ApiError ? e.message : t("error.serverDown")
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
  $grid.innerHTML = `<div class="empty"><strong>${t("empty.title")}</strong>${t("empty.hint")}</div>`;
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
      ${plateHtml(car)}
      <div>
        <div class="modal__date">${escapeHtml(formatDate(car.createdAt))}</div>
        <div class="modal__addr">${escapeHtml(car.address)}</div>
      </div>
    </div>
    <fieldset class="clarify__formats">
      <legend class="clarify__label">${t("clarify.format")}</legend>
      ${[["ru", "RU"], ["eu", "EU"], ["us", "US"], ["", t("clarify.formatOther")]]
        .map(
          ([v, label]) => `<label class="seg"><input type="radio" name="ptype" value="${v}"${
            v === (car.plateType || "") ? " checked" : ""
          } /><span>${escapeHtml(label)}</span></label>`
        )
        .join("")}
    </fieldset>
    <label class="clarify">
      <span class="clarify__label">${t("clarify.question")}</span>
      <span class="clarify__row">
        <input class="clarify__input" name="plate" required maxlength="32" autocomplete="off" spellcheck="false"
          placeholder="${escapeHtml(car.plate)}" value="${escapeHtml(car.plate)}" />
        <input class="clarify__input clarify__region" name="region" maxlength="3" autocomplete="off" spellcheck="false"
          value="${escapeHtml(car.plateRegion ?? "")}" aria-label="${t("clarify.regionEu")}" />
      </span>
      <span class="clarify__hint">${t("clarify.hint")}</span>
    </label>
    <div class="clarify__preview"><span class="clarify__label">${t("clarify.preview")}</span><span class="clarify__plate"></span></div>
    <p class="clarify__msg" role="status" hidden></p>`;
  $modalFoot.innerHTML = `
    <button class="btn btn--ghost" value="close" formnovalidate>${t("clarify.cancel")}</button>
    <button class="btn btn--primary" value="send" type="submit">${t("clarify.send")}</button>`;
  clarifyFor = car;
  $modal.showModal();
  const $input = $modalBody.querySelector<HTMLInputElement>(".clarify__input")!;
  const $region = $modalBody.querySelector<HTMLInputElement>(".clarify__region")!;
  const $preview = $modalBody.querySelector<HTMLElement>(".clarify__plate")!;
  // Регион — только для EU (страна) и US (штат); предпросмотр номера — по выбранному формату.
  const sync = () => {
    const type = clarifyType();
    $region.hidden = type !== "eu" && type !== "us";
    $region.placeholder = type === "us" ? "CA" : "D";
    $region.setAttribute("aria-label", t(type === "us" ? "clarify.regionUs" : "clarify.regionEu"));
    const plate = $input.value.trim().toUpperCase() || car.plate;
    $preview.innerHTML = plateHtml({ plate, plateType: type, plateRegion: $region.value.trim().toUpperCase() });
  };
  $modalBody.addEventListener("input", sync);
  $modalBody.addEventListener("change", sync);
  sync();
  $input.select();
}

function clarifyType(): string {
  return $modalBody.querySelector<HTMLInputElement>('input[name="ptype"]:checked')?.value ?? "";
}

let clarifyFor: Car | null = null;

$modalForm.addEventListener("submit", async (e) => {
  const submitter = (e as SubmitEvent).submitter as HTMLButtonElement | null;
  if (submitter?.value !== "send" || !clarifyFor) return; // «Отмена» и крестик — обычное закрытие
  e.preventDefault();
  const $input = $modalBody.querySelector<HTMLInputElement>(".clarify__input")!;
  const $msg = $modalBody.querySelector<HTMLElement>(".clarify__msg")!;
  submitter.disabled = true;
  submitter.textContent = t("clarify.sending");
  try {
    const type = clarifyType();
    const region = $modalBody.querySelector<HTMLInputElement>(".clarify__region")!.value;
    const c = await clarify(clarifyFor.id, $input.value, type, type === "eu" || type === "us" ? region : "");
    $msg.hidden = false;
    $msg.className = "clarify__msg is-ok";
    $msg.textContent = t("clarify.thanks", {
      plate: c.plate,
      status: t(c.accepted ? "clarify.accepted" : "clarify.notAccepted"),
    });
    $input.disabled = true;
    $modalFoot.innerHTML = `<button class="btn btn--primary" value="close">${t("clarify.done")}</button>`;
  } catch (err) {
    $msg.hidden = false;
    $msg.className = "clarify__msg is-error";
    $msg.textContent = err instanceof ApiError ? err.message : t("clarify.failed");
    submitter.disabled = false;
    submitter.textContent = t("clarify.send");
  }
});

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

// Смена языка перезагружает страницу — запрос поиска переносим через перезагрузку.
const SEARCH_Q = "search-q";
addEventListener("pagehide", () => sessionStorage.setItem(SEARCH_Q, $plate.value));
if (sessionStorage.getItem(SWITCH_FLAG)) {
  sessionStorage.removeItem(SWITCH_FLAG);
  $plate.value = sessionStorage.getItem(SEARCH_Q) ?? "";
  app.querySelector<HTMLButtonElement>(`[data-clear="${$plate.id}"]`)!.hidden = !$plate.value;
}

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
  document.title = t("app.title");
  if (!searchStarted) {
    searchStarted = true;
    newSearch();
  }
  requestAnimationFrame(() => window.scrollTo({ top: searchScroll }));
});
