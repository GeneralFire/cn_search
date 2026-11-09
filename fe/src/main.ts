import "./style.css";
import { cars, type Car } from "./data";

type SearchBy = "plate" | "key";

const modeLabels: Record<SearchBy, string> = {
  plate: "Номер",
  key: "Ключ",
};

let mode: SearchBy = "plate";
let query = "";

const app = document.querySelector<HTMLDivElement>("#app")!;

app.innerHTML = `
  <main class="page">
    <section class="search">
      <h1 class="search__title">Поиск по авто</h1>
      <div class="search__box">
        <div class="search__modes" role="radiogroup" aria-label="Искать по">
          ${(Object.keys(modeLabels) as SearchBy[])
            .map(
              (m) => `
            <label class="search__mode">
              <input type="radio" name="mode" value="${m}" ${m === mode ? "checked" : ""} />
              <span>${modeLabels[m]}</span>
            </label>`
            )
            .join("")}
        </div>
        <input class="search__input" type="search" autocomplete="off" />
      </div>
    </section>
    <section class="results" aria-live="polite"></section>
  </main>
`;

const input = app.querySelector<HTMLInputElement>(".search__input")!;
const results = app.querySelector<HTMLElement>(".results")!;

function formatDate(iso: string): string {
  // timeZone: "UTC" — иначе "2026-11-02" в западных поясах покажется как 01.11
  return new Date(iso).toLocaleDateString("ru-RU", { timeZone: "UTC" });
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!
  );
}

function renderCard(car: Car): string {
  const photo = car.photo
    ? `<img src="${escapeHtml(car.photo)}" alt="Фото авто ${escapeHtml(car.plate)}" />`
    : "";
  return `
    <article class="card">
      <div class="card__photo">${photo}</div>
      <p class="card__address">${escapeHtml(car.address)}</p>
      <p class="card__meta">
        <span class="card__plate">${escapeHtml(car.plate)}</span>,
        <time datetime="${escapeHtml(car.date)}">${formatDate(car.date)}</time>
      </p>
    </article>
  `;
}

// Буквы госномера, совпадающие по написанию с латиницей: "A123BC" найдёт "А123ВС".
const latToCyr: Record<string, string> = {
  A: "А", B: "В", E: "Е", K: "К", M: "М", H: "Н", O: "О", P: "Р", C: "С", T: "Т", Y: "У", X: "Х",
};

function normalize(s: string, by: SearchBy): string {
  const r = s.toUpperCase().replace(/\s+/g, "");
  return by === "plate" ? r.replace(/[ABEKMHOPCTYX]/g, (c) => latToCyr[c]!) : r;
}

function render(): void {
  input.placeholder = mode === "plate" ? "Например, А123ВС 777" : "Например, KEY-0451";
  const q = normalize(query, mode);
  const found = cars.filter((c) => normalize(mode === "plate" ? c.plate : c.key, mode).includes(q));
  results.innerHTML = found.length
    ? found.map(renderCard).join("")
    : `<p class="results__empty">Ничего не найдено. Проверьте ${
        mode === "plate" ? "номер" : "ключ"
      } или переключите режим поиска.</p>`;
}

app.querySelectorAll<HTMLInputElement>('input[name="mode"]').forEach((radio) =>
  radio.addEventListener("change", () => {
    mode = radio.value as SearchBy;
    render();
    input.focus();
  })
);

input.addEventListener("input", () => {
  query = input.value;
  render();
});

render();
