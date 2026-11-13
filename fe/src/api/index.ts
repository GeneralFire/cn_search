// Обёртка над сгенерированным клиентом (generated.ts — не править руками, см. `make gen-ts`).
import {
  CarSearchApi,
  CarsearchCameraRequestStatus,
  CarsearchClarificationStatus,
  CarsearchComplaintReason,
  CarsearchComplaintStatus,
  CarsearchPaymentStatus,
  CarsearchPlateType,
  type CarsearchClarification,
  CarsearchUserPlan,
  type CarsearchCameraRequest,
  type CarsearchGetResponse,
  type CarsearchPayment,
  type CarsearchUser,
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
    /** Скрыта админом — приходит только в админ-режиме. */
    hidden?: boolean;
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
  /** «Возможно, вы искали»: только при пустой выдаче по непустому запросу. */
  suggestions: Car[];
}

// include-test-data: бэк отдаёт и строки с is_test_data (сейчас в базе только тестовые данные).
// Объект общий для всех запросов клиента: админ-режим (adminmode.ts) добавляет в него admin-token.
const headers: Record<string, string> =
  import.meta.env.VITE_INCLUDE_TEST_DATA === "true" ? { "include-test-data": "true" } : {};

/** Админ-режим: все запросы API идут с заголовком admin-token; "" — без него. */
export function setAdminHeader(token: string): void {
  if (token) headers["admin-token"] = token;
  else delete headers["admin-token"];
}

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
    ...(c.hidden ? { hidden: true } : {}),
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
    suggestions: (data.suggestions ?? []).map(toCar),
  };
}

/** Уточнения к фиксациям номера с одним предложенным номером (rpc Get). */
export interface PlateClarification {
  plate: string;
  /** Формат: "ru" | "eu" | "us" | "ua" | "cn" | "" (не указан). */
  plateType: string;
  plateRegion: string;
  count: number;
  accepted: boolean;
  /** Фиксации, к которым этот номер предлагали. */
  sightingIds: string[];
  /** Уточнения группы, от новых к старым (модерация в админ-режиме). */
  items: Clarification[];
}

/** Страница номера: все фиксации (от новых к старым) и уточнения к ним (сначала принятые). */
export interface PlatePage {
  items: Car[];
  clarifications: PlateClarification[];
}

/** Все фиксации номера, к которому относится фиксация id, и уточнения к ним. */
export async function plateSightings(id: string, signal?: AbortSignal): Promise<PlatePage> {
  let data: CarsearchGetResponse;
  try {
    ({ data } = await api.carsearchCarSearchService.carSearchServiceGet({ id }, { signal }));
  } catch (e) {
    if (signal?.aborted) throw e;
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
  return {
    items: (data.items ?? []).map(toCar),
    clarifications: (data.clarifications ?? []).map((c) => ({
      plate: c.plate ?? "",
      plateType: plateTypes[c.plateType ?? ""] ?? "",
      plateRegion: c.plateRegion ?? "",
      count: c.count ?? 0,
      accepted: c.status === CarsearchClarificationStatus.CLARIFICATION_STATUS_ACCEPTED,
      sightingIds: c.sightingIds ?? [],
      items: (c.items ?? []).map(toClarification),
    })),
  };
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
export async function listComplaints(
  status: ComplaintStatus | "",
  offset: number,
  limit = 30,
  sightingId = ""
): Promise<ComplaintsPage> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceListComplaints(
      { status: statusEnum[status], offset, limit, sightingId },
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

// ---------------------------------------------------------------------------
// Админ-режим: Me, скрытие фиксаций, разбор уточнений
// ---------------------------------------------------------------------------

/** Верен ли токен (rpc Me); ошибка сети — исключение, неверный токен — false. */
export async function checkAdmin(token: string): Promise<boolean> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceMe({}, { headers: { "admin-token": token } });
    return data.isAdmin === true;
  } catch (e) {
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
}

/** Скрыть фиксацию из выдачи (hidden = true) или вернуть её (админ). */
export async function hideSighting(id: string, hidden: boolean): Promise<Car> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceHideSighting({ id, hidden }, adminParams());
    return toCar(data);
  } catch (e) {
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
}

export type ClarificationStatus = "accepted" | "not_accepted";

/** Уточнение номера к фиксации (админ-панель); sighting — только из listClarifications. */
export interface Clarification {
  id: string;
  sightingId: string;
  plate: string;
  plateType: string;
  plateRegion: string;
  status: ClarificationStatus;
  createdAt: string;
  sighting: Car | null;
}

const clarStatusEnum: Record<ClarificationStatus | "", CarsearchClarificationStatus> = {
  "": CarsearchClarificationStatus.CLARIFICATION_STATUS_UNSPECIFIED,
  accepted: CarsearchClarificationStatus.CLARIFICATION_STATUS_ACCEPTED,
  not_accepted: CarsearchClarificationStatus.CLARIFICATION_STATUS_NOT_ACCEPTED,
};

function toClarification(c: CarsearchClarification): Clarification {
  return {
    id: c.id ?? "",
    sightingId: c.sightingId ?? "",
    plate: c.plate ?? "",
    plateType: plateTypes[c.plateType ?? ""] ?? "",
    plateRegion: c.plateRegion ?? "",
    status: c.status === CarsearchClarificationStatus.CLARIFICATION_STATUS_ACCEPTED ? "accepted" : "not_accepted",
    createdAt: c.createdAt ?? "",
    sighting: c.sighting ? toCar(c.sighting) : null,
  };
}

export interface ClarificationsPage {
  items: Clarification[];
  total: number;
  nextOffset: number | null;
}

/** Страница уточнений (админ); status "" — все, sightingId — только к этой фиксации. */
export async function listClarifications(
  status: ClarificationStatus | "",
  offset: number,
  limit = 30,
  sightingId = ""
): Promise<ClarificationsPage> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceListClarifications(
      { status: clarStatusEnum[status], offset, limit, sightingId },
      adminParams()
    );
    return { items: (data.items ?? []).map(toClarification), total: data.total ?? 0, nextOffset: data.nextOffset ?? null };
  } catch (e) {
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
}

