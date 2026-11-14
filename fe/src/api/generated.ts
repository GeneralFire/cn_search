/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export type CarSearchServiceClarifyData = CarsearchClarifyResponse;

export type CarSearchServiceComplainData = CarsearchComplaint;

export type CarSearchServiceConfirmMockPaymentData = CarsearchPayment;

export type CarSearchServiceCreateApiTokenData =
  CarsearchCreateApiTokenResponse;

export type CarSearchServiceCreatePaymentData = CarsearchCreatePaymentResponse;

export type CarSearchServiceGetData = CarsearchGetResponse;

export type CarSearchServiceHideSightingData = CarsearchPlateSighting;

export type CarSearchServiceIngestData = CarsearchIngestResponse;

export type CarSearchServiceListApiTokensData = CarsearchListApiTokensResponse;

export type CarSearchServiceListCameraRequestsData =
  CarsearchListCameraRequestsResponse;

export type CarSearchServiceListClarificationsData =
  CarsearchListClarificationsResponse;

export type CarSearchServiceListComplaintsData =
  CarsearchListComplaintsResponse;

export type CarSearchServiceListMyPaymentsData =
  CarsearchListMyPaymentsResponse;

export type CarSearchServiceListPlateViewsData =
  CarsearchListPlateViewsResponse;

export type CarSearchServiceMeData = CarsearchMeResponse;

export type CarSearchServicePlansData = CarsearchPlansResponse;

export type CarSearchServiceRegisterData = CarsearchRegisterResponse;

export type CarSearchServiceResolveComplaintData = CarsearchComplaint;

export type CarSearchServiceRestoreSightingPhotoData = CarsearchPlateSighting;

export type CarSearchServiceRevokeApiTokenData = CarsearchApiToken;

export type CarSearchServiceSearchData = CarsearchSearchResponse;

export type CarSearchServiceSeeAlsoData = CarsearchSeeAlsoResponse;

export type CarSearchServiceSetCameraRequestStatusData = CarsearchCameraRequest;

export type CarSearchServiceSetClarificationStatusData = CarsearchClarification;

export type CarSearchServiceSubmitCameraRequestData = CarsearchCameraRequest;

export type CarSearchServiceTokenInfoData = CarsearchApiToken;

export type CarSearchServiceVoteCameraRequestData = CarsearchCameraRequest;

/** ApiToken — API-токен без самого токена (он показывается один раз в CreateApiToken). */
export interface CarsearchApiToken {
  /** @format date-time */
  createdAt?: string;
  id?: string;
  /**
   * Последний списанный запрос; не задан — токен ещё не использовался.
   * @format date-time
   */
  lastUsedAt?: string;
  name?: string;
  /** @format int32 */
  requestsLeft?: number;
  /** @format int32 */
  requestsTotal?: number;
  revoked?: boolean;
  /**
   * Сколько раз открывали страницы номеров с этим токеном (только в ListApiTokens).
   * @format int32
   */
  viewsCount?: number;
}

/** CameraRequest — заявка на расположение камеры. */
export interface CarsearchCameraRequest {
  address?: string;
  /** Контакт автора (e-mail/телеграм) — только администратору (ListCameraRequests с admin-token). */
  contact?: string;
  /** @format date-time */
  createdAt?: string;
  id?: string;
  /**
   * Точка (WGS 84, градусы).
   * @format double
   */
  lat?: number;
  /** @format double */
  lon?: number;
  reason?: string;
  /** Статус заявки на камеру. */
  status?: CarsearchCameraRequestStatus;
  /** @format int32 */
  votes?: number;
}

/**
 * Статус заявки на камеру.
 * @default "CAMERA_REQUEST_STATUS_UNSPECIFIED"
 */
export enum CarsearchCameraRequestStatus {
  CAMERA_REQUEST_STATUS_UNSPECIFIED = "CAMERA_REQUEST_STATUS_UNSPECIFIED",
  CAMERA_REQUEST_STATUS_NEW = "CAMERA_REQUEST_STATUS_NEW",
  CAMERA_REQUEST_STATUS_APPROVED = "CAMERA_REQUEST_STATUS_APPROVED",
  CAMERA_REQUEST_STATUS_REJECTED = "CAMERA_REQUEST_STATUS_REJECTED",
  CAMERA_REQUEST_STATUS_INSTALLED = "CAMERA_REQUEST_STATUS_INSTALLED",
}

/** Clarification — уточнение номера к фиксации. */
export interface CarsearchClarification {
  /** @format date-time */
  createdAt?: string;
  id?: string;
  /** Предложенный номер в нормализованном виде: "А123ВС 77". */
  plate?: string;
  plateRegion?: string;
  /** Формат номерного знака — от него зависит, как его рисовать. */
  plateType?: CarsearchPlateType;
  /** Фиксация (номер, адрес, фото) — только в ListClarifications. */
  sighting?: CarsearchPlateSighting;
  /** uuid фиксации, к которой уточнение. */
  sightingId?: string;
  /** Актуальный статус: ACCEPTED, когда совпали 3 уточнения фиксации. */
  status?: CarsearchClarificationStatus;
}

/**
 * Статус уточнения.
 * @default "CLARIFICATION_STATUS_UNSPECIFIED"
 */
export enum CarsearchClarificationStatus {
  CLARIFICATION_STATUS_UNSPECIFIED = "CLARIFICATION_STATUS_UNSPECIFIED",
  CLARIFICATION_STATUS_NOT_ACCEPTED = "CLARIFICATION_STATUS_NOT_ACCEPTED",
  CLARIFICATION_STATUS_ACCEPTED = "CLARIFICATION_STATUS_ACCEPTED",
}

