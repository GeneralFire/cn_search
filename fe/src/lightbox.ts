// Лайтбокс-галерея: все фото номера на весь экран. Стрелки ←/→ (кнопки и клавиатура), свайп на таче,
// зум колесом / щипком / двойным кликом с панорамированием, подпись «адрес · дата · 3 / 7»,
// предзагрузка соседних фото, ссылка на фиксацию. Esc закрывает, фокус возвращается на открывшую кнопку.
import "./lightbox.css";
import { plateHref } from "./router";
import { formatDate, icon } from "./ui";
import { t } from "./i18n/i18n";

export interface LightboxItem {
  id: string;
  address: string;
  createdAt: string;
  photoUrl?: string;
}

const MAX_SCALE = 5;
/** Масштаб по двойному клику / двойному тапу. */
const DBL_SCALE = 2.5;
/** Сдвиг пальца (px), после которого свайп листает. */
const SWIPE_PX = 60;
/** Окно двойного тапа (мс) и допуск дрожания пальца (px). */
const TAP_MS = 300;
const TAP_SLOP = 10;

const arrow = (d: string) =>
  `<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" aria-hidden="true"><path d="${d}"/></svg>`;

let $dlg: HTMLDialogElement | null = null;
let $stage: HTMLElement;
let $img: HTMLImageElement;
let $prev: HTMLButtonElement;
let $next: HTMLButtonElement;
let $place: HTMLElement;
let $when: HTMLTimeElement;
let $count: HTMLElement;
let $open: HTMLAnchorElement;

let items: LightboxItem[] = [];
let index = 0;
let currentId = "";
let opener: HTMLElement | null = null;

// Зум и сдвиг картинки: transform = translate(tx, ty) scale(scale) от центра картинки.
let scale = 1;
let tx = 0;
let ty = 0;

/** Открыть галерею по фото фиксаций (без фото пропускаются) начиная с фиксации startId. */
export function openLightbox(all: LightboxItem[], startId: string, from?: HTMLElement | null): void {
  items = all.filter((c) => c.photoUrl);
  if (!items.length) return;
  build();
  currentId = startId;
  opener = from ?? (document.activeElement as HTMLElement | null);
  show(Math.max(0, items.findIndex((c) => c.id === startId)));
  $dlg!.showModal();
}

function build(): void {
  if ($dlg) return;
  $dlg = document.createElement("dialog");
  $dlg.className = "lb";
  $dlg.setAttribute("aria-label", t("lightbox.title"));
  $dlg.innerHTML = `
    <div class="lb__stage">
      <img class="lb__img" alt="${t("photo.alt")}" draggable="false" />
    </div>
    <button class="lb__nav lb__nav--prev" type="button" aria-label="${t("lightbox.prev")}">${arrow("M15 18l-6-6 6-6")}</button>
    <button class="lb__nav lb__nav--next" type="button" aria-label="${t("lightbox.next")}">${arrow("M9 18l6-6-6-6")}</button>
    <div class="lb__bar">
      <div class="lb__caption">
        <span class="lb__place">${icon.pin}<span></span></span>
        <time class="lb__when"></time>
      </div>
      <span class="lb__count" aria-live="polite"></span>
      <a class="lb__open" data-link></a>
    </div>
    <button class="lb__x" type="button" aria-label="${t("modal.close")}" autofocus>${icon.clear}</button>`;
  document.body.append($dlg);

  const q = <T extends Element>(s: string) => $dlg!.querySelector<T>(s)!;
  $stage = q(".lb__stage");
  $img = q(".lb__img");
  $prev = q(".lb__nav--prev");
  $next = q(".lb__nav--next");
  $place = q(".lb__place span");
  $when = q(".lb__when");
  $count = q(".lb__count");
  $open = q(".lb__open");

  $prev.addEventListener("click", () => go(-1));
  $next.addEventListener("click", () => go(1));
  q(".lb__x").addEventListener("click", close);
  // Переход по ссылке делает роутер (data-link); галерею просто закрываем.
  $open.addEventListener("click", () => $dlg!.close());
  $img.addEventListener("load", () => $img.classList.remove("is-loading"));
  $img.addEventListener("error", () => $img.classList.remove("is-loading"));

  $dlg.addEventListener("keydown", onKey);
  $dlg.addEventListener("close", () => {
    if (opener?.isConnected) opener.focus();
    opener = null;
  });
  // «Назад» в браузере уводит со страницы — галерею не оставляем висеть над новой.
  addEventListener("popstate", close);

  $stage.addEventListener("wheel", onWheel, { passive: false });
  $stage.addEventListener("pointerdown", onDown);
  $stage.addEventListener("pointermove", onMove);
  $stage.addEventListener("pointerup", onUp);
  $stage.addEventListener("pointercancel", onUp);
}