/** Принять или отклонить уточнение (админ). */
export async function setClarificationStatus(id: string, status: ClarificationStatus): Promise<Clarification> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceSetClarificationStatus(
      { id, status: clarStatusEnum[status] },
      adminParams()
    );
    return toClarification(data);
  } catch (e) {
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
}

// ---------------------------------------------------------------------------
// Заявки на расположение камер (страница /cameras и вкладка админки)
// ---------------------------------------------------------------------------

export type CameraStatus = "new" | "approved" | "rejected" | "installed";
export const CAMERA_STATUSES: CameraStatus[] = ["new", "approved", "rejected", "installed"];

const cameraStatusEnum: Record<CameraStatus, CarsearchCameraRequestStatus> = {
  new: CarsearchCameraRequestStatus.CAMERA_REQUEST_STATUS_NEW,
  approved: CarsearchCameraRequestStatus.CAMERA_REQUEST_STATUS_APPROVED,
  rejected: CarsearchCameraRequestStatus.CAMERA_REQUEST_STATUS_REJECTED,
  installed: CarsearchCameraRequestStatus.CAMERA_REQUEST_STATUS_INSTALLED,
};
const cameraStatusName = Object.fromEntries(Object.entries(cameraStatusEnum).map(([k, v]) => [v, k])) as Record<string, CameraStatus>;

/** Заявка на камеру; contact — только в админском списке. */
export interface CameraRequest {
  id: string;
  lat: number;
  lon: number;
  address: string;
  reason: string;
  contact: string;
  status: CameraStatus;
  votes: number;
  createdAt: string;
}

export interface CameraRequestsPage {
  items: CameraRequest[];
  total: number;
  nextOffset: number | null;
}

function toCameraRequest(c: CarsearchCameraRequest): CameraRequest {
  return {
    id: c.id ?? "",
    lat: c.lat ?? 0,
    lon: c.lon ?? 0,
    address: c.address ?? "",
    reason: c.reason ?? "",
    contact: c.contact ?? "",
    status: cameraStatusName[c.status ?? ""] ?? "new",
    votes: c.votes ?? 0,
    createdAt: c.createdAt ?? "",
  };
}

const apiError = async (e: unknown) => new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);

/** Новая заявка на камеру (статус «новая»). */
export async function submitCameraRequest(req: {
  lat: number;
  lon: number;
  address: string;
  reason: string;
  contact: string;
}): Promise<CameraRequest> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceSubmitCameraRequest(req);
    return toCameraRequest(data);
  } catch (e) {
    throw await apiError(e);
  }
}

/** Страница заявок; admin — с контактами (заголовок admin-token, 403 — неверный токен). */
export async function listCameraRequests(
  opts: { status?: CameraStatus | ""; offset?: number; limit?: number; admin?: boolean },
  signal?: AbortSignal
): Promise<CameraRequestsPage> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceListCameraRequests(
      {
        status: opts.status ? cameraStatusEnum[opts.status] : CarsearchCameraRequestStatus.CAMERA_REQUEST_STATUS_UNSPECIFIED,
        offset: opts.offset ?? 0,
        limit: opts.limit ?? 0,
      },
      { signal, ...(opts.admin ? adminParams() : {}) }
    );
    return {
      items: (data.items ?? []).map(toCameraRequest),
      total: data.total ?? 0,
      nextOffset: data.nextOffset ?? null,
    };
  } catch (e) {
    if (signal?.aborted) throw e;
    throw await apiError(e);
  }
}

/** Голос «+1» за заявку; ответ — заявка с новым числом голосов. */
export async function voteCameraRequest(id: string): Promise<CameraRequest> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceVoteCameraRequest({ id });
    return toCameraRequest(data);
  } catch (e) {
    throw await apiError(e);
  }
}

