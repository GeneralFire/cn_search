// Страница номера: все фиксации номера — где и когда его видели.
import { ApiError, plateSightings, seeAlso, type Car } from "./api";
import { mountQuickSearch, type QuickSearchHandlers } from "./quicksearch";
import { plateHref } from "./router";
import { cardHtml, escapeHtml, formatAgo, formatDate, formatShortDate, formatTime, icon, logoHtml, photoAttrs, plateHtml, plural } from "./ui";

let ctrl: AbortController | null = null;

const city = (address: string) => address.split(",")[0]?.trim() ?? "";

/** Рисует страницу номера в root по id любой его фиксации. Шапка с поиском создаётся один раз
 *  и не перерисовывается при переходе между номерами — меняется только содержимое main. */
export async function renderPlatePage(root: HTMLElement, id: string, handlers: QuickSearchHandlers): Promise<void> {
  ctrl?.abort();
  const my = (ctrl = new AbortController());

  if (!root.querySelector(".pp-bar")) {
    root.innerHTML = `
      <header class="pp-bar">
        <a class="pp-bar__home" href="/" data-link aria-label="На главную — поиск по авто">${logoHtml}</a>
        <div class="pp-bar__search"></div>
      </header>
      <main class="pp"></main>`;
    mountQuickSearch(root.querySelector<HTMLElement>(".pp-bar__search")!, handlers);
  }
  const $main = root.querySelector<HTMLElement>(".pp")!;

  $main.innerHTML = `
    <div class="pp-hero pp-hero--loading">
      <div class="skeleton skeleton--plate"></div>
      <div class="skeleton skeleton--line"></div>
    </div>
    <div class="pp-grid">
      <div class="skeleton skeleton--photo"></div>
      <div class="skeleton skeleton--photo"></div>
    </div>`;
  document.title = "Номер — Поиск по номеру";
  window.scrollTo({ top: 0 });

  try {
    const items = await plateSightings(id, my.signal);
    if (my.signal.aborted) return;
    $main.innerHTML = pageHtml(items, id);
    void loadSeeAlso($main, id, my.signal);
    $main.querySelector<HTMLButtonElement>("[data-zoom]")?.addEventListener("click", (e) =>
      openLightbox((e.currentTarget as HTMLElement).dataset.zoom!)
    );
    document.title = `${items[0].plate} — Поиск по номеру`;
  } catch (e) {
    if (my.signal.aborted) return;
    const notFound = e instanceof ApiError && e.status === 404;
    $main.innerHTML = `
      <div class="empty pp-empty">
        <strong>${notFound ? "Фиксация не найдена" : "Не удалось загрузить"}</strong>
        ${notFound ? "Возможно, ссылка устарела." : escapeHtml(e instanceof ApiError ? e.message : "Сервер недоступен")}
      </div>`;
  }
}

