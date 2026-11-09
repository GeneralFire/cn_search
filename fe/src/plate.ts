// Страница номера: все фиксации номера — где и когда его видели.
import { ApiError, plateSightings, type Car } from "./api";
import { back } from "./router";
import { escapeHtml, formatAgo, formatDate, formatShortDate, formatTime, icon, plateHtml, plural } from "./ui";

let ctrl: AbortController | null = null;

const city = (address: string) => address.split(",")[0]?.trim() ?? "";

/** Рисует страницу номера в root по id любой его фиксации. */
export async function renderPlatePage(root: HTMLElement, id: string): Promise<void> {
  ctrl?.abort();
  const my = (ctrl = new AbortController());

  root.innerHTML = shell(`
    <div class="pp-hero pp-hero--loading">
      <div class="skeleton skeleton--plate"></div>
      <div class="skeleton skeleton--line"></div>
    </div>
    <div class="pp-grid">
      <div class="skeleton skeleton--photo"></div>
      <div class="skeleton skeleton--photo"></div>
    </div>`);
  bindBack(root);
  document.title = "Номер — Поиск по авто";

  try {
    const items = await plateSightings(id, my.signal);
    if (my.signal.aborted) return;
    root.innerHTML = shell(pageHtml(items, id));
    document.title = `${items[0].plate} — Поиск по авто`;
    root.querySelector(".tl__item.is-current")?.scrollIntoView({ block: "nearest" });
  } catch (e) {
    if (my.signal.aborted) return;
    const notFound = e instanceof ApiError && e.status === 404;
    root.innerHTML = shell(`
      <div class="empty pp-empty">
        <strong>${notFound ? "Фиксация не найдена" : "Не удалось загрузить"}</strong>
        ${notFound ? "Возможно, ссылка устарела." : escapeHtml(e instanceof ApiError ? e.message : "Сервер недоступен")}
      </div>`);
  }
  bindBack(root);
  window.scrollTo({ top: 0 });
}

function shell(body: string): string {
  return `
    <header class="pp-bar">
      <button class="pp-back" type="button">${icon.back}К поиску</button>
    </header>
    <main class="pp">${body}</main>`;
}

function bindBack(root: HTMLElement): void {
  root.querySelector(".pp-back")?.addEventListener("click", back);
}

function pageHtml(items: Car[], currentId: string): string {
  const latest = items[0];
  const first = items[items.length - 1];
  const cities = [...new Set(items.map((c) => city(c.address)))];
  const places = new Set(items.map((c) => c.address)).size;

  const stat = (value: string, label: string) =>
    `<div class="pp-stat"><div class="pp-stat__value">${value}</div><div class="pp-stat__label">${label}</div></div>`;

  return `
    <section class="pp-hero">
      ${plateHtml(latest.plate, "xl")}
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
    </section>

    <div class="pp-grid">
      <figure class="pp-photo">
        ${latest.photoUrl ? `<img src="${escapeHtml(latest.photoUrl)}" alt="Последняя фиксация" />` : icon.car}
        <figcaption>
          <span class="pp-photo__tag">Последняя фиксация</span>
          <span>${escapeHtml(latest.address)}</span>
          <time datetime="${escapeHtml(latest.createdAt)}">${escapeHtml(formatDate(latest.createdAt))}</time>
        </figcaption>
      </figure>

      <section class="pp-history" aria-labelledby="pp-history-title">
        <h2 id="pp-history-title">История фиксаций</h2>
        <ol class="tl">
          ${items.map((c) => timelineItem(c, c.id === currentId)).join("")}
        </ol>
      </section>
    </div>`;
}

function timelineItem(c: Car, current: boolean): string {
  const thumb = c.photoUrl ? `<img src="${escapeHtml(c.photoUrl)}" alt="" loading="lazy" />` : icon.car;
  return `
    <li class="tl__item${current ? " is-current" : ""}">
      <div class="tl__thumb">${thumb}</div>
      <div class="tl__body">
        <div class="tl__when">
          <time datetime="${escapeHtml(c.createdAt)}">${escapeHtml(formatShortDate(c.createdAt))}, ${escapeHtml(formatTime(c.createdAt))}</time>
          <span class="tl__ago">${escapeHtml(formatAgo(c.createdAt))}</span>
        </div>
        <div class="tl__where">${icon.pin}${escapeHtml(c.address)}</div>
        ${current ? `<span class="tl__badge">Вы открыли эту фиксацию</span>` : ""}
      </div>
    </li>`;
}
