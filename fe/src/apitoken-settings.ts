// Панель настроек: остаток запросов API-токена (cst_…, cookie cs_token; rpc TokenInfo) и выход из него.
// Вход — общей формой входа по токену (auth.ts): она принимает и пользовательские "cs_…", и API "cst_…".
import "./apitoken-settings.css";
import { ApiError, apiTokenInfo, getApiTokenCookie, setApiTokenCookie } from "./api";
import { t } from "./i18n/i18n";

/** Рисует секцию «API-токен» в $box (внутри панели настроек); без API-токена в cookie — пусто. */
export function mountApiTokenSection($box: HTMLElement): void {
  $box.hidden = !getApiTokenCookie();
  if (!$box.hidden) void showInfo($box);
}

async function showInfo($box: HTMLElement): Promise<void> {
  $box.innerHTML = `
    <div class="st__label">${t("tokens.settingsLabel")}</div>
    <div class="stk-info"><span class="stk-left">…</span>
      <button class="stk-btn stk-btn--ghost" type="button">${t("admin.logout")}</button></div>`;
  $box.querySelector(".stk-btn")!.addEventListener("click", () => {
    setApiTokenCookie("");
    location.reload(); // обновятся панель и страница номера
  });
  const $left = $box.querySelector<HTMLElement>(".stk-left")!;
  try {
    const info = await apiTokenInfo();
    $left.textContent = info.revoked
      ? t("tokens.statusRevoked")
      : t("tokens.settingsLeft", { n: info.requestsLeft, total: info.requestsTotal });
    $left.classList.toggle("is-empty", info.revoked || info.requestsLeft === 0);
  } catch (e) {
    $left.classList.add("is-empty");
    $left.textContent =
      e instanceof ApiError && e.status === 401 ? t("tokens.settingsInvalid") : e instanceof ApiError ? e.message : t("error.serverDown");
  }
}
