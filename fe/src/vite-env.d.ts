/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** "true" — запросы к API с заголовком include-test-data (ответы включают тестовые данные). */
  readonly VITE_INCLUDE_TEST_DATA?: string;
}
