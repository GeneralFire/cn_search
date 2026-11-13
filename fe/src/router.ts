// Маршруты на History API: "/" — поиск, "/plate/<id фиксации>" — страница номера,
// "/admin" — админ-панель (жалобы на фото; ссылки в интерфейсе нет — вход по прямому адресу).
// Vite (dev и preview) отдаёт index.html на любой путь, поэтому прямые ссылки работают.

// Аккаунт (auth.ts): "/signup", "/pricing", "/account", "/pay/mock/<id платежа>" — демо-касса.
export type Route =
  | { name: "search" }
  | { name: "plate"; id: string }
  | { name: "admin" }
  | { name: "signup" | "pricing" | "account" }
  | { name: "pay"; id: string };

export function parseRoute(path = location.pathname): Route {
  if (/^\/admin\/?$/.test(path)) return { name: "admin" };
  const acc = path.match(/^\/(signup|pricing|account)\/?$/);
  if (acc) return { name: acc[1] as "signup" | "pricing" | "account" };
  const pay = path.match(/^\/pay\/mock\/([^/]+)\/?$/);
  if (pay) return { name: "pay", id: decodeURIComponent(pay[1]) };
  const m = path.match(/^\/plate\/([^/]+)\/?$/);
  return m ? { name: "plate", id: decodeURIComponent(m[1]) } : { name: "search" };
}

export const plateHref = (id: string) => `/plate/${encodeURIComponent(id)}`;

let listener: (r: Route) => void = () => {};

/** Подписка на смену маршрута; сразу вызывается с текущим. */
export function onRoute(fn: (r: Route) => void): void {
  listener = fn;
  fn(parseRoute());
}

/** Переход без перезагрузки. Со страницы номера на другую страницу номера (другая фиксация в ленте,
 *  номер из поиска в шапке, «свежие фиксации») — замена записи истории: «Назад» ведёт туда, откуда
 *  открыли первую страницу номера (обычно на главную), а не листает просмотренные фиксации. */
export function navigate(path: string): void {
  if (path === location.pathname) return;
  const replace = parseRoute().name === "plate" && parseRoute(path).name === "plate";
  history[replace ? "replaceState" : "pushState"]({ fromApp: true }, "", path);
  listener(parseRoute());
}

addEventListener("popstate", () => listener(parseRoute()));

// Ссылки с data-link открываются без перезагрузки страницы (кроме Ctrl/⌘/Shift-кликов).
document.addEventListener("click", (e) => {
  const a = (e.target as Element).closest<HTMLAnchorElement>("a[data-link]");
  if (!a || e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
  e.preventDefault();
  navigate(a.getAttribute("href")!);
});
