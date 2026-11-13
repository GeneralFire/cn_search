// Вкладки админ-панели /admin: реестр. Каждая фича подключает свою вкладку вызовом registerAdminTab
// (жалобы — admin.ts, уточнения — adminclar.ts; заявки на камеры подключаются так же).
// Порядок вкладок — по order; активная — ?tab=<id> в адресе, по умолчанию первая.
import type { Key } from "./i18n/i18n";

export interface AdminTab {
  id: string;
  label: Key;
  order: number;
  /** Рисует вкладку в main; onDenied() — бэк ответил 403 (токен неверный): вернуться к форме входа. */
  render(main: HTMLElement, onDenied: () => void): void | Promise<void>;
}

export const adminTabs: AdminTab[] = [];

export function registerAdminTab(tab: AdminTab): void {
  adminTabs.push(tab);
  adminTabs.sort((a, b) => a.order - b.order);
}