function close(): void {
  if ($dlg?.open) $dlg.close();
}

function go(step: number): void {
  if (items.length < 2) return;
  show((index + step + items.length) % items.length);
}

function show(i: number): void {
  index = i;
  const c = items[i];
  resetZoom(false);
  $img.classList.add("is-loading");
  $img.src = c.photoUrl!;
  if ($img.complete) $img.classList.remove("is-loading");
  $place.textContent = c.address;
  $when.dateTime = c.createdAt;
  $when.textContent = formatDate(c.createdAt);
  $count.textContent = t("lightbox.counter", { n: i + 1, total: items.length });
  const here = c.id === currentId;
  $open.href = plateHref(c.id);
  $open.textContent = t(here ? "lightbox.here" : "lightbox.open");
  $open.classList.toggle("is-here", here);
  if (here) $open.setAttribute("aria-current", "page");
  else $open.removeAttribute("aria-current");
  const single = items.length < 2;
  $prev.hidden = single;
  $next.hidden = single;
  $count.hidden = single;
  // Предзагрузка соседних полных фото — листание без ожидания.
  for (const d of [1, -1]) {
    const n = items[(i + d + items.length) % items.length];
    if (n !== c) new Image().src = n.photoUrl!;
  }
}

function onKey(e: KeyboardEvent): void {
  if (e.altKey || e.ctrlKey || e.metaKey) return;
  const r = $stage.getBoundingClientRect();
  const cx = r.left + r.width / 2;
  const cy = r.top + r.height / 2;
  switch (e.key) {
    case "ArrowLeft":
      go(-1);
      break;
    case "ArrowRight":
      go(1);
      break;
    case "+":
    case "=":
      zoomAt(scale * 1.5, cx, cy, true);
      break;
    case "-":
      zoomAt(scale / 1.5, cx, cy, true);
      break;
    case "0":
      resetZoom(true);
      break;
    default:
      return;
  }
  e.preventDefault();
}

// ---------- Зум и панорамирование ----------

function apply(animate: boolean): void {
  $img.classList.toggle("is-anim", animate);
  $img.style.transform = scale === 1 && !tx && !ty ? "" : `translate(${tx}px, ${ty}px) scale(${scale})`;
  $img.classList.toggle("is-zoomed", scale > 1);
}

function resetZoom(animate: boolean): void {
  scale = 1;
  tx = ty = 0;
  apply(animate);
}

/** Не даём увести картинку за край: сдвиг не больше «лишнего» размера увеличенной картинки. */
function clamp(): void {
  const mx = ($img.offsetWidth * (scale - 1)) / 2;
  const my = ($img.offsetHeight * (scale - 1)) / 2;
  tx = Math.max(-mx, Math.min(mx, tx));
  ty = Math.max(-my, Math.min(my, ty));
}

/** Масштаб s с неподвижной точкой экрана (px, py). */
function zoomAt(s: number, px: number, py: number, animate = false): void {
  s = Math.max(1, Math.min(MAX_SCALE, s));
  const r = $stage.getBoundingClientRect();
  // Центр картинки без трансформации = центр сцены (картинка в ней по центру).
  const cx = r.left + r.width / 2;
  const cy = r.top + r.height / 2;
  tx = px - cx - ((px - cx - tx) * s) / scale;
  ty = py - cy - ((py - cy - ty) * s) / scale;
  scale = s;
  if (scale === 1) tx = ty = 0;
  clamp();
  apply(animate);
}

function toggleZoom(px: number, py: number): void {
  if (scale > 1) resetZoom(true);
  else zoomAt(DBL_SCALE, px, py, true);
}