export interface CarsearchClarifyRequest {
  /**
   * Номер, 1..32 символа. RU: А123ВС 77 / А123ВС 777 (пробел перед регионом можно не ставить);
   * EU/US: 2..12 символов — латиница, цифры, пробел, дефис; DEFAULT — формат не проверяется.
   */
  plate?: string;
  /** Страна (ЕС: "D") или штат (США: "CA"), до 3 латинских букв; для RU/DEFAULT не нужен. */
  plateRegion?: string;
  /** Формат номера; DEFAULT — другой/неизвестен. */
  plateType?: CarsearchPlateType;
  /** uuid фиксации. */
  sightingId?: string;
}

export interface CarsearchClarifyResponse {
  /** Clarification — уточнение номера к фиксации. */
  clarification?: CarsearchClarification;
}

export interface CarsearchComplainRequest {
  /** Комментарий, до 500 символов; можно пусто. */
  comment?: string;
  /** Причина, обязательна. */
  reason?: CarsearchComplaintReason;
  /** uuid фиксации. */
  sightingId?: string;
}

/** Complaint — жалоба на фото фиксации. */
export interface CarsearchComplaint {
  comment?: string;
  /** @format date-time */
  createdAt?: string;
  id?: string;
  /** Причина жалобы на фото. */
  reason?: CarsearchComplaintReason;
  /** Фиксация (номер, адрес, фото, время) — только в ListComplaints. */
  sighting?: CarsearchPlateSighting;
  /** uuid фиксации, на фото которой жалоба. */
  sightingId?: string;
  /** Статус жалобы. */
  status?: CarsearchComplaintStatus;
}

/**
 * Причина жалобы на фото.
 * @default "COMPLAINT_REASON_UNSPECIFIED"
 */
export enum CarsearchComplaintReason {
  COMPLAINT_REASON_UNSPECIFIED = "COMPLAINT_REASON_UNSPECIFIED",
  COMPLAINT_REASON_WRONG_PLATE = "COMPLAINT_REASON_WRONG_PLATE",
  COMPLAINT_REASON_BAD_PHOTO = "COMPLAINT_REASON_BAD_PHOTO",
  COMPLAINT_REASON_PERSONAL_DATA = "COMPLAINT_REASON_PERSONAL_DATA",
  COMPLAINT_REASON_OTHER = "COMPLAINT_REASON_OTHER",
}

/**
 * Статус жалобы.
 * @default "COMPLAINT_STATUS_UNSPECIFIED"
 */
export enum CarsearchComplaintStatus {
  COMPLAINT_STATUS_UNSPECIFIED = "COMPLAINT_STATUS_UNSPECIFIED",
  COMPLAINT_STATUS_NEW = "COMPLAINT_STATUS_NEW",
  COMPLAINT_STATUS_RESOLVED = "COMPLAINT_STATUS_RESOLVED",
}

export interface CarsearchConfirmMockPaymentRequest {
  paymentId?: string;
  /** true — «Оплатить», false — «Отменить». */
  success?: boolean;
}

export interface CarsearchCreateApiTokenRequest {
  /** Название для админки, 1..100 символов. */
  name?: string;
  /**
   * Сколько запросов (открытий страницы номера) доступно, 1..1000000.
   * @format int32
   */
  requests?: number;
}

export interface CarsearchCreateApiTokenResponse {
  /** ApiToken — API-токен без самого токена (он показывается один раз в CreateApiToken). */
  info?: CarsearchApiToken;
  /** Токен "cst_…" — единственный раз; потом его не узнать. */
  token?: string;
}

export interface CarsearchCreatePaymentRequest {
  /** Покупаемый тариф; сейчас только PRO. */
  plan?: CarsearchUserPlan;
}

export interface CarsearchCreatePaymentResponse {
  /** Куда перейти для оплаты: "/pay/mock/<id>". */
  checkoutUrl?: string;
  /** Payment — платёж за тариф. */
  payment?: CarsearchPayment;
  paymentId?: string;
}

/** Рамка карты: заявки с min_lat <= lat <= max_lat и min_lon <= lon <= max_lon. */
export interface CarsearchGeoBox {
  /** @format double */
  maxLat?: number;
  /** @format double */
  maxLon?: number;
  /** @format double */
  minLat?: number;
  /** @format double */
  minLon?: number;
}

export interface CarsearchGetRequest {
  /** uuid любой фиксации номера. */
  id?: string;
}

export interface CarsearchGetResponse {
  /**
   * Уточнения ко всем фиксациям номера, сгруппированные по предложенному номеру (без учёта
   * похожих символов): сначала принятые, затем по числу предложений и свежести. Нет — пусто.
   */
  clarifications?: CarsearchPlateClarification[];
  /** Все фиксации номера, от новых к старым; первая по id всегда среди них. */
  items?: CarsearchPlateSighting[];
}

export interface CarsearchHideSightingRequest {
  /** true — скрыть, false — показать снова. */
  hidden?: boolean;
  /** uuid фиксации. */
  id?: string;
}

export interface CarsearchIngestRequest {
  /** Адрес, до 300 символов; можно пусто. Язык адреса определяется по тексту (пусто — "ru"). */
  address?: string;
  /**
   * Координаты места (WGS 84); не заданы — из EXIF GPS, иначе неизвестны. Только парой.
   * @format double
   */
  lat?: number;
  /** @format double */
  lon?: number;
  /**
   * Фото (JPEG или PNG), до 8 МБ.
   * @format byte
   */
  photo?: Blob;
  /**
   * Когда снято; не задано — из EXIF (DateTimeOriginal), иначе время загрузки.
   * @format date-time
   */
  takenAt?: string;
}

