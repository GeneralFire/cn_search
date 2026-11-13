"""Сервис распознавания госномеров: POST /recognize (multipart, поле image) → JSON-список кандидатов.

Модель — fast-alpr: детектор номеров YOLOv9 (open-image-models) + OCR номера (fast-plate-ocr),
обе в ONNX, работают на CPU. Модель выдаёт латиницу и цифры; приведение к формату страны
(например, кириллица для номеров РФ) делает бэкенд.
"""

import os

import cv2
import numpy as np
from fastapi import FastAPI, File, HTTPException, UploadFile

DETECTOR_MODEL = os.environ.get("OCR_DETECTOR_MODEL", "yolo-v9-t-384-license-plate-end2end")
OCR_MODEL = os.environ.get("OCR_MODEL", "cct-xs-v2-global-model")
MAX_IMAGE_BYTES = 10 * 1024 * 1024

_alpr = None


def load():
    """Загружает модели (при первом вызове скачивает веса в кэш)."""
    global _alpr
    if _alpr is None:
        from fast_alpr import ALPR

        _alpr = ALPR(detector_model=DETECTOR_MODEL, ocr_model=OCR_MODEL)
    return _alpr


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
