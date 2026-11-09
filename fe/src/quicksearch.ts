// Поле поиска в шапке страницы номера: при вводе — выпадающий список номеров с прокруткой
// и догрузкой страниц. Один номер — одна строка (последняя фиксация из выдачи).
import { ApiError, search, type Car } from "./api";
import { escapeHtml, formatAgo, icon, plateHtml } from "./ui";

const PAGE_SIZE = 35;
const DEBOUNCE_MS = 200;

export interface QuickSearchHandlers {
  open(id: string): void; // открыть страницу номера (по id фиксации)
  submit(query: string): void; // полная выдача на главной
}

export function mountQuickSearch(root: HTMLElement, h: QuickSearchHandlers): void {
  root.innerHTML = `
    <div class="qs" role="combobox" aria-haspopup="listbox" aria-owns="qs-list" aria-expanded="false">
      <span class="qs__icon">${icon.search}</span>
      <input class="qs__input" type="search" autocomplete="off" spellcheck="false" maxlength="32"
        placeholder="Новый поиск: У54, А123ВС 777…" aria-label="Поиск по номеру" aria-controls="qs-list" />
      <div class="qs__panel" hidden>
        <ul class="qs__list" id="qs-list" role="listbox"></ul>
        <div class="qs__foot"></div>
      </div>
    </div>`;

  const $box = root.querySelector<HTMLElement>(".qs")!;
  const $input = root.querySelector<HTMLInputElement>(".qs__input")!;
  const $panel = root.querySelector<HTMLElement>(".qs__panel")!;
  const $list = root.querySelector<HTMLUListElement>(".qs__list")!;
  const $foot = root.querySelector<HTMLElement>(".qs__foot")!;

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

  function renderFoot(error = ""): void {
    $foot.innerHTML = error
      ? `<span class="qs__err">${escapeHtml(error)}</span>`
      : st.loading
        ? `<span class="qs__spinner"></span>Ищем…`
        : st.plates.size === 0
          ? "Ничего не найдено — ищется по началу серии и региона"
          : `${st.plates.size} ${st.nextOffset === null ? "" : "+ "}номеров · <kbd>Enter</kbd> — все ${st.total} фиксаций`;
  }

  function itemHtml(c: Car, i: number): string {
    return `
      <li class="qs__item" role="option" id="qs-opt-${i}" data-id="${escapeHtml(c.id)}" aria-selected="false">
        ${plateHtml(c.plate)}
        ${c.thumbUrl ? `<img class="qs__thumb" src="${escapeHtml(c.thumbUrl)}" alt="" loading="lazy" />` : ""}
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
      st.total = page.total;
      st.nextOffset = page.nextOffset;
      st.loading = false;
      renderFoot();
      // Если номеров мало и прокрутки ещё нет — догружаем, пока не появится или не кончится выдача.
      if ($list.scrollHeight <= $list.clientHeight + 4) void loadMore();
    } catch (e) {
      if (ctrl.signal.aborted || gen !== st.gen) return;
      st.loading = false;
      renderFoot(e instanceof ApiError ? e.message : "Сервер недоступен");
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
    setOpen(false);
    $input.blur();
    h.open(id);
  }

  $input.addEventListener("input", () => {
    clearTimeout(debounce);
    debounce = window.setTimeout(newQuery, DEBOUNCE_MS);
  });
  $input.addEventListener("focus", () => {
    if ($input.value.trim() && $list.children.length) setOpen(true);
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
        setOpen(false);
        h.submit($input.value);
      }
    } else if (e.key === "Escape") {
      // В type=search браузер сам стирает текст по Esc — при открытом списке только закрываем его.
      if (!$panel.hidden) {
        e.preventDefault();
        setOpen(false);
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
    if (!$box.contains(e.target as Node)) setOpen(false);
  });
}
