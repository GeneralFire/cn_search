// Заглушка «Нет фото»: силуэт машины и подпись — когда у фиксации нет фото или картинка не загрузилась.
// Одна разметка для всех мест (карточки, мини-карточки, большое фото, лента, поиск, админка);
// размер подстраивается под контейнер (container queries в nophoto.css).
import "./nophoto.css";
import { t } from "./i18n/i18n";

const carSvg = `<svg class="nophoto__car" viewBox="0 0 64 32" fill="currentColor" aria-hidden="true"><path d="M10 22h-4a2 2 0 0 1-2-2v-4c0-1.5 1-2.6 2.4-3l6.6-1.8 6-5.4A6 6 0 0 1 23 4.5h14a6 6 0 0 1 4.3 1.8L47 12l9 1.6c1.8.3 3 1.9 3 3.6V20a2 2 0 0 1-2 2h-3"/><circle cx="16" cy="23" r="5"/><circle cx="48" cy="23" r="5"/></svg>`;

const esc = (s: string) =>
  s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!);

/** Заглушка; заполняет ближайший позиционированный контейнер (position: absolute; inset: 0). */
export function noPhotoHtml(): string {
  const label = t("noPhoto.label");
  return `<span class="nophoto" role="img" aria-label="${label}">${carSvg}<span class="nophoto__label" aria-hidden="true">${label}</span></span>`;
}

/** Картинка фото с подменой на заглушку при ошибке загрузки; без src — сразу заглушка. */
export function photoHtml(src: string, opts: { alt?: string; cls?: string; lazy?: boolean } = {}): string {
  if (!src) return noPhotoHtml();
  const cls = opts.cls ? ` class="${opts.cls}"` : "";
  const lazy = opts.lazy === false ? "" : ` loading="lazy"`;
  return `<img${cls} src="${esc(src)}" alt="${opts.alt ?? ""}"${lazy} data-photo />`;
}

/** Битая картинка → заглушка. Событие error не всплывает, поэтому слушаем на захвате. */
function onPhotoError(e: Event): void {
  const img = e.target;
  if (!(img instanceof HTMLImageElement) || !img.hasAttribute("data-photo")) return;
  // Большое фото страницы номера: вместе с картинкой уходят кнопки увеличения и жалобы на фото.
  const zoom = img.closest(".pp-photo__zoom");
  const scope = img.closest(".pp-photo, .tl__item");
  scope?.querySelector(".cm-btn")?.remove();
  // Превью по наведению показывать нечего.
  img.closest("[data-full]")?.removeAttribute("data-full");
  (zoom ?? img).outerHTML = noPhotoHtml();
}

document.addEventListener("error", onPhotoError, true);