export interface CarsearchIngestResponse {
  /** Все распознанные номера, по убыванию уверенности; в фиксацию попал первый. */
  candidates?: CarsearchPlateCandidate[];
  /** Созданная фиксация. */
  sighting?: CarsearchPlateSighting;
  /**
   * Сколько загрузок ещё можно за последние 24 часа (тариф Free); не задано — без ограничений (Pro).
   * @format int32
   */
  uploadsLeft?: number;
}

export interface CarsearchListApiTokensRequest {
  /**
   * Размер страницы, 1..100; 0 — по умолчанию (30).
   * @format int32
   */
  limit?: number;
  /** @format int32 */
  offset?: number;
}

export interface CarsearchListApiTokensResponse {
  items?: CarsearchApiToken[];
  /** @format int32 */
  nextOffset?: number;
  /** @format int32 */
  total?: number;
}

export interface CarsearchListCameraRequestsRequest {
  /** Рамка карты; не задана — все. */
  bbox?: CarsearchGeoBox;
  /**
   * Размер страницы, 1..500; 0 — по умолчанию (100).
   * @format int32
   */
  limit?: number;
  /**
   * Сколько заявок пропустить («Загрузить ещё»).
   * @format int32
   */
  offset?: number;
  /** Фильтр по статусу; UNSPECIFIED — все. */
  status?: CarsearchCameraRequestStatus;
}

export interface CarsearchListCameraRequestsResponse {
  items?: CarsearchCameraRequest[];
  /**
   * Смещение следующей страницы; не задано — дальше ничего нет.
   * @format int32
   */
  nextOffset?: number;
  /**
   * Всего заявок по фильтру.
   * @format int32
   */
  total?: number;
}

export interface CarsearchListClarificationsRequest {
  /**
   * Размер страницы, 1..100; 0 — по умолчанию (30).
   * @format int32
   */
  limit?: number;
  /**
   * Сколько пропустить («Загрузить ещё»).
   * @format int32
   */
  offset?: number;
  /** Только уточнения к этой фиксации (uuid); пусто — все. */
  sightingId?: string;
  /** Фильтр по статусу; UNSPECIFIED — все. */
  status?: CarsearchClarificationStatus;
}

export interface CarsearchListClarificationsResponse {
  items?: CarsearchClarification[];
  /**
   * Смещение следующей страницы; не задано — дальше ничего нет.
   * @format int32
   */
  nextOffset?: number;
  /**
   * Всего уточнений по фильтру.
   * @format int32
   */
  total?: number;
}

export interface CarsearchListComplaintsRequest {
  /**
   * Размер страницы, 1..100; 0 — по умолчанию (30).
   * @format int32
   */
  limit?: number;
  /**
   * Сколько жалоб пропустить («Загрузить ещё»).
   * @format int32
   */
  offset?: number;
  /** Только жалобы на эту фиксацию (uuid); пусто — все. */
  sightingId?: string;
  /** Фильтр по статусу; UNSPECIFIED — все. */
  status?: CarsearchComplaintStatus;
}

export interface CarsearchListComplaintsResponse {
  items?: CarsearchComplaint[];
  /**
   * Смещение следующей страницы; не задано — дальше ничего нет.
   * @format int32
   */
  nextOffset?: number;
  /**
   * Всего жалоб по фильтру.
   * @format int32
   */
  total?: number;
}

export type CarsearchListMyPaymentsRequest = object;

export interface CarsearchListMyPaymentsResponse {
  items?: CarsearchPayment[];
}

export interface CarsearchListPlateViewsRequest {
  /**
   * Размер страницы, 1..100; 0 — по умолчанию (30).
   * @format int32
   */
  limit?: number;
  /** @format int32 */
  offset?: number;
  /** Фильтр по uuid номера (plates.id); пусто — все. */
  plateId?: string;
  /** Фильтр по uuid токена; пусто — все. */
  tokenId?: string;
}

export interface CarsearchListPlateViewsResponse {
  items?: CarsearchPlateView[];
  /** @format int32 */
  nextOffset?: number;
  /** @format int32 */
  total?: number;
}

export type CarsearchMeRequest = object;

export interface CarsearchMeResponse {
  /** Токен пользователя верный; иначе user не задан. */
  authenticated?: boolean;
  /** Заголовок admin-token верный. */
  isAdmin?: boolean;
  /**
   * Сколько загрузок фото (rpc Ingest) осталось за последние 24 часа; не задано — без ограничений
   * (Pro), не вошёл или загрузка фото выключена.
   * @format int32
   */
  uploadsLeft?: number;
  /** User — пользователь. */
  user?: CarsearchUser;
}

/** Payment — платёж за тариф. */
export interface CarsearchPayment {
  /** @format int32 */
  amountCents?: number;
  /**
   * Когда оплачен или отменён.
   * @format date-time
   */
  confirmedAt?: string;
  /** @format date-time */
  createdAt?: string;
  currency?: string;
  id?: string;
  /** Тариф пользователя. */
  plan?: CarsearchUserPlan;
  /** Платёжный провайдер; пока только "mock". */
  provider?: string;
  /** Статус платежа. */
  status?: CarsearchPaymentStatus;
}

/**
 * Статус платежа.
 * @default "PAYMENT_STATUS_UNSPECIFIED"
 */
export enum CarsearchPaymentStatus {
  PAYMENT_STATUS_UNSPECIFIED = "PAYMENT_STATUS_UNSPECIFIED",
  PAYMENT_STATUS_PENDING = "PAYMENT_STATUS_PENDING",
  PAYMENT_STATUS_SUCCEEDED = "PAYMENT_STATUS_SUCCEEDED",
  PAYMENT_STATUS_CANCELED = "PAYMENT_STATUS_CANCELED",
}

