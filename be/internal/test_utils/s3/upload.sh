#!/bin/sh
# Заливка тестовых фото в S3: бакет cars с анонимным чтением, car-N.jpg и car-N_thumb.jpg.
# Запускается сервисом s3-init (образ amazon/aws-cli) при make etc; идемпотентен.
set -eu
aws() { command aws --endpoint-url "$S3_URL" "$@"; }

aws s3api head-bucket --bucket cars 2>/dev/null || aws s3api create-bucket --bucket cars
aws s3api put-bucket-policy --bucket cars --policy file:///seed/policy.json
aws s3 cp /seed/cars/ s3://cars/ --recursive --exclude '*' --include '*.jpg' \
  --content-type image/jpeg --cache-control 'public, max-age=86400' --only-show-errors
echo "s3: в бакете cars $(aws s3 ls s3://cars/ | wc -l | tr -d ' ') файлов"
