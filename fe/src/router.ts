// Маршруты на History API: "/" — поиск, "/plate/<id фиксации>" — страница номера.
// Vite (dev и preview) отдаёт index.html на любой путь, поэтому прямые ссылки работают.

export type Route = { name: "search" } | { name: "plate"; id: string };

export function parseRoute(path = location.pathname): Route {
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

export function navigate(path: string): void {
  if (path === location.pathname) return;
  history.pushState({ fromApp: true }, "", path);
  listener(parseRoute());
}

/** Назад к поиску: если пришли из приложения — history.back() (вернутся запрос и прокрутка). */
export function back(): void {
  if (history.state?.fromApp) history.back();
  else navigate("/");
}

addEventListener("popstate", () => listener(parseRoute()));

// Ссылки с data-link открываются без перезагрузки страницы (кроме Ctrl/⌘/Shift-кликов).
document.addEventListener("click", (e) => {
  const a = (e.target as Element).closest<HTMLAnchorElement>("a[data-link]");
  if (!a || e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
  e.preventDefault();
  navigate(a.getAttribute("href")!);
});
