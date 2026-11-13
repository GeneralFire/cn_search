// Подвал страниц: текст зависит от языка интерфейса (ключ footer.message); пустой — подвала нет.
import "./footer.css";
import { getLang, t } from "./i18n/i18n";

export function footerHtml(): string {
  const msg = t("footer.message");
  if (!msg) return "";
  return `<footer class="site-foot site-foot--${getLang()}"><p class="site-foot__msg">${msg}</p></footer>`;
}
