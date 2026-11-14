// Страница 404: показывается для любого неизвестного адреса (router.ts: маршрут "notfound") вместо главной.
import "./notfound.css";
import { t } from "./i18n/i18n";

/** Рисует страницу 404 в root; заголовок вкладки — название приложения. */
export function renderNotFoundPage(root: HTMLElement): void {
  document.title = `404 — ${t("app.title")}`;
  window.scrollTo({ top: 0 });
  root.innerHTML = `
    <section class="nf">
      <p class="nf__code" aria-hidden="true">404</p>
      <h1 class="nf__title">${t("notFound.title")}</h1>
      <p class="nf__text">${t("notFound.text")}</p>
      <a class="nf__home" href="/" data-link>${t("notFound.home")}</a>
    </section>`;
}