function onWheel(e: WheelEvent): void {
  e.preventDefault();
  // Щипок на тачпаде приходит колесом с ctrlKey и мелкими deltaY — для него множитель больше.
  const k = e.ctrlKey ? 0.01 : 0.002;
  const dy = e.deltaMode === 1 ? e.deltaY * 16 : e.deltaY;
  zoomAt(scale * Math.exp(-dy * k), e.clientX, e.clientY);
}

// ---------- Жесты: перетаскивание, свайп, щипок, двойной тап ----------

type Pt = { x: number; y: number };
const pointers = new Map<number, Pt>();
let start: Pt & { t: number; tx: number; ty: number; onImg: boolean } = { x: 0, y: 0, t: 0, tx: 0, ty: 0, onImg: false };
let pinch: { d: number; s: number } | null = null;
let moved = false;
let lastTap = { t: 0, x: 0, y: 0 };

const dist = (a: Pt, b: Pt) => Math.hypot(a.x - b.x, a.y - b.y);
const mid = (a: Pt, b: Pt): Pt => ({ x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 });

function onDown(e: PointerEvent): void {
  if (e.pointerType === "mouse" && e.button !== 0) return;
  $stage.setPointerCapture(e.pointerId);
  pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
  if (pointers.size === 1) {
    start = { x: e.clientX, y: e.clientY, t: performance.now(), tx, ty, onImg: e.target === $img };
    moved = false;
  } else if (pointers.size === 2) {
    const [a, b] = [...pointers.values()];
    pinch = { d: dist(a, b), s: scale };
    moved = true;
  }
}

function onMove(e: PointerEvent): void {
  const p = pointers.get(e.pointerId);
  if (!p) return;
  p.x = e.clientX;
  p.y = e.clientY;

  if (pinch && pointers.size >= 2) {
    const [a, b] = [...pointers.values()];
    const m = mid(a, b);
    zoomAt((pinch.s * dist(a, b)) / pinch.d, m.x, m.y);
    return;
  }
  if (pointers.size !== 1) return;
  const dx = e.clientX - start.x;
  const dy = e.clientY - start.y;
  if (!moved && Math.hypot(dx, dy) < TAP_SLOP) return;
  moved = true;
  $stage.classList.add("is-dragging");
  if (scale > 1) {
    tx = start.tx + dx;
    ty = start.ty + dy;
    clamp();
    apply(false);
  } else if (items.length > 1) {
    // Без зума картинка едет за пальцем по горизонтали — подсказка, что можно листать.
    tx = dx;
    apply(false);
  }
}

function onUp(e: PointerEvent): void {
  if (!pointers.delete(e.pointerId)) return;
  if (pointers.size > 0) {
    // Отпустили один палец из щипка — оставшийся продолжает тащить с текущего места.
    if (pinch && pointers.size === 1) {
      const [p] = [...pointers.values()];
      start = { x: p.x, y: p.y, t: performance.now(), tx, ty, onImg: true };
    }
    pinch = null;
    return;
  }
  pinch = null;
  $stage.classList.remove("is-dragging");
  const dx = e.clientX - start.x;

  if (!moved && e.type === "pointerup") {
    onTap(e);
    return;
  }
  if (scale === 1 && e.type === "pointerup" && Math.abs(dx) > SWIPE_PX && items.length > 1) {
    go(dx < 0 ? 1 : -1);
    return;
  }
  if (scale === 1) {
    tx = ty = 0;
    apply(true); // вернуть на место недотянутый свайп
  }
}

function onTap(e: PointerEvent): void {
  const now = performance.now();
  if (!start.onImg) {
    // Тап по пустому месту вокруг фото закрывает, как и раньше.
    close();
    return;
  }
  // Двойной клик / двойной тап — сами: при захвате указателя браузер dblclick не присылает.
  const dbl = now - lastTap.t < TAP_MS && Math.hypot(e.clientX - lastTap.x, e.clientY - lastTap.y) < TAP_SLOP * 3;
  lastTap = dbl ? { t: 0, x: 0, y: 0 } : { t: now, x: e.clientX, y: e.clientY };
  if (dbl) toggleZoom(e.clientX, e.clientY);
}
