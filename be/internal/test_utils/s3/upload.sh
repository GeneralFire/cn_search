#!/bin/sh
# Заливка тестовых картинок в S3: бакет cars с анонимным чтением, car-1..5.png.
# Запускается сервисом sample-data-s3 (образ amazon/aws-cli), см. docker-compose.yml.
set -eu
aws() { command aws --endpoint-url "$S3_URL" "$@"; }

aws s3api head-bucket --bucket cars 2>/dev/null || aws s3api create-bucket --bucket cars
aws s3api put-bucket-policy --bucket cars --policy file:///seed/policy.json
aws s3 cp /seed/cars/ s3://cars/ --recursive --exclude '*' --include '*.png' \
  --content-type image/png --cache-control 'public, max-age=86400' --only-show-errors
echo "s3: в бакете cars $(aws s3 ls s3://cars/ | wc -l | tr -d ' ') картинок"