/** PlanInfo — тариф. */
export interface CarsearchPlanInfo {
  /**
   * Цена за период в копейках/центах; 0 — бесплатно.
   * @format int32
   */
  amountCents?: number;
  /** Валюта ISO 4217: "RUB". */
  currency?: string;
  /** Ключи возможностей: "search", "uploads_limited", "pro_badge", "unlimited_uploads", "notifications". */
  features?: string[];
  /**
   * Период в днях; 0 — бессрочно.
   * @format int32
   */
  periodDays?: number;
  /** Тариф пользователя. */
  plan?: CarsearchUserPlan;
  title?: string;
  /**
   * Лимит загрузок фото в сутки; 0 — без ограничений.
   * @format int32
   */
  uploadsPerDay?: number;
}

export type CarsearchPlansRequest = object;

export interface CarsearchPlansResponse {
  items?: CarsearchPlanInfo[];
}

/** PlateCandidate — номер, распознанный на фото. */
export interface CarsearchPlateCandidate {
  /** Рамка номера на фото, пиксели: x1, y1, x2, y2. */
  bbox?: number[];
  /**
   * Уверенность 0..1.
   * @format double
   */
  confidence?: number;
  /** Номер после нормализации по формату: "Р466УК 79" (RU — кириллицей). */
  plate?: string;
  /** Формат номерного знака — от него зависит, как его рисовать. */
  plateType?: CarsearchPlateType;
  /** Как прочитала модель: "P466YK79". */
  raw?: string;
}

/** PlateClarification — предложенный номер и сколько раз его предлагали к фиксациям номера. */
export interface CarsearchPlateClarification {
  /**
   * Сколько уточнений с этим номером.
   * @format int32
   */
  count?: number;
  /** Сами уточнения группы, от новых к старым (модерация в админ-режиме — по id). */
  items?: CarsearchClarification[];
  /** Предложенный номер (написание из самого свежего уточнения группы): "А123ВС 77". */
  plate?: string;
  plateRegion?: string;
  /** Формат номерного знака — от него зависит, как его рисовать. */
  plateType?: CarsearchPlateType;
  /** uuid фиксаций, к которым этот номер предлагали. */
  sightingIds?: string[];
  /** ACCEPTED, если среди них есть принятые; иначе NOT_ACCEPTED. */
  status?: CarsearchClarificationStatus;
}

/** PlateSighting — фиксация номера: где и когда замечена машина. */
export interface CarsearchPlateSighting {
  address?: string;
  /** Язык, на котором записан адрес: "ru", "en", "de", … */
  addressLocale?: string;
  /**
   * Когда номер зафиксирован.
   * @format date-time
   */
  createdAt?: string;
  /** Фиксация скрыта админом; приходит только админу (обычным пользователям скрытые не отдаются). */
  hidden?: boolean;
  /** uuid фиксации. */
  id?: string;
  /**
   * Координаты места фиксации (WGS 84, градусы); не заданы — точка неизвестна.
   * @format double
   */
  lat?: number;
  /** @format double */
  lon?: number;
  /**
   * Фото скрыто по жалобам (≥ 3 новых жалоб bad_photo/personal_data): обычным пользователям photo_url
   * и thumb_url приходят пустыми, админу — как есть. Снимают ResolveComplaint и RestoreSightingPhoto.
   */
  photoHidden?: boolean;
  /** URL полного фото (S3). */
  photoUrl?: string;
  /** Госномер кириллицей: "А123ВС 777". */
  plate?: string;
  /** Страна (ЕС: "D") или штат (США: "CA") для формата номера. */
  plateRegion?: string;
  /** Формат номера (plates.meta.type); не задан — неизвестен. */
  plateType?: CarsearchPlateType;
  /** URL миниатюры (S3) — для списков; полное фото — photo_url. */
  thumbUrl?: string;
}

/**
 * Формат номерного знака — от него зависит, как его рисовать.
 * @default "PLATE_TYPE_DEFAULT"
 */
export enum CarsearchPlateType {
  PLATE_TYPE_DEFAULT = "PLATE_TYPE_DEFAULT",
  PLATE_TYPE_RU = "PLATE_TYPE_RU",
  PLATE_TYPE_EU = "PLATE_TYPE_EU",
  PLATE_TYPE_US = "PLATE_TYPE_US",
  PLATE_TYPE_UA = "PLATE_TYPE_UA",
  PLATE_TYPE_CN = "PLATE_TYPE_CN",
}

/** PlateView — открытие страницы номера (запись журнала plate_views). */
export interface CarsearchPlateView {
  /** Списан ли запрос из токена. */
  consumed?: boolean;
  /** @format date-time */
  createdAt?: string;
  id?: string;
  /** Открыл администратор (admin-token). */
  isAdmin?: boolean;
  /** Открытая фиксация (номер, фото, адрес). */
  sighting?: CarsearchPlateSighting;
  /** API-токен запроса; пусто — без токена. */
  tokenId?: string;
  tokenName?: string;
  userAgent?: string;
}

export interface CarsearchRegisterRequest {
  /** Необязательно; уникален. */
  email?: string;
}

export interface CarsearchRegisterResponse {
  /** Токен входа "cs_…" — показывается один раз, восстановить нельзя. */
  token?: string;
  /** User — пользователь. */
  user?: CarsearchUser;
}

export interface CarsearchResolveComplaintRequest {
  /** uuid жалобы. */
  id?: string;
}

export interface CarsearchRestoreSightingPhotoRequest {
  /** uuid фиксации. */
  id?: string;
}

export interface CarsearchRevokeApiTokenRequest {
  /** uuid токена. */
  id?: string;
}

