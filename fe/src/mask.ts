// Поиск по маске: кнопка-переключатель «*» рядом с полем поиска. В режиме маски «*» в запросе —
// ровно один любой символ (бэк: SearchRequest.mask). Режим хранится в адресе (?mask=1 рядом с ?q=,
// фильтрами и ?tab=), как они — через replaceState: F5 и «Назад» его сохраняют.
import "./mask.css";
import { t } from "./i18n/i18n";
import { escapeHtml } from "./ui";

/** Сколько известных символов (не «*») нужно в маске — как models.MaskMinKnown на бэке. */
const MIN_KNOWN = 3;

/** Известные символы маски: после отбрасывания всего, кроме букв, цифр, пробелов и «*» (как на бэке). */
export function knownChars(q: string): number {
  return [...q.replace(/[^\p{L}\p{N}\s*]/gu, "")].filter((c) => c !== "*" && !/\s/.test(c)).length;
}

/** Текст ошибки маски; "" — маска годится (или запрос пуст — тогда это главная, а не поиск). */
export function maskError(q: string): string {
  return q.trim() && knownChars(q) < MIN_KNOWN ? t("mask.tooShort") : "";
}

const maskFromUrl = (): boolean => location.pathname === "/" && new URLSearchParams(location.search).get("mask") === "1";

function setMaskInUrl(on: boolean): void {
  if (location.pathname !== "/") return;
  const url = new URL(location.href);
  if (on) url.searchParams.set("mask", "1");
  else url.searchParams.delete("mask");
  if (url.href !== location.href) history.replaceState(history.state, "", url);
}

export interface MaskToggle {
  /** Включён ли режим маски. */
  on(): boolean;
  /** Включить/выключить режим (обновляет кнопку, подсказку и адрес). */
  set(on: boolean): void;
  /** Подхватить режим из адреса (переход по истории); true — изменился. */
  syncFromUrl(): boolean;
  /** Показать ошибку маски под полем (или подсказку, если её нет) для текущего запроса. */
  refresh(q: string): void;
}

/** Кнопка «Маска» в строке поиска (до значка поиска) и подсказка под ней; onChange — режим переключили. */
export function mountMask(bar: HTMLElement, input: HTMLInputElement, onChange: () => void): MaskToggle {
  const placeholder = input.placeholder;
  const btn = document.createElement("button");
  btn.type = "button";
  btn.className = "mask-btn";
  btn.innerHTML = `<span class="mask-btn__star" aria-hidden="true">*</span><span class="mask-btn__label">${escapeHtml(t("mask.button"))}</span>`;
  bar.insertBefore(btn, bar.querySelector(".searchbar__go"));
  const note = document.createElement("p");
  note.className = "mask-note";
  note.hidden = true;
  bar.after(note);

  let on = maskFromUrl();
  const paint = () => {
    btn.setAttribute("aria-pressed", String(on));
    btn.title = t("mask.hint");
    input.placeholder = on ? t("mask.placeholder") : placeholder;
  };
  const refresh = (q: string) => {
    const err = on ? maskError(q) : "";
    note.hidden = !on;
    note.textContent = err || t("mask.hint");
    note.classList.toggle("is-error", !!err);
    note.setAttribute("role", err ? "alert" : "note");
    input.toggleAttribute("aria-invalid", !!err);
  };
  const api: MaskToggle = {
    on: () => on,
    set(v) {
      on = v;
      paint();
      refresh(input.value);
      setMaskInUrl(on);
    },
    syncFromUrl() {
      const v = maskFromUrl();
      if (v === on) return false;
      api.set(v);
      return true;
    },
    refresh,
  };
  btn.addEventListener("click", () => {
    api.set(!on);
    onChange();
    input.focus();
  });
  paint();
  refresh(input.value);
  return api;
}
