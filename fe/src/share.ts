// Кнопка «Поделиться» на странице номера: системное меню (navigator.share), а если его нет —
// копирование ссылки в буфер и короткая подсказка «Ссылка скопирована».
import "./share.css";
import { t, tp } from "./i18n/i18n";

const shareIcon = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M12 3v12M7 8l5-5 5 5"/><path d="M5 13v6a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-6"/></svg>`;

/** Сколько висит подсказка «Ссылка скопирована». */
const TOAST_MS = 2000;

let toastTimer = 0;

/** Кнопка для шапки страницы номера (стиль .btn--ghost). Обработчик вешает bindShare. */
export function shareButtonHtml(): string {
  return `<button class="btn btn--ghost pp-share" type="button" data-share>${shareIcon}<span>${t("share.button")}</span></button>`;
}

/** Вешает обработчик на кнопку внутри root. Номер и число фиксаций — для текста шаринга. */
export function bindShare(root: HTMLElement, plate: string, sightings: number): void {
  root.querySelector<HTMLButtonElement>("[data-share]")?.addEventListener("click", () => void share(plate, sightings));
}

async function share(plate: string, sightings: number): Promise<void> {
  const url = location.href;
  if (typeof navigator.share === "function") {
    try {
      await navigator.share({ title: plate, text: tp("share.text", sightings, { plate }), url });
    } catch {
      // Пользователь закрыл меню — это не ошибка.
    }
    return;
  }
  try {
    await navigator.clipboard.writeText(url);
    toast(t("share.copied"));
  } catch {
    // Буфер недоступен (например, не https) — говорим об этом, ссылка остаётся в адресной строке.
    toast(t("share.copyFailed"));
  }
}

function toast(message: string): void {
  const $el = document.querySelector<HTMLElement>(".share-toast") ?? createToast();
  $el.textContent = message;
  $el.classList.add("is-visible");
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => $el.classList.remove("is-visible"), TOAST_MS);
}

function createToast(): HTMLElement {
  const $el = document.createElement("div");
  $el.className = "share-toast";
  $el.setAttribute("role", "status");
  document.body.append($el);
  return $el;
}
