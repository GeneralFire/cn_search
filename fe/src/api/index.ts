// Обёртка над сгенерированным клиентом (generated.ts — не править руками, см. `make gen-ts`).
import { CarSearchApi, type V1Car, type V1SearchCarsRequest, type V1SearchCarsResponse } from "./generated";

export type Car = Required<Pick<V1Car, "id" | "plate" | "key" | "address" | "date">> & Pick<V1Car, "photoUrl">;

export interface CarsPage {
  items: Car[];
  total: number;
  nextOffset: number | null;
}

const api = new CarSearchApi({ baseUrl: "/api" });

/** Ошибка API с сообщением для пользователя. */
export class ApiError extends Error {}

function toCar(c: V1Car): Car {
  return {
    id: c.id ?? 0,
    plate: c.plate ?? "",
    key: c.key ?? "",
    address: c.address ?? "",
    date: c.date ?? "",
    photoUrl: c.photoUrl,
  };
}

export async function searchCars(req: V1SearchCarsRequest, signal?: AbortSignal): Promise<CarsPage> {
  let data: V1SearchCarsResponse;
  try {
    ({ data } = await api.carsearchV1CarSearchService.carSearchServiceSearchCars(req, { signal }));
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
