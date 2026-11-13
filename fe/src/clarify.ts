// Модалка «Уточнить номер»: формат (сегменты), номер, регион (EU/US), предпросмотр, «Отправить» → статус.
// Открывается с карточек главной и со страницы номера; диалог создаётся один раз при первом открытии.
import { ApiError, clarify } from "./api";
import { escapeHtml, formatDate, icon, plateHtml, type PlateLike } from "./ui";
import { t } from "./i18n/i18n";

/** Что нужно модалке о фиксации. */
export type ClarifyTarget = PlateLike & { id: string; address: string; createdAt: string };

let $dialog: HTMLDialogElement | null = null;
let target: ClarifyTarget | null = null;
let onSent: (() => void) | undefined;

const $body = () => $dialog!.querySelector<HTMLElement>(".modal__body")!;
const $foot = () => $dialog!.querySelector<HTMLElement>(".modal__foot")!;

function dialog(): HTMLDialogElement {
  if ($dialog) return $dialog;
  $dialog = document.createElement("dialog");
  $dialog.className = "modal";
  $dialog.setAttribute("aria-labelledby", "clarify-title");
  $dialog.innerHTML = `
    <form method="dialog" class="modal__inner">
      <header class="modal__head">
        <h3 id="clarify-title">${t("modal.title")}</h3>
        <button class="modal__x" value="close" formnovalidate aria-label="${t("modal.close")}">${icon.clear}</button>
      </header>
      <div class="modal__body"></div>
      <footer class="modal__foot"></footer>
    </form>`;
  // Клик по подложке закрывает модалку.
  $dialog.addEventListener("click", (e) => {
    if (e.target === $dialog) $dialog!.close();
  });
  $dialog.querySelector("form")!.addEventListener("submit", onSubmit);
  document.body.append($dialog);
  return $dialog;
}

/** Открыть модалку уточнения номера фиксации; sent — вызывается после успешной отправки. */
export function openClarify(car: ClarifyTarget, sent?: () => void): void {
  const $d = dialog();
  target = car;
  onSent = sent;
  const $b = $body();
  $b.innerHTML = `
    <div class="modal__car">
      ${plateHtml(car)}
      <div>
        <div class="modal__date">${escapeHtml(formatDate(car.createdAt))}</div>
        <div class="modal__addr">${escapeHtml(car.address)}</div>
      </div>
    </div>
    <fieldset class="clarify__formats">
      <legend class="clarify__label">${t("clarify.format")}</legend>
      ${[["ru", "RU"], ["eu", "EU"], ["us", "US"], ["", t("clarify.formatOther")]]
        .map(
          ([v, label]) => `<label class="seg"><input type="radio" name="ptype" value="${v}"${
            v === (car.plateType || "") ? " checked" : ""
          } /><span>${escapeHtml(label)}</span></label>`
        )
        .join("")}
    </fieldset>
    <label class="clarify">
      <span class="clarify__label">${t("clarify.question")}</span>
      <span class="clarify__row">
        <input class="clarify__input" name="plate" required maxlength="32" autocomplete="off" spellcheck="false"
          placeholder="${escapeHtml(car.plate)}" value="${escapeHtml(car.plate)}" />
        <input class="clarify__input clarify__region" name="region" maxlength="3" autocomplete="off" spellcheck="false"
          value="${escapeHtml(car.plateRegion ?? "")}" aria-label="${t("clarify.regionEu")}" />
      </span>
      <span class="clarify__hint">${t("clarify.hint")}</span>
    </label>
    <div class="clarify__preview"><span class="clarify__label">${t("clarify.preview")}</span><span class="clarify__plate"></span></div>
    <p class="clarify__msg" role="status" hidden></p>`;
  $foot().innerHTML = `
    <button class="btn btn--ghost" value="close" formnovalidate>${t("clarify.cancel")}</button>
    <button class="btn btn--primary" value="send" type="submit">${t("clarify.send")}</button>`;
  $d.showModal();
  const $input = $b.querySelector<HTMLInputElement>(".clarify__input")!;
  const $region = $b.querySelector<HTMLInputElement>(".clarify__region")!;
  const $preview = $b.querySelector<HTMLElement>(".clarify__plate")!;
  // Регион — только для EU (страна) и US (штат); предпросмотр номера — по выбранному формату.
  const sync = () => {
    const type = clarifyType();
    $region.hidden = type !== "eu" && type !== "us";
    $region.placeholder = type === "us" ? "CA" : "D";
    $region.setAttribute("aria-label", t(type === "us" ? "clarify.regionUs" : "clarify.regionEu"));
    const plate = $input.value.trim().toUpperCase() || car.plate;
    $preview.innerHTML = plateHtml({ plate, plateType: type, plateRegion: $region.value.trim().toUpperCase() });
  };
  // Тело пересоздаётся при каждом открытии — обработчики не накапливаются.
  $b.querySelector(".clarify")!.addEventListener("input", sync);
  $b.querySelector(".clarify__formats")!.addEventListener("change", sync);
  sync();
  $input.select();
}

function clarifyType(): string {
  return $body().querySelector<HTMLInputElement>('input[name="ptype"]:checked')?.value ?? "";
}

async function onSubmit(e: Event): Promise<void> {
  const submitter = (e as SubmitEvent).submitter as HTMLButtonElement | null;
  if (submitter?.value !== "send" || !target) return; // «Отмена» и крестик — обычное закрытие
  e.preventDefault();
  const $b = $body();
  const $input = $b.querySelector<HTMLInputElement>(".clarify__input")!;
  const $msg = $b.querySelector<HTMLElement>(".clarify__msg")!;
  submitter.disabled = true;
  submitter.textContent = t("clarify.sending");
  try {
    const type = clarifyType();
    const region = $b.querySelector<HTMLInputElement>(".clarify__region")!.value;
    const c = await clarify(target.id, $input.value, type, type === "eu" || type === "us" ? region : "");
    $msg.hidden = false;
    $msg.className = "clarify__msg is-ok";
    $msg.textContent = t("clarify.thanks", {
      plate: c.plate,
      status: t(c.accepted ? "clarify.accepted" : "clarify.notAccepted"),
    });
    $input.disabled = true;
    $foot().innerHTML = `<button class="btn btn--primary" value="close">${t("clarify.done")}</button>`;
    onSent?.();
  } catch (err) {
    $msg.hidden = false;
    $msg.className = "clarify__msg is-error";
    $msg.textContent = err instanceof ApiError ? err.message : t("clarify.failed");
    submitter.disabled = false;
    submitter.textContent = t("clarify.send");
  }
}
