// Подписка на номер («Следить»): звёздочка на странице номера, кнопка в пустой выдаче поиска и полоса
// «Отслеживаемые» на главной. Подписка — на строку номера: подписаться можно и на номер, которого ещё нет.
// Вошедший пользователь (cookie cs_token) — подписки на бэке (rpc Watch/Unwatch/ListWatches/MarkWatchSeen).
// Гость — в localStorage (номер и время последней увиденной фиксации); при входе (auth.ts перезагружает
// страницу) migrateGuestWatches переносит их на бэк и очищает localStorage.
import "./watch.css";
import {
  API_TOKEN_PREFIX, ApiError, listWatches, markWatchSeen, plateSightings, search, unwatchPlate, watchPlate,
  type Car, type WatchedPlate,
} from "./api";
import { getToken } from "./auth";
import { t } from "./i18n/i18n";
import { plateSearchKey } from "./searchkey";
import { escapeHtml, miniCardHtml, plateHtml } from "./ui";

/** Ключ подписки — как plate_watch_key в БД: ключ поиска без регистра и пробелов. */
export const watchKey = (plate: string): string => plateSearchKey(plate).toLowerCase().replace(/\s/g, "");

// ---------------------------------------------------------------- хранилище гостя

const KEY = "cs_watches";

/** Гостевая подписка. seenAt — время последней фиксации номера, которую гость видел (пусто — ещё не было). */
interface GuestWatch {
  plate: string;
  seenAt: string;
  addedAt: string;
}

function readGuest(): GuestWatch[] {
  try {
    const v: unknown = JSON.parse(localStorage.getItem(KEY) ?? "[]");
    return Array.isArray(v) ? v.filter((w): w is GuestWatch => !!w && typeof w.plate === "string" && w.plate !== "") : [];
  } catch {
    return [];
  }
}

function writeGuest(list: GuestWatch[]): void {
  try {
    if (list.length) localStorage.setItem(KEY, JSON.stringify(list));
    else localStorage.removeItem(KEY);
  } catch {
    // Хранилище недоступно (приватный режим, квота) — подписка гостя не сохранится.
  }
}

/** Вошёл ли пользователь: токен "cst_…" (API-токен) аккаунтом не является. */
const loggedIn = (): boolean => !!getToken() && !getToken().startsWith(API_TOKEN_PREFIX);

const time = (iso: string): number => Date.parse(iso) || 0;

// ---------------------------------------------------------------- перенос гостевых подписок при входе

/** Вошёл, а локально есть подписки гостя — переносит их на бэк и очищает localStorage. Не перенесённые
 *  из-за сбоя остаются до следующей загрузки. */
export async function migrateGuestWatches(): Promise<void> {
  const guest = readGuest();
  if (!loggedIn() || !guest.length) return;
  const rest: GuestWatch[] = [];
  for (const w of guest) {
    try {
      await watchPlate(w.plate);
    } catch (e) {
      if (!(e instanceof ApiError && e.status === 400)) rest.push(w); // 400 — номер не принят, повторять нечего
    }
  }
  writeGuest(rest);
}

// ---------------------------------------------------------------- состояние подписки на номер

async function isWatched(plate: string): Promise<boolean> {
  const key = watchKey(plate);
  if (!loggedIn()) return readGuest().some((w) => watchKey(w.plate) === key);
  return (await listWatches()).some((w) => watchKey(w.plate) === key);
}

/** Подписать/отписать. latest — время последней известной фиксации номера (гость: всё до неё «увидено»). */
async function setWatched(plate: string, on: boolean, latest = ""): Promise<void> {
  const key = watchKey(plate);
  if (loggedIn()) {
    await (on ? watchPlate(plate) : unwatchPlate(plate));
    return;
  }
  const rest = readGuest().filter((w) => watchKey(w.plate) !== key);
  if (on) rest.push({ plate, seenAt: latest, addedAt: new Date().toISOString() });
  writeGuest(rest);
}

