// Поле поиска в шапке страницы номера: при вводе — выпадающий список номеров с прокруткой
// и догрузкой страниц. Один номер — одна строка (последняя фиксация из выдачи).
import { search, type Car } from "./api";
import { errorLine } from "./states";
import { escapeHtml, formatAgo, icon, plateHtml } from "./ui";
import { t, tp } from "./i18n/i18n";
import { highlightPlate, plateSearchKey } from "./searchkey";
import "./quicksearch.css";
import { photoHtml } from "./nophoto";

const PAGE_SIZE = 35;
const DEBOUNCE_MS = 200;
const RECENT_KEY = "qs.recent";
const RECENT_MAX = 5;

/** Недавние запросы (localStorage), от новых к старым. */
function loadRecent(): string[] {
  try {
    const v = JSON.parse(localStorage.getItem(RECENT_KEY) ?? "[]");
    return Array.isArray(v) ? v.filter((x) => typeof x === "string").slice(0, RECENT_MAX) : [];
  } catch {
    return [];
  }
}

function saveRecent(list: string[]): void {
  localStorage.setItem(RECENT_KEY, JSON.stringify(list.slice(0, RECENT_MAX)));
}

/** Запомнить запрос; похожие написания («о515» и «О515») считаем одним запросом. */
function rememberQuery(q: string): void {
  q = q.trim();
  if (!q) return;
  const key = plateSearchKey(q).toLowerCase();
  saveRecent([q, ...loadRecent().filter((x) => plateSearchKey(x).toLowerCase() !== key)]);
}

export interface QuickSearchHandlers {
  open(id: string): void; // открыть страницу номера (по id фиксации)
  submit(query: string): void; // полная выдача на главной
}