export interface CarsearchSearchRequest {
  /** Только фиксации с координатами (есть на карте). */
  hasCoords?: boolean;
  /**
   * Размер страницы, 1..100; 0 — по умолчанию (35).
   * @format int32
   */
  limit?: number;
  /**
   * «Рынок» номеров (главная: блок по языку интерфейса и вкладки стран):
   * "ru" — российские, "ua" — украинские, "de" — ЕС с регионом D, "us" — США, "cn" — китайские,
   * "other" — всё, что не попало ни в один рынок (формат неизвестен или другая страна).
   * Пусто — без фильтра.
   */
  market?: string;
  /**
   * Сколько записей пропустить (для бесконечной прокрутки).
   * @format int32
   */
  offset?: number;
  /**
   * Поисковая строка, как есть передаётся в plainto_tsquery('simple', plate) и сопоставляется
   * с to_tsvector('simple', plate): каждое слово должно совпасть со словом номера —
   * серией "А123ВС" или регионом "777". Пусто — все записи.
   */
  plate?: string;
  /**
   * Фильтр периода по времени фиксации: since <= created_at < until. Не задано — без границы.
   * @format date-time
   */
  since?: string;
  /** @format date-time */
  until?: string;
}

export interface CarsearchSearchResponse {
  items?: CarsearchPlateSighting[];
  /**
   * Смещение следующей страницы; не задано — дальше ничего нет.
   * @format int32
   */
  nextOffset?: number;
  /**
   * «Возможно, вы искали»: только при total = 0 и непустом запросе — до 5 номеров, ближайших
   * к запросу по расстоянию правки (levenshtein) между ключами plate_search_key без пробелов,
   * не дальше 2 правок; по одной (последней) фиксации на номер, ближайшие первыми.
   */
  suggestions?: CarsearchPlateSighting[];
  /**
   * Всего найдено номеров по запросу.
   * @format int32
   */
  total?: number;
}

export interface CarsearchSeeAlsoRequest {
  /** uuid открытой фиксации: её номер в выдачу не попадает. Пусто — без исключения. */
  id?: string;
  /**
   * Сколько карточек, 1..21; 0 — 21.
   * @format int32
   */
  limit?: number;
}

export interface CarsearchSeeAlsoResponse {
  items?: CarsearchPlateSighting[];
}

export interface CarsearchSetCameraRequestStatusRequest {
  /** uuid заявки. */
  id?: string;
  /** Новый статус (не UNSPECIFIED). */
  status?: CarsearchCameraRequestStatus;
}

export interface CarsearchSetClarificationStatusRequest {
  /** uuid уточнения. */
  id?: string;
  /** ACCEPTED или NOT_ACCEPTED. */
  status?: CarsearchClarificationStatus;
}

export interface CarsearchSubmitCameraRequestRequest {
  /** Адрес, до 300 символов; можно пусто. */
  address?: string;
  /** E-mail/телеграм для связи, до 200 символов; можно пусто. */
  contact?: string;
  /**
   * Точка, обязательна: lat -90..90, lon -180..180.
   * @format double
   */
  lat?: number;
  /** @format double */
  lon?: number;
  /** Почему здесь нужна камера, 1..1000 символов. */
  reason?: string;
}

export type CarsearchTokenInfoRequest = object;

/** User — пользователь. */
export interface CarsearchUser {
  /** @format date-time */
  createdAt?: string;
  /** E-mail; пусто — не указан. */
  email?: string;
  /** uuid пользователя. */
  id?: string;
  /** Действующий тариф (Pro с истёкшим сроком — уже Free). */
  plan?: CarsearchUserPlan;
  /**
   * До какого момента действует Pro; не задано — не покупал.
   * @format date-time
   */
  proUntil?: string;
}

/**
 * Тариф пользователя.
 * @default "USER_PLAN_UNSPECIFIED"
 */
export enum CarsearchUserPlan {
  USER_PLAN_UNSPECIFIED = "USER_PLAN_UNSPECIFIED",
  USER_PLAN_FREE = "USER_PLAN_FREE",
  USER_PLAN_PRO = "USER_PLAN_PRO",
}

export interface CarsearchVoteCameraRequestRequest {
  /** uuid заявки. */
  id?: string;
}

export type QueryParamsType = Record<string | number, any>;
export type ResponseFormat = keyof Omit<Body, "body" | "bodyUsed">;

export interface FullRequestParams extends Omit<RequestInit, "body"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseFormat;
  /** request body */
  body?: unknown;
  /** base url */
  baseUrl?: string;
  /** request cancellation token */
  cancelToken?: CancelToken;
}

export type RequestParams = Omit<
  FullRequestParams,
  "body" | "method" | "query" | "path"
>;

export interface ApiConfig<SecurityDataType = unknown> {
  baseUrl?: string;
  baseApiParams?: Omit<RequestParams, "baseUrl" | "cancelToken" | "signal">;
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<RequestParams | void> | RequestParams | void;
  customFetch?: typeof fetch;
}

export interface HttpResponse<D extends unknown, E extends unknown = unknown>
  extends Response {
  data: D;
  error: E;
}

type CancelToken = Symbol | string | number;

export enum ContentType {
  Json = "application/json",
  JsonApi = "application/vnd.api+json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
  Text = "text/plain",
}

export class HttpClient<SecurityDataType = unknown> {
  public baseUrl: string = "";
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private abortControllers = new Map<CancelToken, AbortController>();
  private customFetch = (...fetchParams: Parameters<typeof fetch>) =>
    fetch(...fetchParams);

  private baseApiParams: RequestParams = {
    credentials: "same-origin",
    headers: {},
    redirect: "follow",
    referrerPolicy: "no-referrer",
  };

