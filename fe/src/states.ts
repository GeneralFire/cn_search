// Единые состояния блоков с данными: пусто / ошибка / офлайн / загрузка (скелетон).
// Ошибка — с кратким понятным описанием (без «rpc error: code = …») и кнопкой «Повторить»;
// офлайн — повтор сам по событию online. Разметка — строкой (вставляется в innerHTML),
// повтор подключает bindRetry после вставки.
import "./states.css";
import { ApiError } from "./api";
import { t } from "./i18n/i18n";
import { escapeHtml } from "./ui";

/** Вид ошибки для пользователя. */
export type ErrorKind = "offline" | "network" | "server" | "exhausted";

/** Убрать технический префикс gRPC: "rpc error: code = X desc = текст" → "текст". */
export function cleanMessage(msg: string): string {
  return msg.replace(/^rpc error: code = \w+ desc = /, "").trim();
}

/** Вид ошибки: нет сети в браузере / бэк не отвечает / лимит (429, ResourceExhausted) / прочее. */
export function errorKind(e: unknown): ErrorKind {
  if (typeof navigator !== "undefined" && !navigator.onLine) return "offline";
  if (e instanceof ApiError) {
    if (e.status === 429 || /code = ResourceExhausted/.test(e.message)) return "exhausted";
    // Нет ответа (fetch упал), прокси вернул 502/503/504 или 5xx без JSON-тела clay (так прокси vite
    // отвечает при остановленном бэке; errorMessage тогда даёт «HTTP 500») — бэк недоступен.
    if (e.status === 0 || e.status === 502 || e.status === 503 || e.status === 504) return "network";
    if (e.status >= 500 && /^HTTP \d+$/.test(e.message)) return "network";
    return "server";
  }
  return "network";
}

/** Короткий текст под заголовком: для «server» — desc от бэка (если он не технический), иначе общий. */
export function errorText(e: unknown, kind = errorKind(e)): string {
  if (kind !== "server") return t(`states.${kind}Hint` as const);
  const msg = e instanceof ApiError ? cleanMessage(e.message) : "";
  // Пустой, «HTTP 500» или всё ещё технический текст — общий.
  return msg && !/^HTTP \d+$/.test(msg) && !/^rpc error/.test(msg) ? msg : t("states.serverHint");
}

/** Одной строкой: «Сервер недоступен. Проверьте…» — для мест, где блока нет (подвал быстрого поиска, админка). */
export function errorLine(e: unknown): string {
  const kind = errorKind(e);
  return `${t(`states.${kind}` as const)}. ${errorText(e, kind)}`;
}

const ICONS = {
  error: `<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="M12 7v6M12 16.5v.5"/></svg>`,
  offline: `<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M2 8.5a15 15 0 0 1 20 0M5 12a10 10 0 0 1 14 0M8.5 15.5a5 5 0 0 1 7 0"/><circle cx="12" cy="19" r="1"/><path d="M3 3l18 18"/></svg>`,
  empty: `<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5M8 11h6"/></svg>`,
};

/** Блок «ошибка» (span — можно вставлять и в <p>): иконка, заголовок, описание, «Повторить» (data-st-retry). compact — без рамки, меньше отступы. */
export function errorStateHtml(e: unknown, opts: { compact?: boolean; title?: string } = {}): string {
  const kind = errorKind(e);
  const title = opts.title ?? t(`states.${kind}` as const);
  return `
    <span class="ustate ustate--error ustate--${kind}${opts.compact ? " ustate--compact" : ""}" role="alert" data-st-kind="${kind}">
      <span class="ustate__icon">${kind === "offline" ? ICONS.offline : ICONS.error}</span>
      <strong class="ustate__title">${escapeHtml(title)}</strong>
      <span class="ustate__text">${escapeHtml(errorText(e, kind))}</span>
      <button class="btn btn--ghost ustate__retry" type="button" data-st-retry>${t("states.retry")}</button>
    </span>`;
}

/** Блок «пусто»: заголовок, подсказка (HTML) и необязательный хвост (например, «возможно, вы искали»). */
export function emptyStateHtml(title: string, hintHtml = "", extraHtml = "", compact = false): string {
  return `
    <div class="ustate ustate--empty${compact ? " ustate--compact" : ""}">
      <span class="ustate__icon">${ICONS.empty}</span>
      <strong class="ustate__title">${escapeHtml(title)}</strong>
      ${hintHtml ? `<span class="ustate__text">${hintHtml}</span>` : ""}
      ${extraHtml}
    </div>`;
}

/** Скелетон сетки карточек (.grid) на время первой загрузки. */
export function cardSkeletonHtml(n: number): string {
  return `<div class="ustate-sk" aria-hidden="true"><div class="skeleton ustate-sk__photo"></div><div class="skeleton ustate-sk__line"></div></div>`.repeat(n);
}

/** Подключить повтор к блоку ошибки внутри root: кнопка «Повторить» и автоповтор при возврате сети.
 *  Автоповтор срабатывает один раз и только пока блок ошибки ещё в документе. */
export function bindRetry(root: ParentNode, retry: () => void): void {
  const box = root.querySelector<HTMLElement>(".ustate--error");
  if (!box) return;
  let done = false;
  const run = () => {
    if (done) return;
    done = true;
    window.removeEventListener("online", onOnline);
    retry();
  };
  const onOnline = () => {
    if (box.isConnected) run();
    else window.removeEventListener("online", onOnline);
  };
  box.querySelector("[data-st-retry]")?.addEventListener("click", run);
  window.addEventListener("online", onOnline);
}

/** Показать ошибку в строке-заметке (adm-note в админке): компактный блок с «Повторить». */
export function showErrorIn(el: HTMLElement, e: unknown, retry: () => void): void {
  el.hidden = false;
  el.className = "adm-note";
  el.innerHTML = errorStateHtml(e, { compact: true });
  bindRetry(el, retry);
}
