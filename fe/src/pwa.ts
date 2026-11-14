// PWA: регистрация service worker (только в production-сборке — в dev он мешает hot-reload Vite)
// и подсказка «Установить приложение» (beforeinstallprompt).

interface InstallPromptEvent extends Event {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

let deferred: InstallPromptEvent | null = null;
const listeners = new Set<() => void>();
const notify = () => listeners.forEach((f) => f());

export function registerPwa(): void {
  window.addEventListener("beforeinstallprompt", (e) => {
    e.preventDefault(); // своя кнопка в настройках вместо мини-баннера
    deferred = e as InstallPromptEvent;
    notify();
  });
  window.addEventListener("appinstalled", () => {
    deferred = null;
    notify();
  });
  if (import.meta.env.PROD && "serviceWorker" in navigator) {
    window.addEventListener("load", () => {
      navigator.serviceWorker.register("/sw.js").catch(() => {});
    });
  }
}

export const canInstall = (): boolean => deferred !== null;

/** Подписка на появление/исчезновение возможности установки; возвращает отписку. */
export function onInstallChange(f: () => void): () => void {
  listeners.add(f);
  return () => listeners.delete(f);
}

export async function promptInstall(): Promise<void> {
  const e = deferred;
  if (!e) return;
  deferred = null;
  notify();
  await e.prompt();
  await e.userChoice;
}