  constructor(apiConfig: ApiConfig<SecurityDataType> = {}) {
    Object.assign(this, apiConfig);
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  protected encodeQueryParam(key: string, value: any) {
    const encodedKey = encodeURIComponent(key);
    return `${encodedKey}=${encodeURIComponent(typeof value === "number" ? value : `${value}`)}`;
  }

  protected addQueryParam(query: QueryParamsType, key: string) {
    return this.encodeQueryParam(key, query[key]);
  }

  protected addArrayQueryParam(query: QueryParamsType, key: string) {
    const value = query[key];
    return value.map((v: any) => this.encodeQueryParam(key, v)).join("&");
  }

  protected toQueryString(rawQuery?: QueryParamsType): string {
    const query = rawQuery || {};
    const keys = Object.keys(query).filter(
      (key) => "undefined" !== typeof query[key],
    );
    return keys
      .map((key) =>
        Array.isArray(query[key])
          ? this.addArrayQueryParam(query, key)
          : this.addQueryParam(query, key),
      )
      .join("&");
  }

  protected addQueryParams(rawQuery?: QueryParamsType): string {
    const queryString = this.toQueryString(rawQuery);
    return queryString ? `?${queryString}` : "";
  }

  private contentFormatters: Record<ContentType, (input: any) => any> = {
    [ContentType.Json]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.JsonApi]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.Text]: (input: any) =>
      input !== null && typeof input !== "string"
        ? JSON.stringify(input)
        : input,
    [ContentType.FormData]: (input: any) => {
      if (input instanceof FormData) {
        return input;
      }

      return Object.keys(input || {}).reduce((formData, key) => {
        const property = input[key];
        formData.append(
          key,
          property instanceof Blob
            ? property
            : typeof property === "object" && property !== null
              ? JSON.stringify(property)
              : `${property}`,
        );
        return formData;
      }, new FormData());
    },
    [ContentType.UrlEncoded]: (input: any) => this.toQueryString(input),
  };

  protected mergeRequestParams(
    params1: RequestParams,
    params2?: RequestParams,
  ): RequestParams {
    return {
      ...this.baseApiParams,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...(this.baseApiParams.headers || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  protected createAbortSignal = (
    cancelToken: CancelToken,
  ): AbortSignal | undefined => {
    if (this.abortControllers.has(cancelToken)) {
      const abortController = this.abortControllers.get(cancelToken);
      if (abortController) {
        return abortController.signal;
      }
      return void 0;
    }

    const abortController = new AbortController();
    this.abortControllers.set(cancelToken, abortController);
    return abortController.signal;
  };

  public abortRequest = (cancelToken: CancelToken) => {
    const abortController = this.abortControllers.get(cancelToken);

    if (abortController) {
      abortController.abort();
      this.abortControllers.delete(cancelToken);
    }
  };

  public request = async <T = any, E = any>({
    body,
    secure,
    path,
    type,
    query,
    format,
    baseUrl,
    cancelToken,
    ...params
  }: FullRequestParams): Promise<HttpResponse<T, E>> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.baseApiParams.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const queryString = query && this.toQueryString(query);
    const payloadFormatter = this.contentFormatters[type || ContentType.Json];
    const responseFormat = format || requestParams.format;

    return this.customFetch(
      `${baseUrl || this.baseUrl || ""}${path}${queryString ? `?${queryString}` : ""}`,
      {
        ...requestParams,
        headers: {
          ...(requestParams.headers || {}),
          ...(type && type !== ContentType.FormData
            ? { "Content-Type": type }
            : {}),
        },
        signal:
          (cancelToken
            ? this.createAbortSignal(cancelToken)
            : requestParams.signal) || null,
        body:
          typeof body === "undefined" || body === null
            ? null
            : payloadFormatter(body),
      },
    ).then(async (response) => {
      const r = response as HttpResponse<T, E>;
      r.data = null as unknown as T;
      r.error = null as unknown as E;

      const responseToParse = responseFormat ? response.clone() : response;
      const data = !responseFormat
        ? r
        : await responseToParse[responseFormat]()
            .then((data) => {
              if (r.ok) {
                r.data = data;
              } else {
                r.error = data;
              }
              return r;
            })
            .catch((e) => {
              r.error = e;
              return r;
            });

      if (cancelToken) {
        this.abortControllers.delete(cancelToken);
      }

      if (!response.ok) throw data;
      return data;
    });
  };
}

/**
 * @title carsearch/carsearch.proto
 * @version version not set
 */
export class CarSearchApi<
  SecurityDataType extends unknown,
> extends HttpClient<SecurityDataType> {
  carsearchCarSearchService = {
    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceClarify
     * @summary Clarify — уточнение номера к фиксации («на самом деле номер такой»). Номер нормализуется (пробелы, регистр; для RU — латинские двойники → кириллица, «0» в буквенных позициях → «О») и проверяется по формату plate_type; ошибка формата или совпадение с текущим номером фиксации — INVALID_ARGUMENT с текстом для пользователя. Создаётся «не принят»; когда 3 уточнения фиксации совпали (без учёта похожих символов), все они становятся «принят».
     * @request POST:/carsearch.CarSearchService/Clarify
     */
    carSearchServiceClarify: (
      body: CarsearchClarifyRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceClarifyData, any>({
        path: `/carsearch.CarSearchService/Clarify`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceComplain
     * @summary Complain — жалоба на фото фиксации (кнопка «Пожаловаться»). Создаётся со статусом «новая».
     * @request POST:/carsearch.CarSearchService/Complain
     */
    carSearchServiceComplain: (
      body: CarsearchComplainRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceComplainData, any>({
        path: `/carsearch.CarSearchService/Complain`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceConfirmMockPayment
     * @summary ConfirmMockPayment — итог демо-оплаты: success — платёж оплачен, Pro продлён на 30 дней от max(сейчас, pro_until); иначе — отменён. Только свой платёж в статусе pending. Нужен токен.
     * @request POST:/carsearch.CarSearchService/ConfirmMockPayment
     */
    carSearchServiceConfirmMockPayment: (
      body: CarsearchConfirmMockPaymentRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceConfirmMockPaymentData, any>({
        path: `/carsearch.CarSearchService/ConfirmMockPayment`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceCreateApiToken
     * @summary CreateApiToken — новый API-токен на requests запросов (админ). Токен в открытом виде возвращается только здесь, бэк хранит лишь sha256 его подписи.
     * @request POST:/carsearch.CarSearchService/CreateApiToken
     */
    carSearchServiceCreateApiToken: (
      body: CarsearchCreateApiTokenRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceCreateApiTokenData, any>({
        path: `/carsearch.CarSearchService/CreateApiToken`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceCreatePayment
     * @summary CreatePayment — платёж за тариф и адрес кассы (пока демо-касса /pay/mock/<id>). Нужен токен пользователя, иначе UNAUTHENTICATED.
     * @request POST:/carsearch.CarSearchService/CreatePayment
     */
    carSearchServiceCreatePayment: (
      body: CarsearchCreatePaymentRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceCreatePaymentData, any>({
        path: `/carsearch.CarSearchService/CreatePayment`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceGet
     * @summary Get по идентификатору фиксации возвращает все фиксации того же номера — от новых к старым. Каждое открытие пишется в журнал plate_views. С API-токеном (cookie cs_token = "cst_…" или authorization: Bearer cst_…) списывает один запрос: исчерпан — RESOURCE_EXHAUSTED, отозван — PERMISSION_DENIED, подпись неверна — UNAUTHENTICATED. С admin-token квота не списывается.
     * @request POST:/carsearch.CarSearchService/Get
     */
    carSearchServiceGet: (
      body: CarsearchGetRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceGetData, any>({
        path: `/carsearch.CarSearchService/Get`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceHideSighting
     * @summary HideSighting — скрыть фиксацию (hidden = true) или вернуть её. Скрытые фиксации не попадают в Search/Get/SeeAlso обычных пользователей; админу отдаются с hidden = true. Нужен admin-token.
     * @request POST:/carsearch.CarSearchService/HideSighting
     */
    carSearchServiceHideSighting: (
      body: CarsearchHideSightingRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceHideSightingData, any>({
        path: `/carsearch.CarSearchService/HideSighting`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceIngest
     * @summary Ingest — новая фиксация по фото: номер распознаёт OCR-сервис (ocr/), фото и миниатюра уходят в S3, номер находится или создаётся, фиксация сохраняется. Номер не распознан (уверенность ниже порога) или файл не фото — INVALID_ARGUMENT; OCR недоступен — UNAVAILABLE. По HTTP (clay, JSON) photo передаётся строкой base64. Нужен вход (токен пользователя), иначе UNAUTHENTICATED; тариф Free — не больше 5 загрузок за 24 часа, сверх — RESOURCE_EXHAUSTED; Pro — без лимита.
     * @request POST:/carsearch.CarSearchService/Ingest
     */
    carSearchServiceIngest: (
      body: CarsearchIngestRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceIngestData, any>({
        path: `/carsearch.CarSearchService/Ingest`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceListApiTokens
     * @summary ListApiTokens — токены от новых к старым (админ).
     * @request POST:/carsearch.CarSearchService/ListApiTokens
     */
    carSearchServiceListApiTokens: (
      body: CarsearchListApiTokensRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceListApiTokensData, any>({
        path: `/carsearch.CarSearchService/ListApiTokens`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceListCameraRequests
     * @summary ListCameraRequests — заявки на камеры, от новых к старым, постранично; рамка карты и статус — фильтры. Публичный: contact не отдаётся. С заголовком admin-token — с contact (неверный токен — PERMISSION_DENIED).
     * @request POST:/carsearch.CarSearchService/ListCameraRequests
     */
    carSearchServiceListCameraRequests: (
      body: CarsearchListCameraRequestsRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceListCameraRequestsData, any>({
        path: `/carsearch.CarSearchService/ListCameraRequests`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceListClarifications
     * @summary ListClarifications — уточнения для админ-панели, от новых к старым, постранично, вместе с фиксацией. Нужен заголовок admin-token.
     * @request POST:/carsearch.CarSearchService/ListClarifications
     */
    carSearchServiceListClarifications: (
      body: CarsearchListClarificationsRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceListClarificationsData, any>({
        path: `/carsearch.CarSearchService/ListClarifications`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceListComplaints
     * @summary ListComplaints — жалобы для админ-панели, от новых к старым, постранично, вместе с фиксацией. Нужен заголовок admin-token (= ADMIN_TOKEN бэка), иначе PERMISSION_DENIED.
     * @request POST:/carsearch.CarSearchService/ListComplaints
     */
    carSearchServiceListComplaints: (
      body: CarsearchListComplaintsRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceListComplaintsData, any>({
        path: `/carsearch.CarSearchService/ListComplaints`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceListMyPayments
     * @summary ListMyPayments — история платежей пользователя, от новых к старым (последние 50). Нужен токен.
     * @request POST:/carsearch.CarSearchService/ListMyPayments
     */
    carSearchServiceListMyPayments: (
      body: CarsearchListMyPaymentsRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceListMyPaymentsData, any>({
        path: `/carsearch.CarSearchService/ListMyPayments`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceListPlateViews
     * @summary ListPlateViews — журнал открытий страниц номеров (rpc Get), от новых к старым (админ).
     * @request POST:/carsearch.CarSearchService/ListPlateViews
     */
    carSearchServiceListPlateViews: (
      body: CarsearchListPlateViewsRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceListPlateViewsData, any>({
        path: `/carsearch.CarSearchService/ListPlateViews`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceMe
     * @summary Me — кто я: is_admin — верный заголовок admin-token; authenticated и user — по токену пользователя (cookie cs_token или Authorization: Bearer). Нет токена или он неверный — authenticated = false.
     * @request POST:/carsearch.CarSearchService/Me
     */
    carSearchServiceMe: (
      body: CarsearchMeRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceMeData, any>({
        path: `/carsearch.CarSearchService/Me`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServicePlans
     * @summary Plans — тарифы: цена, период, состав.
     * @request POST:/carsearch.CarSearchService/Plans
     */
    carSearchServicePlans: (
      body: CarsearchPlansRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServicePlansData, any>({
        path: `/carsearch.CarSearchService/Plans`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceRegister
     * @summary Register — новый пользователь (тариф Free). Токен входа возвращается один раз — в базе только его sha256.
     * @request POST:/carsearch.CarSearchService/Register
     */
    carSearchServiceRegister: (
      body: CarsearchRegisterRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceRegisterData, any>({
        path: `/carsearch.CarSearchService/Register`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceResolveComplaint
     * @summary ResolveComplaint — отметить жалобу решённой. Нужен заголовок admin-token.
     * @request POST:/carsearch.CarSearchService/ResolveComplaint
     */
    carSearchServiceResolveComplaint: (
      body: CarsearchResolveComplaintRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceResolveComplaintData, any>({
        path: `/carsearch.CarSearchService/ResolveComplaint`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceRestoreSightingPhoto
     * @summary RestoreSightingPhoto — вернуть фото, скрытое по жалобам (photo_hidden = false); новые жалобы на фото этой фиксации (bad_photo, personal_data) отмечаются решёнными. Нужен admin-token.
     * @request POST:/carsearch.CarSearchService/RestoreSightingPhoto
     */
    carSearchServiceRestoreSightingPhoto: (
      body: CarsearchRestoreSightingPhotoRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceRestoreSightingPhotoData, any>({
        path: `/carsearch.CarSearchService/RestoreSightingPhoto`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceRevokeApiToken
     * @summary RevokeApiToken — отозвать токен (админ): дальше Get с ним отвечает PERMISSION_DENIED.
     * @request POST:/carsearch.CarSearchService/RevokeApiToken
     */
    carSearchServiceRevokeApiToken: (
      body: CarsearchRevokeApiTokenRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceRevokeApiTokenData, any>({
        path: `/carsearch.CarSearchService/RevokeApiToken`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceSearch
     * @summary Search ищет номера: по самому номеру и по уточнениям (clarifications) к его фиксациям. Один номер — одна строка (последняя подходящая фиксация). От новых к старым, постранично.
     * @request POST:/carsearch.CarSearchService/Search
     */
    carSearchServiceSearch: (
      body: CarsearchSearchRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceSearchData, any>({
        path: `/carsearch.CarSearchService/Search`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceSeeAlso
     * @summary SeeAlso — свежие фиксации других номеров (блок внизу страницы номера): один номер — одна карточка, номер открытой фиксации исключён. Не более 21.
     * @request POST:/carsearch.CarSearchService/SeeAlso
     */
    carSearchServiceSeeAlso: (
      body: CarsearchSeeAlsoRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceSeeAlsoData, any>({
        path: `/carsearch.CarSearchService/SeeAlso`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceSetCameraRequestStatus
     * @summary SetCameraRequestStatus — сменить статус заявки. Нужен заголовок admin-token.
     * @request POST:/carsearch.CarSearchService/SetCameraRequestStatus
     */
    carSearchServiceSetCameraRequestStatus: (
      body: CarsearchSetCameraRequestStatusRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceSetCameraRequestStatusData, any>({
        path: `/carsearch.CarSearchService/SetCameraRequestStatus`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceSetClarificationStatus
     * @summary SetClarificationStatus — принять или отклонить уточнение вручную. Нужен заголовок admin-token.
     * @request POST:/carsearch.CarSearchService/SetClarificationStatus
     */
    carSearchServiceSetClarificationStatus: (
      body: CarsearchSetClarificationStatusRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceSetClarificationStatusData, any>({
        path: `/carsearch.CarSearchService/SetClarificationStatus`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceSubmitCameraRequest
     * @summary SubmitCameraRequest — заявка на расположение камеры (страница /cameras). Создаётся со статусом NEW.
     * @request POST:/carsearch.CarSearchService/SubmitCameraRequest
     */
    carSearchServiceSubmitCameraRequest: (
      body: CarsearchSubmitCameraRequestRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceSubmitCameraRequestData, any>({
        path: `/carsearch.CarSearchService/SubmitCameraRequest`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceTokenInfo
     * @summary TokenInfo — остаток API-токена запроса (cookie cs_token или authorization: Bearer cst_…). Нет токена или он недействителен — UNAUTHENTICATED.
     * @request POST:/carsearch.CarSearchService/TokenInfo
     */
    carSearchServiceTokenInfo: (
      body: CarsearchTokenInfoRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceTokenInfoData, any>({
        path: `/carsearch.CarSearchService/TokenInfo`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @tags CarSearchService
     * @name CarSearchServiceVoteCameraRequest
     * @summary VoteCameraRequest — голос «+1» за заявку. Без авторизации; повторный голос с того же браузера отсекает фронт (localStorage).
     * @request POST:/carsearch.CarSearchService/VoteCameraRequest
     */
    carSearchServiceVoteCameraRequest: (
      body: CarsearchVoteCameraRequestRequest,
      params: RequestParams = {},
    ) =>
      this.request<CarSearchServiceVoteCameraRequestData, any>({
        path: `/carsearch.CarSearchService/VoteCameraRequest`,
        method: "POST",
        body: body,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
}
