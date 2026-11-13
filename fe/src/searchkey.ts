// Ключ поиска номера — JS-копия SQL-функции plate_search_key (be/migrations/0001_init.sql):
// похожие символы сводятся к одному (кириллица номера → латиница, О/o → 0, дефис и · → пробел).
// Нужен, чтобы подсветить в выпадающем списке то же совпадение, по которому нашёл бэк.
// Сверка с миграцией: `node fe/scripts/searchkey-check.ts`.

/** Аргументы translate() из plate_search_key — менять вместе с миграцией. */
export const SEARCH_KEY_FROM = "АВЕКМНОРСТУХавекмнорстухOo-Іі·";
export const SEARCH_KEY_TO = "ABEKMH0PCTYXabekmh0pctyx00 Ii ";

const MAP = new Map([...SEARCH_KEY_FROM].map((c, i) => [c, SEARCH_KEY_TO[i]]));

/** plate_search_key(s): посимвольная замена, длина строки сохраняется. */
export function plateSearchKey(s: string): string {
  let out = "";
  for (const c of s) out += MAP.get(c) ?? c;
  return out;
}

/** Слова запроса так же, как в tsQueryFrom (repository/postgres.go): ключ, только буквы/цифры/пробелы,
 *  разбивка по пробелам; плюс нижний регистр — tsvector 'simple' регистр не различает. */
export function queryWords(q: string): string[] {
  return plateSearchKey(q)
    .replace(/[^\p{L}\p{N}\s]/gu, "")
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean);
}

/** Диапазоны [начало, конец) в plate, совпавшие с запросом по началу слов (поиск «по началу слов»):
 *  в каждом слове номера — самое длинное слово запроса, с которого оно начинается. */
export function matchRanges(plate: string, q: string): [number, number][] {
  const words = queryWords(q);
  if (!words.length) return [];
  // Ключ посимвольный, поэтому индексы в ключе = индексы в исходном номере.
  const chars = [...plateSearchKey(plate)].map((c) => c.toLowerCase());
  const ranges: [number, number][] = [];
  let i = 0;
  while (i < chars.length) {
    if (/\s/.test(chars[i])) {
      i++;
      continue;
    }
    let j = i;
    while (j < chars.length && !/\s/.test(chars[j])) j++;
    const word = chars.slice(i, j).join("");
    const best = Math.max(0, ...words.filter((w) => word.startsWith(w)).map((w) => w.length));
    if (best) ranges.push([i, i + best]);
    i = j;
  }
  return ranges;
}

/** Части текста номера в разметке plateHtml (без «RUS», штата, звёзд ЕС). */
const PLATE_TEXT = ".plate__num, .plate__reg, .plate-eu__num, .plate-us__num, .plate-any";

/** Обернуть в <mark> совпадение с запросом внутри отрисованного номера (plateHtml из ui.ts).
 *  Номер в разметке разбит на части без пробелов (серия, регион), поэтому символы сопоставляются
 *  с исходной строкой номера по порядку, пропуская пробелы. */
export function highlightPlate(root: Element, plate: string, q: string, markClass = "qs__mark"): void {
  const ranges = matchRanges(plate, q);
  if (!ranges.length) return;
  const chars = [...plate];
  const marked = (idx: number) => ranges.some(([a, b]) => idx >= a && idx < b);
  let pi = 0; // позиция в plate
  const nodes: Text[] = [];
  for (const part of root.querySelectorAll(PLATE_TEXT)) {
    const walker = document.createTreeWalker(part, NodeFilter.SHOW_TEXT);
    // В .plate__reg вложен «RUS» — берём только прямые текстовые узлы частей.
    for (let n = walker.nextNode(); n; n = walker.nextNode()) if (n.parentElement === part) nodes.push(n as Text);
  }
  for (const node of nodes) {
    const text = [...(node.data)];
    const frag = document.createDocumentFragment();
    let buf = "";
    let bufMarked = false;
    const flush = () => {
      if (!buf) return;
      if (bufMarked) {
        const m = document.createElement("mark");
        m.className = markClass;
        m.textContent = buf;
        frag.append(m);
      } else frag.append(buf);
      buf = "";
    };
    for (const c of text) {
      while (pi < chars.length && /\s/.test(chars[pi])) pi++;
      const isMarked = !/\s/.test(c) && pi < chars.length && marked(pi);
      if (!/\s/.test(c)) pi++;
      if (isMarked !== bufMarked) flush();
      bufMarked = isMarked;
      buf += c;
    }
    flush();
    node.replaceWith(frag);
  }
}
