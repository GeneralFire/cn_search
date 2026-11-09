// Обёртка над сгенерированным клиентом (generated.ts — не править руками, см. `make gen-ts`).
import { CarSearchApi, type CarsearchPlateSighting, type CarsearchSearchRequest, type CarsearchSearchResponse } from "./generated";

/** Фиксация номера: где и когда замечена машина. createdAt — ISO 8601 (RFC 3339). */
export type Car = Required<Pick<CarsearchPlateSighting, "id" | "plate" | "address" | "createdAt">> &
  Pick<CarsearchPlateSighting, "photoUrl">;

export interface CarsPage {
  items: Car[];
  total: number;
  nextOffset: number | null;
}

const api = new CarSearchApi({ baseUrl: "/api" });

/** Ошибка API с сообщением для пользователя. */
export class ApiError extends Error {}

function toCar(c: CarsearchPlateSighting): Car {
  return {
    id: c.id ?? "",
    plate: c.plate ?? "",
    address: c.address ?? "",
    createdAt: c.createdAt ?? "",
    photoUrl: c.photoUrl,
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

async function errorMessage(e: unknown): Promise<string> {
  // Клиент бросает сам Response при не-2xx; в теле clay отдаёт {"error": "...", "code": N}.
  if (e instanceof Response) {
    const body = (await e.json().catch(() => null)) as { error?: string; message?: string } | null;
    return body?.error || body?.message || `HTTP ${e.status}`;
  }
  return e instanceof Error ? e.message : String(e);
}