function pageHtml(items: Car[], currentId: string): string {
  const latest = items[0];
  const first = items[items.length - 1];
  // Большое фото — открытая фиксация (по ней перешли из выдачи или ленты), иначе последняя.
  const shown = items.find((c) => c.id === currentId) ?? latest;
  const cities = [...new Set(items.map((c) => city(c.address)))];
  const places = new Set(items.map((c) => c.address)).size;

  const stat = (value: string, label: string) =>
    `<div class="pp-stat"><div class="pp-stat__value">${value}</div><div class="pp-stat__label">${label}</div></div>`;

  return `
    <section class="pp-hero">
      ${plateHtml(latest, "xl")}
      <div class="pp-hero__info">
        <div class="pp-hero__meta">
          <span class="pp-chip">${icon.pin}${escapeHtml(cities.join(", "))}</span>
          <span class="pp-chip">${icon.clock}Последний раз ${escapeHtml(formatAgo(latest.createdAt))}</span>
        </div>
        <div class="pp-stats">
          ${stat(String(items.length), plural(items.length, "фиксация", "фиксации", "фиксаций"))}
          ${stat(String(places), plural(places, "место", "места", "мест"))}
          ${stat(escapeHtml(formatShortDate(first.createdAt)), "первая")}
          ${stat(escapeHtml(formatShortDate(latest.createdAt)), "последняя")}
        </div>
      </div>
    </section>

    <div class="pp-grid">
      <figure class="pp-photo">
        ${
          shown.photoUrl
            ? `<button class="pp-photo__zoom" type="button" data-zoom="${escapeHtml(shown.photoUrl)}" aria-label="Увеличить фото">
                 <img src="${escapeHtml(shown.photoUrl)}" alt="Фото фиксации" />
               </button>`
            : icon.car
        }
        <figcaption>
          <span class="pp-photo__tag">${shown === latest ? "Последняя фиксация" : "Фиксация"}</span>
          <span>${escapeHtml(shown.address)}</span>
          <time datetime="${escapeHtml(shown.createdAt)}">${escapeHtml(formatDate(shown.createdAt))}</time>
        </figcaption>
      </figure>

      <section class="pp-history" aria-labelledby="pp-history-title">
        <h2 id="pp-history-title">История фиксаций</h2>
        <ol class="tl">
          ${items.map((c) => timelineItem(c, c.id === currentId)).join("")}
        </ol>
      </section>
    </div>

    <section class="pp-more" aria-labelledby="pp-more-title">
      <h2 id="pp-more-title">Свежие фиксации</h2>
      <div class="grid pp-more__grid"><div class="skeleton skeleton--photo"></div><div class="skeleton skeleton--photo"></div><div class="skeleton skeleton--photo"></div></div>
      <a class="pp-more__home btn btn--ghost" href="/" data-link>← Все номера на главной</a>
    </section>`;
}

function timelineItem(c: Car, current: boolean): string {
  const { src, full } = photoAttrs(c);
  const thumb = src ? `<img src="${escapeHtml(src)}" alt="" loading="lazy" />` : icon.car;
  return `
    <li class="tl__item${current ? " is-current" : ""}">
      <a class="tl__link" href="${plateHref(c.id)}" data-link${current ? ` aria-current="page"` : ""}
         aria-label="Фиксация ${escapeHtml(formatShortDate(c.createdAt))}: ${escapeHtml(c.address)}"></a>
      <a class="tl__thumb" href="${plateHref(c.id)}" data-link tabindex="-1"${full ? ` data-full="${escapeHtml(full)}"` : ""}>${thumb}</a>
      <div class="tl__body">
        <div class="tl__when">
          <time datetime="${escapeHtml(c.createdAt)}">${escapeHtml(formatShortDate(c.createdAt))}, ${escapeHtml(formatTime(c.createdAt))}</time>
          <span class="tl__ago">${escapeHtml(formatAgo(c.createdAt))}</span>
        </div>
        <div class="tl__where">${icon.pin}${escapeHtml(c.address)}</div>
        ${current ? `<span class="tl__badge">Показана слева</span>` : ""}
      </div>
    </li>`;
}

// Лайтбокс: фото на весь экран; закрывается кликом, Esc или кнопкой.
let $lightbox: HTMLDialogElement | null = null;

function openLightbox(src: string): void {
  if (!$lightbox) {
    $lightbox = document.createElement("dialog");
    $lightbox.className = "lightbox";
    $lightbox.innerHTML = `
      <img class="lightbox__img" alt="Фото фиксации" />
      <button class="lightbox__x" type="button" aria-label="Закрыть">${icon.clear}</button>`;
    $lightbox.addEventListener("click", () => $lightbox!.close());
    document.body.append($lightbox);
  }
  $lightbox.querySelector<HTMLImageElement>(".lightbox__img")!.src = src;
  $lightbox.showModal();
}

/** Блок «Свежие фиксации»: до 21 карточки других номеров. */
async function loadSeeAlso($main: HTMLElement, id: string, signal: AbortSignal): Promise<void> {
  const $grid = $main.querySelector<HTMLElement>(".pp-more__grid");
  if (!$grid) return;
  try {
    const items = await seeAlso(id, signal);
    if (signal.aborted) return;
    $grid.innerHTML = items.length
      ? items.map((c, i) => cardHtml(c, i, { clarify: false })).join("")
      : `<div class="empty">Пока нет других фиксаций</div>`;
  } catch {
    if (!signal.aborted) $grid.innerHTML = `<div class="empty">Не удалось загрузить свежие фиксации</div>`;
  }
}