export function mountQuickSearch(root: HTMLElement, h: QuickSearchHandlers): void {
  root.innerHTML = `
    <div class="qs" role="combobox" aria-haspopup="listbox" aria-owns="qs-list" aria-expanded="false">
      <span class="qs__icon">${icon.search}</span>
      <input class="qs__input" type="search" autocomplete="off" spellcheck="false" maxlength="32"
        placeholder="${t("qs.placeholder")}" aria-label="${t("qs.aria")}" aria-controls="qs-list" />
      <div class="qs__panel" hidden>
        <ul class="qs__list" id="qs-list" role="listbox"></ul>
        <div class="qs__foot"></div>
      </div>
      <div class="qs__recent" role="group" aria-label="${t("qs.recent.title")}" hidden></div>
    </div>`;

  const $box = root.querySelector<HTMLElement>(".qs")!;
  const $input = root.querySelector<HTMLInputElement>(".qs__input")!;
  const $panel = root.querySelector<HTMLElement>(".qs__panel")!;
  const $list = root.querySelector<HTMLUListElement>(".qs__list")!;
  const $foot = root.querySelector<HTMLElement>(".qs__foot")!;
  const $recent = root.querySelector<HTMLElement>(".qs__recent")!;

  const st = {
    gen: 0,
    loading: false,
    total: 0,
    nextOffset: 0 as number | null,
    plates: new Map<string, Car>(), // номер → последняя фиксация из выдачи
    active: -1,
    ctrl: null as AbortController | null,
  };
  let debounce: number | undefined;

  const setOpen = (open: boolean) => {
    $panel.hidden = !open;
    $box.setAttribute("aria-expanded", String(open));
  };

  /** Чипы недавних запросов — под пустым полем в фокусе. */
  function renderRecent(): void {
    const list = loadRecent();
    const show = !$input.value.trim() && list.length > 0 && $box.contains(document.activeElement);
    $recent.hidden = !show;
    if (!show) {
      $recent.innerHTML = "";
      return;
    }
    $recent.innerHTML = `
      <span class="qs__recent-title">${t("qs.recent.title")}</span>
      ${list
        .map(
          (q) => `<span class="qs__chip">
            <button class="qs__chip-q" type="button" data-q="${escapeHtml(q)}">${escapeHtml(q)}</button>
            <button class="qs__chip-x" type="button" data-remove="${escapeHtml(q)}"
              aria-label="${escapeHtml(t("qs.recent.remove", { q }))}">${icon.clear}</button>
          </span>`
        )
        .join("")}
      <button class="qs__recent-clear" type="button" data-clear>${t("qs.recent.clear")}</button>`;
  }

  function renderFoot(error = ""): void {
    $foot.innerHTML = error
      ? `<span class="qs__err">${escapeHtml(error)}</span><button class="qs__retry" type="button" data-qs-retry>${t("states.retry")}</button>`
      : st.loading
        ? `<span class="qs__spinner"></span>${t("qs.searching")}`
        : st.plates.size === 0
          ? t("qs.empty")
          : `${tp("qs.plates", st.plates.size)}${st.nextOffset === null ? "" : "+"} · ${t("qs.all", { total: st.total })}`;
  }

  function itemHtml(c: Car, i: number): string {
    return `
      <li class="qs__item" role="option" id="qs-opt-${i}" data-id="${escapeHtml(c.id)}" aria-selected="false">
        ${plateHtml(c)}
        <span class="qs__thumb qs__thumb-box">${photoHtml(c.thumbUrl ?? "")}</span>
        <span class="qs__meta">
          <span class="qs__addr">${escapeHtml(c.address)}</span>
          <span class="qs__ago">${escapeHtml(formatAgo(c.createdAt))}</span>
        </span>
      </li>`;
  }

  async function loadMore(): Promise<void> {
    if (st.loading || st.nextOffset === null) return;
    const gen = st.gen;
    const ctrl = (st.ctrl = new AbortController());
    st.loading = true;
    renderFoot();
    try {
      const page = await search({ plate: $input.value, offset: st.nextOffset, limit: PAGE_SIZE }, ctrl.signal);
      if (gen !== st.gen) return;
      // Один номер — одна строка: выдача от новых к старым, первая встреча номера — последняя фиксация.
      const start = st.plates.size;
      const fresh: Car[] = [];
      for (const c of page.items) {
        if (st.plates.has(c.plate)) continue;
        st.plates.set(c.plate, c);
        fresh.push(c);
      }
      $list.insertAdjacentHTML("beforeend", fresh.map((c, i) => itemHtml(c, start + i)).join(""));
      // Подсветка совпавшего начала слов номера — с учётом похожих символов, как ищет бэк.
      fresh.forEach((c, i) => {
        const el = $list.querySelector(`#qs-opt-${start + i}`);
        if (el) highlightPlate(el, c.plate, $input.value);
      });
      st.total = page.total;
      st.nextOffset = page.nextOffset;
      st.loading = false;
      renderFoot();
      // Если номеров мало и прокрутки ещё нет — догружаем, пока не появится или не кончится выдача.
      if ($list.scrollHeight <= $list.clientHeight + 4) void loadMore();
    } catch (e) {
      if (ctrl.signal.aborted || gen !== st.gen) return;
      st.loading = false;
      // Кратко и без «rpc error: …»; «Повторить» — та же страница ещё раз.
      renderFoot(errorLine(e));
      $foot.querySelector("[data-qs-retry]")?.addEventListener("mousedown", (ev) => {
        ev.preventDefault(); // не терять фокус поля — иначе выпадашка закроется
        void loadMore();
      });
    }
  }

  function newQuery(): void {
    st.ctrl?.abort();
    st.gen++;
    st.loading = false;
    st.total = 0;
    st.nextOffset = 0;
    st.plates.clear();
    st.active = -1;
    $list.innerHTML = "";
    $list.scrollTop = 0;
    renderRecent();
    if (!$input.value.trim()) return setOpen(false);
    setOpen(true);
    void loadMore();
  }

  function setActive(i: number): void {
    const items = $list.querySelectorAll<HTMLElement>(".qs__item");
    if (!items.length) return;
    st.active = (i + items.length) % items.length;
    items.forEach((el, j) => el.setAttribute("aria-selected", String(j === st.active)));
    const el = items[st.active];
    el.scrollIntoView({ block: "nearest" });
    $input.setAttribute("aria-activedescendant", el.id);
  }

  function choose(id: string): void {
    rememberQuery($input.value);
    setOpen(false);
    $input.blur();
    h.open(id);
  }

  $input.addEventListener("input", () => {
    if (!$input.value.trim()) renderRecent();
    else $recent.hidden = true;
    clearTimeout(debounce);
    debounce = window.setTimeout(newQuery, DEBOUNCE_MS);
  });
  $input.addEventListener("focus", () => {
    if ($input.value.trim() && $list.children.length) setOpen(true);
    renderRecent();
  });
  // Фокус ушёл за пределы поиска (Tab, клик мимо) — прячем недавние.
  $box.addEventListener("focusout", (e) => {
    if (!$box.contains(e.relatedTarget as Node | null)) $recent.hidden = true;
  });
  // mousedown не забирает фокус у поля — панель не мигает; действие — по click (работает и с клавиатуры).
  $recent.addEventListener("mousedown", (e) => {
    if ((e.target as Element).closest("button")) e.preventDefault();
  });
  $recent.addEventListener("click", (e) => {
    const b = (e.target as Element).closest<HTMLButtonElement>("button");
    if (!b) return;
    if (b.dataset.q !== undefined) {
      clearTimeout(debounce);
      $input.value = b.dataset.q;
      $input.focus();
      rememberQuery(b.dataset.q);
      newQuery();
    } else if (b.dataset.remove !== undefined) {
      saveRecent(loadRecent().filter((x) => x !== b.dataset.remove));
      renderRecent();
      if ($recent.hidden) $input.focus();
      else $recent.querySelector<HTMLElement>(".qs__chip-q, .qs__recent-clear")?.focus();
    } else if (b.dataset.clear !== undefined) {
      saveRecent([]);
      $input.focus();
      renderRecent();
    }
  });
  $input.addEventListener("keydown", (e) => {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if ($panel.hidden && $input.value.trim()) setOpen(true);
      setActive(st.active + (e.key === "ArrowDown" ? 1 : -1));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const el = $list.querySelectorAll<HTMLElement>(".qs__item")[st.active];
      if (el && !$panel.hidden) choose(el.dataset.id!);
      else if ($input.value.trim()) {
        rememberQuery($input.value);
        setOpen(false);
        h.submit($input.value);
      }
    } else if (e.key === "Escape") {
      // В type=search браузер сам стирает текст по Esc — при открытом списке только закрываем его.
      if (!$panel.hidden) {
        e.preventDefault();
        setOpen(false);
      } else if (!$recent.hidden) {
        e.preventDefault();
        $recent.hidden = true;
      }
    }
  });

  // Догрузка при прокрутке списка.
  $list.addEventListener("scroll", () => {
    if ($list.scrollTop + $list.clientHeight >= $list.scrollHeight - 120) void loadMore();
  });
  // mousedown, а не click: срабатывает раньше blur.
  $list.addEventListener("mousedown", (e) => {
    const el = (e.target as Element).closest<HTMLElement>(".qs__item");
    if (!el) return;
    e.preventDefault();
    choose(el.dataset.id!);
  });
  $list.addEventListener("mousemove", (e) => {
    const el = (e.target as Element).closest<HTMLElement>(".qs__item");
    if (el) setActive([...$list.children].indexOf(el));
  });
  document.addEventListener("mousedown", (e) => {
    if (!$box.contains(e.target as Node)) {
      setOpen(false);
      $recent.hidden = true;
    }
  });
}
