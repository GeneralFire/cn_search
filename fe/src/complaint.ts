// Жалоба на фото фиксации: кнопка «Пожаловаться» (на большом фото страницы номера и значком на миниатюрах
// ленты истории) открывает модалку — причина (сегменты), комментарий, «Отправить» → «Спасибо».
// Кнопки ловятся одним обработчиком на document, данные фиксации — в data-complain (JSON).
import "./complaint.css";
import { ApiError, complain, COMPLAINT_REASONS, type ComplaintReason } from "./api";
import { escapeHtml, formatDate, icon, plateHtml, type PlateLike } from "./ui";
import { t } from "./i18n/i18n";

const MAX_COMMENT = 500; // как в БД (CHECK) и на бэке

/** Что нужно модалке о фиксации. */
export type ComplaintTarget = PlateLike & { id: string; address: string; createdAt: string };

const flagIcon = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M5 21V4m0 0h11l-2 4 2 4H5"/></svg>`;

/** Кнопка «Пожаловаться»: "photo" — с подписью в углу большого фото, "thumb" — значок на миниатюре. */
export function complainButtonHtml(c: ComplaintTarget, variant: "photo" | "thumb"): string {
  const data: ComplaintTarget = {
    id: c.id, plate: c.plate, plateType: c.plateType, plateRegion: c.plateRegion, address: c.address, createdAt: c.createdAt,
  };
  return `<button class="cm-btn cm-btn--${variant}" type="button" data-complain="${escapeHtml(JSON.stringify(data))}"
    title="${t("complaint.title")}" aria-label="${t("complaint.title")}">${flagIcon}${
    variant === "photo" ? `<span>${t("complaint.button")}</span>` : ""
  }</button>`;
}

let $dialog: HTMLDialogElement | null = null;
let target: ComplaintTarget | null = null;

function dialog(): HTMLDialogElement {
  if ($dialog) return $dialog;
  $dialog = document.createElement("dialog");
  $dialog.className = "modal cm";
  $dialog.setAttribute("aria-labelledby", "cm-title");
  $dialog.innerHTML = `
    <form method="dialog" class="modal__inner">
      <header class="modal__head">
        <h3 id="cm-title">${t("complaint.title")}</h3>
        <button class="modal__x" value="close" formnovalidate aria-label="${t("modal.close")}">${icon.clear}</button>
      </header>
      <div class="cm__body"></div>
      <footer class="modal__foot"></footer>
    </form>`;
  // Клик по подложке закрывает.
  $dialog.addEventListener("click", (e) => {
    if (e.target === $dialog) $dialog!.close();
  });
  $dialog.querySelector("form")!.addEventListener("submit", onSubmit);
  document.body.append($dialog);
  return $dialog;
}

/** Открыть модалку жалобы на фото фиксации. */
export function openComplaint(c: ComplaintTarget): void {
  const $d = dialog();
  target = c;
  $d.querySelector<HTMLElement>(".cm__body")!.innerHTML = `
    <div class="modal__car">
      ${plateHtml(c)}
      <div>
        <div class="modal__date">${escapeHtml(formatDate(c.createdAt))}</div>
        <div class="modal__addr">${escapeHtml(c.address)}</div>
      </div>
    </div>
    <fieldset class="cm__reasons">
      <legend class="cm__label">${t("complaint.reason")}</legend>
      ${COMPLAINT_REASONS.map(
        (r) => `<label class="seg"><input type="radio" name="reason" value="${r}" required /><span>${t(`complaint.reason.${r}`)}</span></label>`
      ).join("")}
    </fieldset>
    <label class="cm__field">
      <span class="cm__label">${t("complaint.comment")}</span>
      <textarea class="cm__comment" name="comment" rows="3" maxlength="${MAX_COMMENT}"
        placeholder="${t("complaint.commentPlaceholder")}"></textarea>
      <span class="cm__count" aria-hidden="true">0 / ${MAX_COMMENT}</span>
    </label>
    <p class="cm__msg" role="status" hidden></p>`;
  footer(false);
  const $comment = $d.querySelector<HTMLTextAreaElement>(".cm__comment")!;
  const $count = $d.querySelector<HTMLElement>(".cm__count")!;
  $comment.addEventListener("input", () => ($count.textContent = `${$comment.value.length} / ${MAX_COMMENT}`));
  $d.showModal();
  $d.querySelector<HTMLInputElement>('input[name="reason"]')!.focus();
}

function footer(done: boolean): void {
  $dialog!.querySelector<HTMLElement>(".modal__foot")!.innerHTML = done
    ? `<button class="btn btn--primary" value="close">${t("complaint.done")}</button>`
    : `<button class="btn btn--ghost" value="close" formnovalidate>${t("complaint.cancel")}</button>
       <button class="btn btn--primary" value="send" type="submit">${t("complaint.send")}</button>`;
}

async function onSubmit(e: SubmitEvent): Promise<void> {
  const submitter = e.submitter as HTMLButtonElement | null;
  if (submitter?.value !== "send" || !target) return; // «Отмена», крестик, «Готово» — обычное закрытие
  e.preventDefault();
  const $d = $dialog!;
  const $msg = $d.querySelector<HTMLElement>(".cm__msg")!;
  const reason = $d.querySelector<HTMLInputElement>('input[name="reason"]:checked')?.value as ComplaintReason | undefined;
  if (!reason) {
    showMsg($msg, "is-error", t("complaint.needReason"));
    return;
  }
  submitter.disabled = true;
  submitter.textContent = t("complaint.sending");
  try {
    await complain(target.id, reason, $d.querySelector<HTMLTextAreaElement>(".cm__comment")!.value.trim());
    $d.querySelector<HTMLElement>(".cm__body")!.innerHTML = `
      <div class="cm__thanks" role="status">
        <span class="cm__check" aria-hidden="true">✓</span>
        <strong>${t("complaint.thanks")}</strong>
        <span>${t("complaint.thanksHint")}</span>
      </div>`;
    footer(true);
    $d.querySelector<HTMLButtonElement>(".modal__foot .btn")!.focus();
  } catch (err) {
    showMsg($msg, "is-error", err instanceof ApiError ? err.message : t("complaint.failed"));
    submitter.disabled = false;
    submitter.textContent = t("complaint.send");
  }
}

function showMsg($msg: HTMLElement, cls: string, text: string): void {
  $msg.hidden = false;
  $msg.className = `cm__msg ${cls}`;
  $msg.textContent = text;
}

// Любая кнопка data-complain на странице открывает модалку (страница номера перерисовывается —
// поэтому обработчик один, на document). preventDefault — кнопка лежит рядом со ссылками и над фото.
document.addEventListener("click", (e) => {
  const b = (e.target as Element).closest<HTMLButtonElement>("[data-complain]");
  if (!b) return;
  e.preventDefault();
  e.stopPropagation();
  openComplaint(JSON.parse(b.dataset.complain!) as ComplaintTarget);
});
