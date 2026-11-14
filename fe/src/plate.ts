// Страница номера: все фиксации номера — где и когда его видели.
import { ApiError, plateSightings, seeAlso, type Car, type PlateClarification } from "./api";
import { mountQuickSearch, type QuickSearchHandlers } from "./quicksearch";
import { plateHref } from "./router";
import { escapeHtml, fitPlates, formatAgo, formatDate, formatShortDate, formatTime, icon, miniCardHtml, miniSkeletonHtml, photoAttrs, plateHtml } from "./ui";
import { t, tp } from "./i18n/i18n";
import { mountSettings } from "./settings";
import { openLightbox } from "./lightbox";
import { mountPlateMap, plateMapHtml } from "./map";
import { complainButtonHtml } from "./complaint";
import { noPhotoHtml, photoHtml } from "./nophoto";
import { bindClarificationControls, mountPlateAdmin } from "./adminpanels";
import { adminReady } from "./adminmode";
import { openClarify } from "./clarify";
import { clarificationsHtml, clarifiedMarkHtml } from "./clarifications";
import { bindShare, shareButtonHtml } from "./share";
import { bindExport, exportButtonHtml } from "./export";
import { bindRetry, emptyStateHtml, errorStateHtml } from "./states";
import { activityHtml } from "./activity";

/** Сколько карточек показывать в «Свежих фиксациях» (ручка отдаёт не больше). */
const SEE_ALSO_LIMIT = 21;

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
        <div class="pp-bar__left">
          <div class="pp-bar__settings"></div>
        </div>
        <div class="pp-bar__search"></div>
      </header>
      <main class="pp"></main>`;
    mountQuickSearch(root.querySelector<HTMLElement>(".pp-bar__search")!, handlers);
    mountSettings(root.querySelector<HTMLElement>(".pp-bar__settings")!);
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
  document.title = t("page.loadingTitle");
  window.scrollTo({ top: 0 });

  try {
    const { items, clarifications } = await plateSightings(id, my.signal);
    if (my.signal.aborted) return;
    $main.innerHTML = pageHtml(items, id);
    // Уточнения: плашка сразу; после отправки уточнения или смены статуса админом — перечитываются,
    // при входе в админ-режим (adminReady) — перерисовываются с кнопками модерации.
    let list = clarifications;
    const show = () => showClarifications($main, list, items[0].plateType, handlers, () => void reload());
    const reload = async () => {
      try {
        const page = await plateSightings(id, my.signal);
        if (my.signal.aborted) return;
        list = page.clarifications;
        show();
      } catch {
        // Не удалось перечитать — плашка обновится при следующем открытии страницы.
      }
    };
    show();
    void adminReady.then((on) => on && !my.signal.aborted && show());
    // «Уточнить» — к открытой фиксации.
    const shown = items.find((c) => c.id === id) ?? items[0];
    $main.querySelector("[data-pc-clarify]")?.addEventListener("click", () => openClarify(shown, () => void reload()));
    bindShare($main, items[0].plate, items.length);
    bindExport($main, items);
    void loadSeeAlso($main, id, my.signal);
    void mountPlateMap($main, items, id, my.signal);
    void mountPlateAdmin($main, items, id, my.signal); // админ-режим: панель «Управление»
    $main.querySelector<HTMLButtonElement>("[data-zoom]")?.addEventListener("click", (e) =>
      openLightbox(items, id, e.currentTarget as HTMLElement)
    );
    document.title = t("page.plateTitle", { plate: items[0].plate });
  } catch (e) {
    if (my.signal.aborted) return;
    const notFound = e instanceof ApiError && e.status === 404;
    const tokenMsg = tokenErrorHtml(e);
    if (tokenMsg) {
      $main.innerHTML = tokenMsg;
      return;
    }
    if (notFound) {
      $main.innerHTML = `<div class="pp-empty">${emptyStateHtml(t("plate.notFound"), t("plate.notFoundHint"))}</div>`;
      return;
    }
    // Сеть / сервер: блок ошибки с «Повторить» — страница загружается заново.
    $main.innerHTML = `<div class="pp-empty">${errorStateHtml(e)}</div>`;
    bindRetry($main, () => void renderPlatePage(root, id, handlers));
  }
}

// Отказ по API-токену (Get: 429 — исчерпан, 403 — отозван, 401 — недействителен); иначе "".
function tokenErrorHtml(e: unknown): string {
  if (!(e instanceof ApiError)) return "";
  const kind = e.status === 429 ? "exhausted" : e.status === 403 ? "revoked" : e.status === 401 ? "invalid" : "";
  if (!kind) return "";
  return `
      <div class="empty pp-empty pp-token">
        <strong>${t(`tokens.page.${kind}` as const)}</strong>
        ${t(`tokens.page.${kind}Hint` as const)}
      </div>`;
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
          <span class="pp-chip">${icon.clock}${escapeHtml(t("plate.lastSeen", { ago: formatAgo(latest.createdAt) }))}</span>
          <button class="pc-clarify" type="button" data-pc-clarify>${icon.edit}${t("card.clarify")}</button>
        </div>
        <div class="pp-stats">
          ${stat(String(items.length), tp("stat.sightings", items.length))}
          ${stat(String(places), tp("stat.places", places))}
          ${stat(escapeHtml(formatShortDate(first.createdAt)), t("stat.first"))}
          ${stat(escapeHtml(formatShortDate(latest.createdAt)), t("stat.last"))}
        </div>
        ${shareButtonHtml()}${exportButtonHtml()}
      </div>
    </section>

    <div class="pp-grid">
      <div class="pp-aside">
      <figure class="pp-photo">
        ${
          shown.photoUrl
            ? `<button class="pp-photo__zoom" type="button" data-zoom="${escapeHtml(shown.photoUrl)}" aria-label="${t("photo.zoom")}">
                 ${photoHtml(shown.photoUrl, { alt: t("photo.alt"), lazy: false })}
               </button>`
            : noPhotoHtml()
        }
        ${shown.photoUrl ? complainButtonHtml(shown, "photo") : ""}
        <figcaption>
          <span class="pp-photo__tag">${t(shown === latest ? "photo.latest" : "photo.sighting")}</span>
          <span>${escapeHtml(shown.address)}</span>
          <time datetime="${escapeHtml(shown.createdAt)}">${escapeHtml(formatDate(shown.createdAt))}</time>
        </figcaption>
      </figure>
      ${plateMapHtml(items)}
      </div>

      <section class="pp-history" aria-labelledby="pp-history-title">
        <h2 id="pp-history-title">${t("history.title")}</h2>
        <ol class="tl">
          ${items.map((c) => timelineItem(c, c.id === currentId)).join("")}
        </ol>
      </section>
    </div>

    ${activityHtml(items)}

    <section class="pp-more" aria-labelledby="pp-more-title">
      <h2 id="pp-more-title">${t("seeAlso.title")}</h2>
      <div class="pp-more__grid" aria-busy="true">${miniSkeletonHtml.repeat(7)}</div>
      <a class="pp-more__home" href="/" data-link>${t("seeAlso.home")}</a>
    </section>`;
}

