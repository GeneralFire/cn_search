// Обёртка над сгенерированным клиентом (generated.ts — не править руками, см. `make gen-ts`).
import {
  CarSearchApi,
  CarsearchClarificationStatus,
  CarsearchComplaintReason,
  CarsearchComplaintStatus,
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
    /** Формат номера: "ru" | "eu" | "us" | "ua" | "cn" | "" (неизвестен). */
    plateType: string;
    /** Координаты места фиксации (WGS 84); нет — точка неизвестна (на карте не показывается). */
    lat?: number;
    lon?: number;
  };

const plateTypes: Record<string, string> = {
  [CarsearchPlateType.PLATE_TYPE_RU]: "ru",
  [CarsearchPlateType.PLATE_TYPE_EU]: "eu",
  [CarsearchPlateType.PLATE_TYPE_US]: "us",
  [CarsearchPlateType.PLATE_TYPE_UA]: "ua",
  [CarsearchPlateType.PLATE_TYPE_CN]: "cn",
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
  ua: CarsearchPlateType.PLATE_TYPE_UA,
  cn: CarsearchPlateType.PLATE_TYPE_CN,
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

// ---------------------------------------------------------------------------
// Жалобы на фото и админ-панель
// ---------------------------------------------------------------------------

/** Причина жалобы — как в БД (enum complaint_reason). */
export type ComplaintReason = "wrong_plate" | "bad_photo" | "personal_data" | "other";
export const COMPLAINT_REASONS: ComplaintReason[] = ["wrong_plate", "bad_photo", "personal_data", "other"];

const reasonEnum: Record<ComplaintReason, CarsearchComplaintReason> = {
  wrong_plate: CarsearchComplaintReason.COMPLAINT_REASON_WRONG_PLATE,
  bad_photo: CarsearchComplaintReason.COMPLAINT_REASON_BAD_PHOTO,
  personal_data: CarsearchComplaintReason.COMPLAINT_REASON_PERSONAL_DATA,
  other: CarsearchComplaintReason.COMPLAINT_REASON_OTHER,
};
const reasonName = Object.fromEntries(Object.entries(reasonEnum).map(([k, v]) => [v, k])) as Record<string, ComplaintReason>;

/** Жалоба на фото фиксации на проверку; статус новой — «новая». */
export async function complain(sightingId: string, reason: ComplaintReason, comment: string): Promise<void> {
  try {
    await api.carsearchCarSearchService.carSearchServiceComplain({ sightingId, reason: reasonEnum[reason], comment });
  } catch (e) {
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
}

// Токен админ-панели (заголовок admin-token) — только в sessionStorage этой вкладки.
const ADMIN_TOKEN_KEY = "admin-token";
export const getAdminToken = (): string => sessionStorage.getItem(ADMIN_TOKEN_KEY) ?? "";
export function setAdminToken(token: string): void {
  if (token) sessionStorage.setItem(ADMIN_TOKEN_KEY, token);
  else sessionStorage.removeItem(ADMIN_TOKEN_KEY);
}
const adminParams = () => ({ headers: { "admin-token": getAdminToken() } });

export type ComplaintStatus = "new" | "resolved";

/** Жалоба вместе с фиксацией (админ-панель). */
export interface Complaint {
  id: string;
  reason: ComplaintReason;
  comment: string;
  status: ComplaintStatus;
  createdAt: string;
  sighting: Car | null;
}

export interface ComplaintsPage {
  items: Complaint[];
  total: number;
  nextOffset: number | null;
}

const statusEnum: Record<ComplaintStatus | "", CarsearchComplaintStatus> = {
  "": CarsearchComplaintStatus.COMPLAINT_STATUS_UNSPECIFIED,
  new: CarsearchComplaintStatus.COMPLAINT_STATUS_NEW,
  resolved: CarsearchComplaintStatus.COMPLAINT_STATUS_RESOLVED,
};

/** Страница жалоб (админ); status "" — все. 403 (ApiError.status) — неверный или не заданный токен. */
export async function listComplaints(status: ComplaintStatus | "", offset: number, limit = 30): Promise<ComplaintsPage> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceListComplaints(
      { status: statusEnum[status], offset, limit },
      adminParams()
    );
    return {
      items: (data.items ?? []).map((c) => ({
        id: c.id ?? "",
        reason: reasonName[c.reason ?? ""] ?? "other",
        comment: c.comment ?? "",
        status: c.status === CarsearchComplaintStatus.COMPLAINT_STATUS_RESOLVED ? "resolved" : "new",
        createdAt: c.createdAt ?? "",
        sighting: c.sighting ? toCar(c.sighting) : null,
      })),
      total: data.total ?? 0,
      nextOffset: data.nextOffset ?? null,
    };
  } catch (e) {
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
}

/** Отметить жалобу решённой (админ). */
export async function resolveComplaint(id: string): Promise<void> {
  try {
    await api.carsearchCarSearchService.carSearchServiceResolveComplaint({ id }, adminParams());
  } catch (e) {
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
