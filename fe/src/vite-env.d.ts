/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** "true" — запросы к API с заголовком include-test-data (ответы включают тестовые данные). */
  readonly VITE_INCLUDE_TEST_DATA?: string;
  /** Путь админки (скрытый, например "/ops-7f3a2c"); не задан — "/admin". Сама админка всё равно требует admin-token. */
  readonly VITE_ADMIN_PATH?: string;
}
