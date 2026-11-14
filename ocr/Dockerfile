# Сервис распознавания номеров (fast-alpr: YOLO-детектор номеров + OCR, ONNX, CPU).
FROM python:3.12-slim
ENV PIP_INDEX_URL=https://pypi.org/simple \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PYTHONUNBUFFERED=1
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
# Детектор лиц YuNet (opencv_zoo, Apache-2.0, ~230 КБ): публичный GitHub, сумма проверяется при сборке.
RUN mkdir -p /app/models && python -c "import hashlib,urllib.request as u; d=u.urlopen('https://github.com/opencv/opencv_zoo/raw/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx').read(); assert hashlib.sha256(d).hexdigest()=='8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4'; open('models/face_detection_yunet_2023mar.onnx','wb').write(d)"
RUN useradd -m -u 10001 app
USER app
COPY --chown=app server.py .
# Веса моделей (публичные релизы GitHub / Hugging Face) скачиваются при сборке и остаются в образе.
RUN python -c "import server; server.load()"
EXPOSE 8000
HEALTHCHECK --interval=5s --timeout=3s --retries=20 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/health')" || exit 1
CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8000"]
