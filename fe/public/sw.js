// Service worker без зависимостей: оболочка — network-first, миниатюры — cache-first с лимитом, /api/* не кэшируем.
const VERSION = "v1";
const SHELL_CACHE = `cs-shell-${VERSION}`;
const THUMB_CACHE = `cs-thumbs-${VERSION}`;
const THUMB_LIMIT = 200;
const OFFLINE_URL = "/offline.html";
const PRECACHE = [OFFLINE_URL, "/favicon.svg", "/manifest.webmanifest", "/icons/icon-192.png"];

const THUMB_RE = /^\/s3\/(cars\/[^/]*_thumb\.jpg|u\/[^/]*_thumb[^/]*)$/;

self.addEventListener("install", (e) => {
  e.waitUntil(caches.open(SHELL_CACHE).then((c) => c.addAll(PRECACHE)).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (e) => {
  e.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== SHELL_CACHE && k !== THUMB_CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener("fetch", (e) => {
  const req = e.request;
  if (req.method !== "GET") return;
  const url = new URL(req.url);
  if (url.origin !== location.origin) return;
  if (url.pathname.startsWith("/api/")) return; // API — только сеть

  if (THUMB_RE.test(url.pathname)) {
    e.respondWith(thumb(req));
    return;
  }
  if (req.mode === "navigate") {
    e.respondWith(navigation(req));
    return;
  }
  // Остальная статика оболочки (/assets/*, иконки, манифест): сеть, при отказе — кэш.
  if (url.pathname.startsWith("/s3/")) return; // полноразмерные фото не кэшируем
  e.respondWith(networkFirst(req));
});

async function networkFirst(req) {
  const cache = await caches.open(SHELL_CACHE);
  try {
    const res = await fetch(req);
    if (res.ok) cache.put(req, res.clone());
    return res;
  } catch (err) {
    const hit = await cache.match(req);
    if (hit) return hit;
    throw err;
  }
}

async function navigation(req) {
  try {
    return await fetch(req);
  } catch (err) {
    // Без сети: страница «Нет соединения» (данные приложения всё равно живут в /api).
    const cache = await caches.open(SHELL_CACHE);
    return (await cache.match(OFFLINE_URL)) || Response.error();
  }
}

async function thumb(req) {
  const cache = await caches.open(THUMB_CACHE);
  const hit = await cache.match(req);
  if (hit) return hit;
  const res = await fetch(req);
  if (res.ok) {
    await cache.put(req, res.clone());
    trim(cache);
  }
  return res;
}

// Вытеснение старых: keys() отдаёт записи в порядке добавления.
async function trim(cache) {
  const keys = await cache.keys();
  for (let i = 0; i < keys.length - THUMB_LIMIT; i++) await cache.delete(keys[i]);
}