// ---------------------------------------------------------------- кнопка на странице номера

const starIcon = (filled: boolean) =>
  `<svg width="14" height="14" viewBox="0 0 24 24" fill="${filled ? "currentColor" : "none"}" stroke="currentColor" stroke-width="2" stroke-linejoin="round" aria-hidden="true"><path d="M12 3l2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1L3.2 9.5l6.1-.9z"/></svg>`;

/** Вставляет кнопку «Следить / Не следить» после «Поделиться» и отмечает номер просмотренным, если на него
 *  есть подписка. items — все фиксации номера, новые первыми. */
export async function mountWatchButton(root: HTMLElement, items: Car[]): Promise<void> {
  const latest = items[0];
  const share = root.querySelector("[data-share]");
  if (!latest || !share) return;
  const btn = document.createElement("button");
  btn.type = "button";
  btn.className = "btn btn--ghost pp-watch";
  btn.hidden = true;
  share.after(btn);

  let on: boolean;
  try {
    on = await isWatched(latest.plate);
    if (on) {
      // Открыл страницу — всё увидел.
      if (loggedIn()) await markWatchSeen(latest.plate);
      else {
        const key = watchKey(latest.plate);
        writeGuest(readGuest().map((w) => (watchKey(w.plate) === key ? { ...w, seenAt: latest.createdAt } : w)));
      }
    }
  } catch {
    return; // подписки недоступны (бэк не отвечает) — страница работает без кнопки
  }
  const paint = () => {
    btn.innerHTML = `${starIcon(on)}<span>${t(on ? "watch.unwatch" : "watch.watch")}</span>`;
    btn.classList.toggle("is-on", on);
    btn.setAttribute("aria-pressed", String(on));
    btn.hidden = false;
  };
  paint();

  btn.addEventListener("click", async () => {
    btn.disabled = true;
    try {
      await setWatched(latest.plate, !on, latest.createdAt);
      on = !on;
      paint();
    } catch {
      // Не вышло (сеть, лимит) — состояние кнопки прежнее, можно нажать ещё раз.
    } finally {
      btn.disabled = false;
    }
  });
}

// ---------------------------------------------------------------- пустая выдача поиска

/** Блок для пустой выдачи: «Номер не найден — подписаться на появление». query — строка поиска.
 *  После вставки в DOM вызвать bindEmptyWatch. */
export function emptyWatchHtml(query: string): string {
  return `<div class="wl-empty" data-wl-empty="${escapeHtml(query)}" hidden></div>`;
}

/** Оживляет блок emptyWatchHtml внутри root: кнопка подписки на текущий запрос (или «вы подписаны»). */
export async function bindEmptyWatch(root: HTMLElement): Promise<void> {
  const box = root.querySelector<HTMLElement>("[data-wl-empty]");
  const query = box?.dataset.wlEmpty?.trim() ?? "";
  if (!box || !watchKey(query).replace(/[^\p{L}\p{N}]/gu, "")) return;
  let on: boolean;
  try {
    on = await isWatched(query);
  } catch {
    return;
  }
  const paint = () => {
    box.hidden = false;
    box.innerHTML = on
      ? `<span class="wl-empty__done">${starIcon(true)}${escapeHtml(t("watch.subscribed", { plate: query }))}</span>`
      : `<button class="btn btn--ghost wl-empty__btn" type="button">${starIcon(false)}<span>${escapeHtml(t("watch.notFoundSubscribe"))}</span></button>`;
  };
  paint();
  box.addEventListener("click", async (e) => {
    const btn = (e.target as HTMLElement).closest<HTMLButtonElement>(".wl-empty__btn");
    if (!btn || on) return;
    btn.disabled = true;
    try {
      await setWatched(query, true);
      on = true;
      paint();
      void refreshWatchStrip();
    } catch {
      btn.disabled = false;
    }
  });
}

// ---------------------------------------------------------------- полоса на главной

