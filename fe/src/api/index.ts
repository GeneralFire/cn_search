// Обёртка над сгенерированным клиентом (generated.ts — не править руками, см. `make gen-ts`).
import {
  CarSearchApi,
  CarsearchClarificationStatus,
  CarsearchPlateType,
  type CarsearchGetResponse,
  type CarsearchPlateSighting,
  type CarsearchSearchRequest,
  type CarsearchSearchResponse,
} from "./generated";
import { localizeAddress, t } from "../i18n/i18n";

/** Фиксация номера: где и когда замечена машина. createdAt — ISO 8601 (RFC 3339).
 *  address — уже на языке интерфейса (localizeAddress), addressLocale — язык исходной записи. */
export type Car = Required<Pick<CarsearchPlateSighting, "id" | "plate" | "address" | "createdAt">> &
  Pick<CarsearchPlateSighting, "photoUrl" | "thumbUrl" | "addressLocale" | "plateRegion"> & {
    /** Формат номера: "ru" | "eu" | "us" | "" (неизвестен). */
    plateType: string;
    /** Координаты места фиксации (WGS 84); нет — точка неизвестна (на карте не показывается). */
    lat?: number;
    lon?: number;
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
    // Адрес показываем на языке интерфейса: совпадает с языком записи — как есть, иначе транслитерация.
    address: localizeAddress(c.address ?? "", c.addressLocale ?? "ru"),
    addressLocale: c.addressLocale ?? "ru",
    createdAt: c.createdAt ?? "",
    photoUrl: c.photoUrl,
    thumbUrl: c.thumbUrl,
    plateType: plateTypes[c.plateType ?? ""] ?? "",
    plateRegion: c.plateRegion,
    // Координаты — только парой: optional double в proto3 JSON приходит, только если задан.
    ...(c.lat != null && c.lon != null ? { lat: c.lat, lon: c.lon } : {}),
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

/** Уточнение номера к фиксации; статус нового — «не принят». */
const plateTypeEnum: Record<string, CarsearchPlateType> = {
  ru: CarsearchPlateType.PLATE_TYPE_RU,
  eu: CarsearchPlateType.PLATE_TYPE_EU,
  us: CarsearchPlateType.PLATE_TYPE_US,
};

/** Уточнение номера к фиксации: номер, формат ("ru" | "eu" | "us" | "") и регион (страна ЕС / штат США). */
export async function clarify(
  sightingId: string,
  plate: string,
  plateType = "",
  plateRegion = ""
): Promise<{ plate: string; accepted: boolean }> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceClarify({
      sightingId,
      plate,
      plateType: plateTypeEnum[plateType] ?? CarsearchPlateType.PLATE_TYPE_DEFAULT,
      plateRegion,
    });
    return {
      plate: data.clarification?.plate ?? plate,
      accepted: data.clarification?.status === CarsearchClarificationStatus.CLARIFICATION_STATUS_ACCEPTED,
    };
  } catch (e) {
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
}

/** Свежие фиксации других номеров (блок внизу страницы номера), не больше 21. */
export async function seeAlso(id: string, signal?: AbortSignal): Promise<Car[]> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceSeeAlso({ id }, { signal });
    return (data.items ?? []).map(toCar);
  } catch (e) {
    if (signal?.aborted) throw e;
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
}

async function errorMessage(e: unknown): Promise<string> {
  // Клиент бросает сам Response при не-2xx; clay отдаёт {"error": "rpc error: code = … desc = …"}.
  if (e instanceof Response) {
    const body = (await e.json().catch(() => null)) as { error?: string; message?: string } | null;
    return body?.error || body?.message || `HTTP ${e.status}`;
  }
  // Не Response — сеть недоступна (fetch бросил TypeError).
  return t("error.serverDown");
}
