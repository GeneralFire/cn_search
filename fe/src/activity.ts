// Тепловая карта «Когда обычно появляется»: фиксации номера по дню недели и часу (локальная зона браузера).
import "./activity.css";
import type { Car } from "./api";
import { escapeHtml } from "./ui";
import { intlLocale, t, tp } from "./i18n/i18n";

const DAYS = 7; // строки: Пн…Вс
const HOURS = 24; // столбцы: 0…23
const TOP_PLACES = 3;
/** Геометрия в единицах viewBox: SVG масштабируется по ширине контейнера. */
const CELL = 12;
const GAP = 2;
const LEFT = 36; // место под подписи дней
const LABEL_H = 16; // место под подписи часов
const HOUR_LABELS = [0, 6, 12, 18];

const pad = (n: number) => String(n).padStart(2, "0");

/** Подписи дней недели с понедельника: опорная дата — понедельник 1 января 2024. */
function dayLabels(): string[] {
  const fmt = new Intl.DateTimeFormat(intlLocale(), { weekday: "short" });
  return Array.from({ length: DAYS }, (_, i) => fmt.format(new Date(2024, 0, 1 + i)));
}

/** Блок тепловой карты под историей; при одной фиксации (или меньше) — пустая строка. */
export function activityHtml(items: Car[]): string {
  if (items.length < 2) return "";

  const grid = Array.from({ length: DAYS }, () => new Array<number>(HOURS).fill(0));
  const places = new Map<string, number>();
  for (const c of items) {
    const d = new Date(c.createdAt);
    if (Number.isNaN(d.getTime())) continue;
    const day = (d.getDay() + 6) % 7; // getDay: 0 = Вс → сдвигаем, чтобы неделя начиналась с понедельника
    grid[day][d.getHours()]++;
    places.set(c.address, (places.get(c.address) ?? 0) + 1);
  }
  const max = Math.max(0, ...grid.flat());
  const days = dayLabels();

  const width = LEFT + HOURS * CELL;
  const height = DAYS * CELL + LABEL_H;
  const cells: string[] = [];
  const labels: string[] = [];
  for (let day = 0; day < DAYS; day++) {
    const y = day * CELL;
    labels.push(
      `<text class="act__label" x="${LEFT - 6}" y="${y + (CELL - GAP) / 2 + 3}" text-anchor="end">${escapeHtml(days[day])}</text>`
    );
    for (let h = 0; h < HOURS; h++) {
      const n = grid[day][h];
      const tip = `${days[day]}, ${pad(h)}:00–${pad(h + 1)}:00 — ${tp("activity.times", n)}`;
      // Насыщенность: пустые ячейки — var(--border), остальные — var(--accent) с прозрачностью по числу.
      const opacity = n ? (0.2 + (0.8 * n) / max).toFixed(2) : "1";
      cells.push(
        `<rect class="act__cell${n ? " is-on" : ""}" x="${LEFT + h * CELL}" y="${y}" width="${CELL - GAP}" height="${CELL - GAP}" rx="2" fill-opacity="${opacity}"><title>${escapeHtml(tip)}</title></rect>`
      );
    }
  }
  for (const h of HOUR_LABELS) {
    labels.push(
      `<text class="act__label" x="${LEFT + h * CELL}" y="${DAYS * CELL + 11}" text-anchor="start">${pad(h)}:00</text>`
    );
  }

  const top = [...places.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, TOP_PLACES)
    .map(([address, n]) => `<li>${escapeHtml(address)} — ${tp("activity.times", n)}</li>`)
    .join("");

  return `
    <section class="pp-activity" aria-labelledby="pp-activity-title">
      <h2 id="pp-activity-title">${t("activity.title")}</h2>
      <svg class="act" viewBox="0 0 ${width} ${height}" role="img" aria-label="${escapeHtml(t("activity.title"))}">
        ${cells.join("")}
        ${labels.join("")}
      </svg>
      <h3 class="act__subtitle">${t("activity.topPlaces")}</h3>
      <ol class="act__places">${top}</ol>
    </section>`;
}
