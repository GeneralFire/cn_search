// Адрес по координатам для заявки на камеру: OpenStreetMap Nominatim, reverse geocoding.
// Правила Nominatim (operations.osmfoundation.org/policies/nominatim): не чаще 1 запроса в секунду,
// кэш результатов, никаких массовых запросов. Поэтому запрос уходит через дебаунс 1 с после последнего
// перемещения точки; новая точка отменяет предыдущий запрос; координаты округляются до 4 знаков
// (≈11 м) и кэшируются в памяти. Ошибки сети тихие: адреса нет — поле остаётся для ручного ввода.
import { intlLocale } from "./i18n/i18n";

const ENDPOINT = "https://nominatim.openstreetmap.org/reverse";
/** Пауза после последнего перемещения точки. */
const DEBOUNCE_MS = 1000;
/** Минимальный интервал между сетевыми запросами (политика Nominatim: не чаще 1 в секунду). */
const MIN_GAP_MS = 1000;
const TIMEOUT_MS = 8000;
const CACHE_MAX = 200;

interface NominatimReverse {
  display_name?: string;
  error?: string;
  address?: { road?: string; house_number?: string; city?: string; town?: string; village?: string };
}

/** Кэш в памяти: ключ — округлённые координаты и язык; null — адреса для точки нет (тоже ответ). */
const cache = new Map<string, string | null>();

const round4 = (n: number) => n.toFixed(4);

function remember(key: string, address: string | null): void {
  if (cache.size >= CACHE_MAX) cache.delete(cache.keys().next().value!);
  cache.set(key, address);
}

/** Укороченный адрес: улица и дом, город. Если этих полей нет — display_name целиком. */
function shortAddress(r: NominatimReverse): string | null {
  const a = r.address ?? {};
  const street = [a.road, a.house_number].filter(Boolean).join(" ");
  const city = a.city || a.town || a.village;
  const parts = [street, city].filter(Boolean) as string[];
  return parts.length ? parts.join(", ") : r.display_name?.trim() || null;
}

async function fetchAddress(lat: string, lon: string, signal: AbortSignal): Promise<string | null> {
  const q = new URLSearchParams({ format: "jsonv2", lat, lon, "accept-language": intlLocale(), zoom: "18" });
  const res = await fetch(`${ENDPOINT}?${q}`, { signal });
  if (!res.ok) throw new Error(`nominatim: ${res.status}`);
  const data = (await res.json()) as NominatimReverse;
  return data.error ? null : shortAddress(data);
}

export interface AddressLookupHandlers {
  onLoading(busy: boolean): void;
  /** null — адреса нет или запрос не удался; вызывается только для актуальной точки. */
  onAddress(address: string | null): void;
}

/** Ленивый поиск адреса для одной «живой» точки: request — после перемещения, cancel — уход/сброс. */
export function createAddressLookup(h: AddressLookupHandlers) {
  let timer: number | undefined;
  let ctl: AbortController | null = null;
  let lastStart = 0;

  const stop = (): void => {
    clearTimeout(timer);
    timer = undefined;
    ctl?.abort();
    ctl = null;
  };

  async function run(key: string, lat: string, lon: string): Promise<void> {
    const c = new AbortController();
    ctl = c;
    const timeout = setTimeout(() => c.abort(), TIMEOUT_MS);
    lastStart = Date.now();
    try {
      const address = await fetchAddress(lat, lon, c.signal);
      remember(key, address);
      if (ctl === c) h.onAddress(address);
    } catch {
      // Тихо: ошибка сети не мешает ручному вводу. Отменённый запрос (ctl уже не c) — молча.
      if (ctl === c) h.onAddress(null);
    } finally {
      clearTimeout(timeout);
      if (ctl === c) {
        ctl = null;
        h.onLoading(false);
      }
    }
  }

  return {
    request(lat: number, lon: number): void {
      stop();
      const la = round4(lat);
      const lo = round4(lon);
      const key = `${intlLocale()}|${la},${lo}`;
      if (cache.has(key)) {
        h.onLoading(false);
        h.onAddress(cache.get(key) ?? null);
        return;
      }
      h.onLoading(true);
      const wait = Math.max(DEBOUNCE_MS, lastStart + MIN_GAP_MS - Date.now());
      timer = window.setTimeout(() => {
        timer = undefined;
        void run(key, la, lo);
      }, wait);
    },
    cancel(): void {
      stop();
      h.onLoading(false);
    },
  };
}
