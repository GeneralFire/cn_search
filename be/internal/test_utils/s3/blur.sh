#!/bin/sh
# Обработка тестовых фото: номерная зона замазана (пикселизация + размытие), в левом верхнем
# углу — полупрозрачная метка «TEST PHOTO». Номер на снимке не совпадает с номером фиксации
# в БД, поэтому номера скрыты, а фото честно помечены как тестовые.
#
# Воспроизводимо из оригиналов: по умолчанию берёт их из git (коммит ORIG_REV, где фото
# лежат без обработки), либо из каталога, переданного первым аргументом (car-N.jpg и
# car-N_thumb.jpg — например, заново скачанные по ссылкам из cars/CREDITS.md и уменьшенные).
# Результат пишется в cars/ рядом со скриптом. Нужен только docker (ImageMagick с Docker Hub).
#
#   ./blur.sh            # оригиналы из git
#   ./blur.sh /path/orig # оригиналы из каталога
set -eu

ORIG_REV=${ORIG_REV:-4496f56}
IMAGE=${IMAGE:-dpokidov/imagemagick:7.1.1-47}
here=$(cd "$(dirname "$0")" && pwd)
out="$here/cars"
work=$(mktemp -d "$here/.blur.XXXXXX")  # внутри $HOME — его видит docker (Colima)
trap 'rm -rf "$work"' EXIT

# Зоны номеров в координатах полного снимка шириной 1600 px: «x y w h» через «;».
# Для миниатюр (480 px) пересчитываются пропорционально. Пусто — номера на снимке нет.
zones() {
  case $1 in
    1) echo "1295 662 190 88" ;;
    2) echo "1205 518 200 76" ;;
    3) echo "1188 688 296 150" ;;
    4) echo "1285 685 260 230" ;;  # рамка дилера на месте номера
    5) echo "140 636 210 98" ;;
    6) echo "120 652 130 70" ;;
    7) echo "1205 440 265 175" ;;
    8) echo "" ;;                  # номер в кадр не попал
    9) echo "1225 410 255 155" ;;
    10) echo "122 675 212 120" ;;
  esac
}

for n in 1 2 3 4 5 6 7 8 9 10; do
  for suffix in "" _thumb; do
    f="car-$n$suffix.jpg"
    if [ $# -ge 1 ]; then
      cp "$1/$f" "$work/$f"
    else
      git -C "$here" show "$ORIG_REV:./cars/$f" >"$work/$f"
    fi
    if [ -z "$suffix" ]; then k=1; else k=0.3; fi  # 480 / 1600

    # Аргументы ImageMagick: каждая зона — вырезать, пикселизовать, размыть, вклеить обратно.
    args=""
    zs=$(zones "$n")
    if [ -n "$zs" ]; then
      x=$(echo "$zs" | awk -v k="$k" '{printf "%d", $1*k}')
      y=$(echo "$zs" | awk -v k="$k" '{printf "%d", $2*k}')
      w=$(echo "$zs" | awk -v k="$k" '{printf "%d", $3*k}')
      h=$(echo "$zs" | awk -v k="$k" '{printf "%d", $4*k}')
      args="( +clone -crop ${w}x${h}+${x}+${y} +repage -scale $(awk -v k="$k" 'BEGIN{printf "%g", 5/k}')% -scale ${w}x${h}! -blur 0x$(awk -v k="$k" 'BEGIN{printf "%g", 6*k}') ) -geometry +${x}+${y} -composite"
    fi

    # Метка: полупрозрачная тёмная плашка с белым текстом в левом верхнем углу.
    pt=$(awk -v k="$k" 'BEGIN{printf "%d", 34*k+0.5}')
    m=$(awk -v k="$k" 'BEGIN{printf "%d", 24*k+0.5}')
    pad=$(awk -v k="$k" 'BEGIN{printf "%d", 12*k+0.5}')
    # shellcheck disable=SC2086
    docker run --rm -v "$work:/w" -w /w "$IMAGE" "$f" $args \
      \( -background '#0008' -fill '#fffd' -font DejaVu-Sans-Bold -pointsize "$pt" \
         label:'TEST PHOTO' -bordercolor '#0008' -border "${pad}x$((pad / 2))" \) \
      -gravity northwest -geometry "+$m+$m" -composite \
      -strip -interlace JPEG -sampling-factor 4:2:0 -quality 92 "out-$f"
    cp "$work/out-$f" "$out/$f"
    echo "$f"
  done
done