function timelineItem(c: Car, current: boolean): string {
  const { src, full } = photoAttrs(c);
  const thumb = photoHtml(src);
  return `
    <li class="tl__item${current ? " is-current" : ""}" data-id="${escapeHtml(c.id)}">
      <a class="tl__link" href="${plateHref(c.id)}" data-link${current ? ` aria-current="page"` : ""}
         aria-label="${escapeHtml(t("history.item", { date: formatShortDate(c.createdAt), address: c.address }))}"></a>
      <a class="tl__thumb" href="${plateHref(c.id)}" data-link tabindex="-1"${full ? ` data-full="${escapeHtml(full)}"` : ""}>${thumb}</a>
      ${src ? complainButtonHtml(c, "thumb") : ""}
      <div class="tl__body">
        <div class="tl__when">
          <time datetime="${escapeHtml(c.createdAt)}">${escapeHtml(formatShortDate(c.createdAt))}, ${escapeHtml(formatTime(c.createdAt))}</time>
          <span class="tl__ago">${escapeHtml(formatAgo(c.createdAt))}</span>
          <span class="tl__pc" data-pc-for="${escapeHtml(c.id)}"></span>
        </div>
        <div class="tl__where">${icon.pin}${escapeHtml(c.address)}</div>
        ${current ? `<span class="tl__badge">${t("history.shownLeft")}</span>` : ""}
      </div>
    </li>`;
}

/** Плашка уточнений под шапкой и значки у фиксаций ленты; ссылка «искать этот номер» — поиск
 *  на главной без перезагрузки (как Enter в поиске шапки). Админ-режим: кнопки «Принять / Отклонить»
 *  у уточнений (adminpanels.ts), после смены статуса — reload. */
function showClarifications(
  $main: HTMLElement, list: PlateClarification[], currentType: string, handlers: QuickSearchHandlers, reload: () => void
): void {
  const html = clarificationsHtml(list, currentType);
  const $old = $main.querySelector(".pc");
  if ($old) $old.outerHTML = html || "<!---->";
  else $main.querySelector(".pp-hero")?.insertAdjacentHTML("afterend", html);
  $main.querySelectorAll<HTMLElement>("[data-pc-for]").forEach((el) => (el.innerHTML = clarifiedMarkHtml(el.dataset.pcFor ?? "", list)));
  const $pc = $main.querySelector<HTMLElement>(".pc");
  if (!$pc) return;
  $pc.querySelectorAll<HTMLAnchorElement>("[data-pc-search]").forEach((a) =>
    a.addEventListener("click", (e) => {
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return; // новая вкладка — по href (/?q=…)
      e.preventDefault();
      handlers.submit(a.dataset.pcSearch ?? "");
    })
  );
  bindClarificationControls($pc, reload);
}

/** Блок «Свежие фиксации»: до 21 компактной карточки других номеров. */
async function loadSeeAlso($main: HTMLElement, id: string, signal: AbortSignal): Promise<void> {
  const $grid = $main.querySelector<HTMLElement>(".pp-more__grid");
  if (!$grid) return;
  $grid.setAttribute("aria-busy", "true");
  $grid.innerHTML = miniSkeletonHtml.repeat(7);
  try {
    const items = (await seeAlso(id, signal)).slice(0, SEE_ALSO_LIMIT);
    if (signal.aborted) return;
    $grid.innerHTML = items.length
      ? items.map((c, i) => miniCardHtml(c, i)).join("")
      : `<p class="pp-more__note">${t("seeAlso.none")}</p>`;
    fitPlates($grid);
  } catch (e) {
    if (signal.aborted) return;
    $grid.innerHTML = errorStateHtml(e, { compact: true, title: t("seeAlso.failed") });
    bindRetry($grid, () => void loadSeeAlso($main, id, signal));
  } finally {
    if (!signal.aborted) $grid.removeAttribute("aria-busy");
  }
}