/** Сменить статус заявки (админ). */
export async function setCameraRequestStatus(id: string, status: CameraStatus): Promise<CameraRequest> {
  try {
    const { data } = await api.carsearchCarSearchService.carSearchServiceSetCameraRequestStatus(
      { id, status: cameraStatusEnum[status] },
      adminParams()
    );
    return toCameraRequest(data);
  } catch (e) {
    throw await apiError(e);
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

// ---------------------------------------------------------------------------
// Аккаунт, тарифы, мок-оплата. Токен пользователя — cookie cs_token (auth.ts); браузер шлёт её сам
// (credentials: same-origin), бэк читает её из заголовка cookie.
// ---------------------------------------------------------------------------

export type UserPlan = "free" | "pro";
export type PaymentStatus = "pending" | "succeeded" | "canceled";

export interface User {
  id: string;
  email: string;
  plan: UserPlan;
  proUntil: string | null;
}

export interface Me {
  isAdmin: boolean;
  user: User | null; // null — не вошёл (нет токена или он неверный)
}

export interface PlanInfo {
  plan: UserPlan;
  title: string;
  amountCents: number;
  currency: string;
  periodDays: number;
  features: string[];
  uploadsPerDay: number;
}

export interface Payment {
  id: string;
  plan: UserPlan;
  amountCents: number;
  currency: string;
  status: PaymentStatus;
  createdAt: string;
  confirmedAt: string | null;
}

const planName = (p?: CarsearchUserPlan): UserPlan => (p === CarsearchUserPlan.USER_PLAN_PRO ? "pro" : "free");
const paymentStatusName: Record<string, PaymentStatus> = {
  [CarsearchPaymentStatus.PAYMENT_STATUS_SUCCEEDED]: "succeeded",
  [CarsearchPaymentStatus.PAYMENT_STATUS_CANCELED]: "canceled",
};

function toUser(u: CarsearchUser): User {
  return { id: u.id ?? "", email: u.email ?? "", plan: planName(u.plan), proUntil: u.proUntil ?? null };
}

function toPayment(p: CarsearchPayment): Payment {
  return {
    id: p.id ?? "",
    plan: planName(p.plan),
    amountCents: p.amountCents ?? 0,
    currency: p.currency ?? "RUB",
    status: paymentStatusName[p.status ?? ""] ?? "pending",
    createdAt: p.createdAt ?? "",
    confirmedAt: p.confirmedAt ?? null,
  };
}

async function call<T>(fn: () => Promise<{ data: T }>): Promise<T> {
  try {
    return (await fn()).data;
  } catch (e) {
    throw new ApiError(await errorMessage(e), e instanceof Response ? e.status : 0);
  }
}

/** Кто я. token — проверить конкретный токен (вход, ещё до cookie); иначе — по cookie. */
export async function me(token?: string): Promise<Me> {
  const params = { headers: { ...headers, ...(getAdminToken() ? { "admin-token": getAdminToken() } : {}),
    ...(token ? { authorization: `Bearer ${token}` } : {}) } };
  // С явным токеном cookie не отправляем: иначе бэк проверит её, а не введённый токен.
  const d = await call(() => api.carsearchCarSearchService.carSearchServiceMe({}, token ? { ...params, credentials: "omit" } : params));
  return { isAdmin: !!d.isAdmin, user: d.authenticated && d.user ? toUser(d.user) : null };
}

/** Регистрация: токен показывается один раз. */
export async function register(email: string): Promise<{ token: string; user: User }> {
  const d = await call(() => api.carsearchCarSearchService.carSearchServiceRegister({ email }));
  return { token: d.token ?? "", user: toUser(d.user ?? {}) };
}

export async function plans(): Promise<PlanInfo[]> {
  const d = await call(() => api.carsearchCarSearchService.carSearchServicePlans({}));
  return (d.items ?? []).map((p) => ({
    plan: planName(p.plan),
    title: p.title ?? "",
    amountCents: p.amountCents ?? 0,
    currency: p.currency ?? "RUB",
    periodDays: p.periodDays ?? 0,
    features: p.features ?? [],
    uploadsPerDay: p.uploadsPerDay ?? 0,
  }));
}

/** Платёж за Pro → адрес демо-кассы. 401 — не вошёл. */
export async function createPayment(): Promise<string> {
  const d = await call(() =>
    api.carsearchCarSearchService.carSearchServiceCreatePayment({ plan: CarsearchUserPlan.USER_PLAN_PRO })
  );
  return d.checkoutUrl ?? "/account";
}

export async function confirmMockPayment(paymentId: string, success: boolean): Promise<Payment> {
  return toPayment(await call(() => api.carsearchCarSearchService.carSearchServiceConfirmMockPayment({ paymentId, success })));
}

export async function myPayments(): Promise<Payment[]> {
  const d = await call(() => api.carsearchCarSearchService.carSearchServiceListMyPayments({}));
  return (d.items ?? []).map(toPayment);
}
