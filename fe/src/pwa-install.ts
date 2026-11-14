// Секция «Установить приложение» в панели настроек: видна, только когда браузер разрешил установку.
import "./pwa-install.css";
import { t } from "./i18n/i18n";
import { canInstall, onInstallChange, promptInstall } from "./pwa";

export function mountInstallSection($box: HTMLElement): void {
  $box.innerHTML = `
    <div class="st__label">${t("pwa.label")}</div>
    <button class="pwa-install__btn" type="button" title="${t("pwa.hint")}">${t("pwa.install")}</button>`;
  $box.querySelector("button")!.addEventListener("click", () => void promptInstall());
  const sync = () => ($box.hidden = !canInstall());
  sync();
  const off = onInstallChange(() => {
    if (!$box.isConnected) return off(); // панель перерисована (смена языка) — старая подписка не нужна
    sync();
  });
}
