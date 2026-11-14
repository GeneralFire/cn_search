"""Сервис распознавания госномеров: POST /recognize (multipart, поле image) → JSON-список кандидатов.

Также POST /anonymize — размытие лиц (YuNet) перед сохранением фото.

Модель — fast-alpr: детектор номеров YOLOv9 (open-image-models) + OCR номера (fast-plate-ocr),
обе в ONNX, работают на CPU. Модель выдаёт латиницу и цифры; приведение к формату страны
(например, кириллица для номеров РФ) делает бэкенд.
"""

import os

import cv2
import numpy as np
from fastapi import FastAPI, File, HTTPException, UploadFile
from fastapi.responses import Response

DETECTOR_MODEL = os.environ.get("OCR_DETECTOR_MODEL", "yolo-v9-t-384-license-plate-end2end")
OCR_MODEL = os.environ.get("OCR_MODEL", "cct-xs-v2-global-model")
MAX_IMAGE_BYTES = 10 * 1024 * 1024

# Детектор лиц YuNet (opencv_zoo, ONNX, ~230 КБ): веса кладутся в образ при сборке (см. Dockerfile).
FACE_MODEL = os.environ.get("FACE_MODEL", "/app/models/face_detection_yunet_2023mar.onnx")
FACE_SCORE = float(os.environ.get("FACE_SCORE", "0.5"))
FACE_MARGIN = 0.2  # запас вокруг рамки лица с каждой стороны, доля от размера рамки
FACE_DETECT_SIDE = 1920  # детектор видит картинку не крупнее этого по длинной стороне
JPEG_QUALITY = 92

_alpr = None


def load():
    """Загружает модели (при первом вызове скачивает веса в кэш)."""
    global _alpr
    if _alpr is None:
        from fast_alpr import ALPR

        _alpr = ALPR(detector_model=DETECTOR_MODEL, ocr_model=OCR_MODEL)
    return _alpr


def _face_detector(w, h):
    """YuNet с входом под размер картинки (объект не потокобезопасен — создаём на запрос, это дёшево)."""
    return cv2.FaceDetectorYN.create(FACE_MODEL, "", (w, h), FACE_SCORE, 0.3, 5000)


def blur_faces(frame):
    """Размывает лица гауссом (рамка + запас 20%). Возвращает (картинка, число лиц). Номера не трогает."""
    h, w = frame.shape[:2]
    k = min(1.0, FACE_DETECT_SIDE / max(h, w))
    small = cv2.resize(frame, (round(w * k), round(h * k)), interpolation=cv2.INTER_AREA) if k < 1 else frame
    det = _face_detector(small.shape[1], small.shape[0])
    _, found = det.detect(small)
    if found is None:
        return frame, 0
    out = frame.copy()
    n = 0
    for f in found:
        x, y, bw, bh = (float(v) / k for v in f[:4])
        x0 = max(0, int(x - bw * FACE_MARGIN))
        y0 = max(0, int(y - bh * FACE_MARGIN))
        x1 = min(w, int(x + bw * (1 + FACE_MARGIN)))
        y1 = min(h, int(y + bh * (1 + FACE_MARGIN)))
        if x1 - x0 < 2 or y1 - y0 < 2:
            continue
        # Ядро пропорционально лицу: даже крупное лицо превращается в пятно.
        ks = max(3, int(max(x1 - x0, y1 - y0) / 3) | 1)
        out[y0:y1, x0:x1] = cv2.GaussianBlur(out[y0:y1, x0:x1], (ks, ks), 0)
        n += 1
    return out, n


def _confidence(c):
    # Уверенность OCR бывает числом или списком по символам — берём среднее.
    if c is None:
        return 0.0
    if isinstance(c, (list, tuple, np.ndarray)):
        return float(np.mean(c)) if len(c) else 0.0
    return float(c)


app = FastAPI(title="car-search OCR")


@app.on_event("startup")
def _startup():
    load()


@app.get("/health")
def health():
    return {"status": "ok"}


@app.post("/anonymize")
async def anonymize(image: UploadFile = File(...)):
    """Фото → JPEG с размытыми лицами; число лиц — в заголовке X-Faces-Blurred."""
    data = await image.read(MAX_IMAGE_BYTES + 1)
    if len(data) > MAX_IMAGE_BYTES:
        raise HTTPException(413, "image too large")
    frame = cv2.imdecode(np.frombuffer(data, np.uint8), cv2.IMREAD_COLOR)
    if frame is None:
        raise HTTPException(400, "not an image")
    out, n = blur_faces(frame)
    ok, buf = cv2.imencode(".jpg", out, [cv2.IMWRITE_JPEG_QUALITY, JPEG_QUALITY])
    if not ok:
        raise HTTPException(500, "encode failed")
    return Response(buf.tobytes(), media_type="image/jpeg", headers={"X-Faces-Blurred": str(n)})


@app.post("/recognize")
async def recognize(image: UploadFile = File(...)):
    data = await image.read(MAX_IMAGE_BYTES + 1)
    if len(data) > MAX_IMAGE_BYTES:
        raise HTTPException(413, "image too large")
    # IMREAD_COLOR учитывает EXIF-ориентацию (фото с телефона).
    frame = cv2.imdecode(np.frombuffer(data, np.uint8), cv2.IMREAD_COLOR)
    if frame is None:
        raise HTTPException(400, "not an image")
    out = []
    for r in load().predict(frame):
        if r.ocr is None:
            continue
        text = "".join(ch for ch in (r.ocr.text or "").upper() if ch.isalnum())
        if not text:
            continue
        bb = r.detection.bounding_box
        det_conf = float(r.detection.confidence)
        ocr_conf = _confidence(r.ocr.confidence)
        out.append({
            "plate": text,
            # Итоговая уверенность — OCR с поправкой на уверенность детектора.
            "confidence": round(ocr_conf * min(1.0, det_conf + 0.2), 4),
            "ocr_confidence": round(ocr_conf, 4),
            "detection_confidence": round(det_conf, 4),
            "bbox": [int(bb.x1), int(bb.y1), int(bb.x2), int(bb.y2)],
        })
    out.sort(key=lambda c: c["confidence"], reverse=True)
    return out
