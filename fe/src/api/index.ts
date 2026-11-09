// Обёртка над сгенерированным клиентом (generated.ts — не править руками, см. `make gen-ts`).
import {
  CarSearchApi,
  CarsearchPlateType,
  type CarsearchGetResponse,
  type CarsearchPlateSighting,
  type CarsearchSearchRequest,
  type CarsearchSearchResponse,
} from "./generated";

/** Фиксация номера: где и когда замечена машина. createdAt — ISO 8601 (RFC 3339). */
export type Car = Required<Pick<CarsearchPlateSighting, "id" | "plate" | "address" | "createdAt">> &
  Pick<CarsearchPlateSighting, "photoUrl" | "thumbUrl" | "addressLocale" | "plateRegion"> & {
    /** Формат номера: "ru" | "eu" | "us" | "" (неизвестен). */
    plateType: string;
  };

const plateTypes: Record<string, string> = {
  [CarsearchPlateType.PLATE_TYPE_RU]: "ru",
  [CarsearchPlateType.PLATE_TYPE_EU]: "eu",
  [CarsearchPlateType.PLATE_TYPE_US]: "us",
};

export interface CarsPage {
  items: Car[];
  total: number;
  nextOffset: number | null;
}

// include-test-data: бэк отдаёт и строки с is_test_data (сейчас в базе только тестовые данные).
const headers: Record<string, string> =
  import.meta.env.VITE_INCLUDE_TEST_DATA === "true" ? { "include-test-data": "true" } : {};

const api = new CarSearchApi({ baseUrl: "/api", baseApiParams: { headers } });

/** Ошибка API с сообщением для пользователя; status — HTTP-код (0 — сеть). */
export class ApiError extends Error {
  constructor(message: string, readonly status = 0) {
    super(message);
  }
}

function toCar(c: CarsearchPlateSighting): Car {
  return {
    id: c.id ?? "",
    plate: c.plate ?? "",
    address: c.address ?? "",
    createdAt: c.createdAt ?? "",
    photoUrl: c.photoUrl,
    thumbUrl: c.thumbUrl,
    addressLocale: c.addressLocale,
    plateType: plateTypes[c.plateType ?? ""] ?? "",
    plateRegion: c.plateRegion,
  };
}

export async function search(req: CarsearchSearchRequest, signal?: AbortSignal): Promise<CarsPage> {
  let data: CarsearchSearchResponse;
  try {
    ({ data } = await api.carsearchCarSearchService.carSearchServiceSearch(req, { signal }));
  } catch (e) {
    if (signal?.aborted) throw e;
    throw new ApiError(await errorMessage(e));
  }
  // proto3 JSON не передаёт нулевые значения: total = 0 и пустой items приходят как «нет поля».
  return {
    items: (data.items ?? []).map(toCar),
    total: data.total ?? 0,
    nextOffset: data.nextOffset ?? null,
  };
}

/** Все фиксации номера, к которому относится фиксация id; от новых к старым. */
export async function plateSightings(id: string, signal?: AbortSignal): Promise<Car[]> {
  let data: CarsearchGetResponse;
  try {
    ({ data } = await api.carsearchCarSearchService.carSearchServiceGet({ id }, { signal }));
  } catch (e) {
    if (signal?.aborted) throw e;
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
  return (data.items ?? []).map(toCar);
}

async function errorMessage(e: unknown): Promise<string> {
  // Клиент бросает сам Response при не-2xx; clay отдаёт {"error": "rpc error: code = … desc = …"}.
  if (e instanceof Response) {
    const body = (await e.json().catch(() => null)) as { error?: string; message?: string } | null;
    return body?.error || body?.message || `HTTP ${e.status}`;
  }
  return e instanceof Error ? e.message : String(e);
}