/** Подписки гостя: по каждой — поиск номера (rpc Search) и его фиксации (rpc Get); новые — позже seenAt. */
async function guestWatches(signal: AbortSignal): Promise<WatchedPlate[]> {
  const res = await Promise.all(
    readGuest().map(async (w): Promise<WatchedPlate | null> => {
      try {
        const key = watchKey(w.plate);
        // Поиск идёт по началу слов, а гость мог ввести номер без пробела ("A123BC777"): если по строке как есть
        // номер не нашёлся, ищем по первым символам ключа (начало первого слова) и сверяем ключ целиком.
        let found: Car | undefined;
        for (const q of [w.plate, key.slice(0, 5)]) {
          const page = await search({ plate: q, limit: 100 }, signal);
          found = page.items.find((c) => watchKey(c.plate) === key);
          if (found) break;
        }
        if (!found) return { plate: w.plate, car: null, newCount: 0, watchedAt: w.addedAt };
        const { items } = await plateSightings(found.id, signal);
        const seen = time(w.seenAt);
        return { plate: w.plate, car: items[0] ?? found, newCount: items.filter((c) => time(c.createdAt) > seen).length, watchedAt: w.addedAt };
      } catch {
        return null; // сбой — карточку пропускаем, подписка остаётся
      }
    })
  );
  return res.filter((w): w is WatchedPlate => !!w);
}

function itemHtml(w: WatchedPlate, i: number): string {
  const x = `<button class="wl__x" type="button" data-unwatch="${escapeHtml(w.plate)}" aria-label="${escapeHtml(t("watch.unwatch"))}" title="${escapeHtml(t("watch.unwatch"))}">×</button>`;
  if (!w.car) {
    return `<div class="wl__item">
      <div class="mini wl__wait">
        <span class="wl__waitplate">${plateHtml(w.plate)}</span>
        <span class="wl__waittext">${escapeHtml(t("watch.waiting"))}</span>
      </div>${x}</div>`;
  }
  const badge = w.newCount > 0
    ? `<span class="wl__badge" title="${escapeHtml(t("watch.newTitle", { n: w.newCount }))}">${escapeHtml(t("watch.new", { n: w.newCount }))}</span>`
    : "";
  return `<div class="wl__item">${miniCardHtml(w.car, i)}${badge}${x}</div>`;
}

function stripHtml(items: WatchedPlate[]): string {
  return `
    <div class="wl__head">
      <h2 class="wl__title" id="wl-title">${t("watch.title")}</h2>
      <span class="wl__sub">${t("watch.hint")}</span>
    </div>
    <div class="wl__row">${items.map(itemHtml).join("")}</div>`;
}

let refreshStrip: () => Promise<void> = async () => {};

/** Перечитать полосу (после подписки из пустой выдачи, при возврате на главную). */
export const refreshWatchStrip = (): Promise<void> => refreshStrip();

/** Полоса «Отслеживаемые» в root (скрыта, пока подписок нет). Возвращает refresh. */
export function mountWatchStrip(root: HTMLElement): () => Promise<void> {
  root.hidden = true;
  let ctrl: AbortController | null = null;
  root.addEventListener("click", async (e) => {
    const btn = (e.target as HTMLElement).closest<HTMLButtonElement>("[data-unwatch]");
    if (!btn) return;
    btn.disabled = true;
    try {
      await setWatched(btn.dataset.unwatch ?? "", false);
      await refreshStrip();
    } catch {
      btn.disabled = false;
    }
  });
  refreshStrip = async () => {
    ctrl?.abort();
    const my = (ctrl = new AbortController());
    try {
      const items = loggedIn() ? await listWatches(my.signal) : await guestWatches(my.signal);
      if (my.signal.aborted) return;
      root.innerHTML = items.length ? stripHtml(items) : "";
      root.hidden = !items.length;
    } catch {
      // Подписки не загрузились — полосу не показываем (на ленту это не влияет).
      if (!my.signal.aborted) root.hidden = true;
    }
  };
  return refreshStrip;
}
