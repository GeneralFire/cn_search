// Генерирует 5 PNG-картинок машин (1280×720) в be/assets/cars/.
// Сцена описывается SVG-разметкой в памяти и растеризуется через resvg — на диск пишется только PNG.
// Запуск: node fe/scripts/gen-car-images.mjs
// Бэкенд вшивает их в бинарник (go:embed) и отдаёт по /images/cars/car-N.png
import { mkdirSync, writeFileSync } from "node:fs";
import { Resvg } from "@resvg/resvg-js";

const PNG_WIDTH = 1280;

const outDir = new URL("../../be/assets/cars/", import.meta.url);
mkdirSync(outDir, { recursive: true });

const W = 640;
const H = 360;
const GROUND = 300;

// ---------------------------------------------------------------- фоны

const road = (asphalt, line = "#f4f4f4") => `
  <rect y="${GROUND}" width="${W}" height="${H - GROUND}" fill="${asphalt}"/>
  <rect y="${GROUND}" width="${W}" height="4" fill="#000" opacity=".15"/>
  ${Array.from({ length: 8 }, (_, i) => `<rect x="${i * 90 - 20}" y="${GROUND + 34}" width="50" height="5" rx="2" fill="${line}" opacity=".7"/>`).join("")}`;

const sky = (top, bottom) => `
  <defs><linearGradient id="sky" x1="0" y1="0" x2="0" y2="1">
    <stop offset="0" stop-color="${top}"/><stop offset="1" stop-color="${bottom}"/>
  </linearGradient></defs>
  <rect width="${W}" height="${H}" fill="url(#sky)"/>`;

const cloud = (x, y, s = 1) => `
  <g fill="#fff" opacity=".9" transform="translate(${x} ${y}) scale(${s})">
    <ellipse cx="0" cy="0" rx="34" ry="16"/><ellipse cx="28" cy="-10" rx="26" ry="18"/><ellipse cx="54" cy="2" rx="28" ry="14"/>
  </g>`;

const scenes = {
  day: () => `
    ${sky("#8ec5ff", "#e6f2ff")}
    <circle cx="540" cy="70" r="34" fill="#fff4c2"/>
    ${cloud(90, 70)}${cloud(330, 50, 0.8)}
    <path d="M0 300 L0 250 Q120 215 240 245 T480 238 T640 246 L640 300 Z" fill="#9fd39b"/>
    ${road("#4a4f57")}`,

  sunset: () => `
    ${sky("#ff8a65", "#ffd59e")}
    <circle cx="150" cy="235" r="62" fill="#fff1c9"/>
    <path d="M0 300 L0 230 Q90 190 200 228 T420 215 T640 225 L640 300 Z" fill="#8a5a8c"/>
    <path d="M0 300 L0 262 Q160 236 320 258 T640 255 L640 300 Z" fill="#6b4470"/>
    ${road("#3d3540", "#ffd59e")}`,

  mountains: () => `
    ${sky("#b8e3ff", "#eef9ff")}
    <path d="M-20 300 L120 120 L250 300 Z" fill="#7d97a8"/>
    <path d="M120 120 L95 152 L112 148 L128 160 L146 150 Z" fill="#fff"/>
    <path d="M160 300 L360 90 L560 300 Z" fill="#62808f"/>
    <path d="M360 90 L322 130 L344 126 L362 140 L384 128 L398 132 Z" fill="#fff"/>
    <path d="M430 300 L560 160 L680 300 Z" fill="#7d97a8"/>
    <path d="M0 300 L0 268 Q200 250 400 266 T640 262 L640 300 Z" fill="#6fae74"/>
    ${road("#50565e")}`,

  desert: () => `
    ${sky("#ffcf8a", "#fff3dc")}
    <circle cx="500" cy="95" r="44" fill="#fff8e6" opacity=".9"/>
    <path d="M0 300 L0 240 Q140 200 300 238 T640 228 L640 300 Z" fill="#e8b46b"/>
    <path d="M0 300 L0 266 Q220 246 420 264 T640 258 L640 300 Z" fill="#d79c52"/>
    <g fill="#5f8f5a"><rect x="78" y="196" width="12" height="60" rx="6"/><rect x="62" y="214" width="10" height="26" rx="5"/><rect x="62" y="234" width="20" height="8" rx="4"/><rect x="96" y="206" width="10" height="24" rx="5"/><rect x="86" y="224" width="20" height="8" rx="4"/></g>
    ${road("#6b5a48", "#fff3dc")}`,

  night: () => {
    const stars = Array.from({ length: 40 }, (_, i) => {
      const x = (i * 157) % W, y = (i * 61) % 150 + 8, r = i % 3 === 0 ? 1.6 : 1;
      return `<circle cx="${x}" cy="${y}" r="${r}" fill="#fff" opacity="${0.4 + (i % 5) / 10}"/>`;
    }).join("");
    const buildings = [[0, 160, 70], [70, 200, 55], [125, 140, 80], [205, 185, 60], [265, 120, 75], [340, 175, 65], [405, 150, 70], [475, 195, 50], [525, 135, 70], [595, 180, 60]]
      .map(([x, y, w], bi) => {
        let win = "";
        for (let wy = y + 14; wy < 290; wy += 22)
          for (let wx = x + 10; wx < x + w - 12; wx += 18)
            if ((wx * 7 + wy * 13 + bi) % 5 < 2) win += `<rect x="${wx}" y="${wy}" width="8" height="10" fill="#ffd866" opacity=".85"/>`;
        return `<rect x="${x}" y="${y}" width="${w}" height="${GROUND - y}" fill="#111836"/>${win}`;
      })
      .join("");
    return `
      ${sky("#0b1533", "#3a4f8a")}
      ${stars}
      <circle cx="560" cy="62" r="26" fill="#f2f0d8"/><circle cx="572" cy="54" r="24" fill="#0f1a3c"/>
      ${buildings}
      ${road("#22252c", "#ffd866")}`;
  },
};

// ---------------------------------------------------------------- машины

/**
 * body — силуэт кузова (низ — по оси колёс), windows — стёкла,
 * pillar — x стойки/двери, wheels — x центров колёс, r — радиус колеса.
 */
const cars = [
  {
    name: "sedan", scene: "day", color: "#e0393e", dark: "#8f1d22",
    body: "M108 268 L108 222 Q110 204 132 199 L205 190 L256 151 Q266 144 282 143 L382 143 Q396 143 406 151 L462 190 L522 199 Q542 203 544 222 L544 268 Z",
    windows: ["M266 156 L322 156 L322 189 L222 191 Z", "M332 156 L378 156 Q390 156 398 163 L440 189 L332 189 Z"],
    pillar: 327, wheels: [200, 450], r: 36,
  },
  {
    name: "hatchback", scene: "sunset", color: "#ffc531", dark: "#b8820c",
    body: "M146 268 L146 205 Q148 178 172 168 L214 140 Q224 134 240 134 L360 134 Q373 134 383 142 L432 186 L500 196 Q519 200 521 218 L521 268 Z",
    windows: ["M178 176 L222 147 Q228 144 238 144 L292 144 L292 186 L170 188 Q170 180 178 176 Z", "M302 144 L358 144 Q368 144 375 151 L412 186 L302 186 Z"],
    pillar: 297, wheels: [210, 452], r: 35,
  },
  {
    name: "suv", scene: "mountains", color: "#2f6e57", dark: "#173b2e",
    body: "M106 268 L106 192 Q106 176 122 173 L150 170 L176 120 Q181 111 196 111 L402 111 Q414 111 422 120 L462 168 L526 177 Q546 181 546 201 L546 268 Z",
    windows: ["M186 124 L262 124 L262 168 L160 170 Z", "M272 124 L346 124 L346 168 L272 168 Z", "M356 124 L398 124 Q408 124 414 131 L444 168 L356 168 Z"],
    pillar: 351, wheels: [196, 450], r: 40,
  },
  {
    name: "pickup", scene: "desert", color: "#2f6bff", dark: "#173a99",
    body: "M92 268 L92 206 L282 206 L282 192 L300 136 Q304 128 316 128 L392 128 Q402 128 410 136 L448 186 L526 193 Q546 197 546 216 L546 268 Z",
    windows: ["M306 140 L346 140 L346 186 L292 188 Z", "M356 140 L390 140 Q398 140 404 147 L432 186 L356 186 Z"],
    pillar: 351, wheels: [178, 452], r: 39,
    extra: `<rect x="92" y="198" width="190" height="10" rx="3" fill="#173a99"/>`,
  },
  {
    name: "minivan", scene: "night", color: "#e9edf2", dark: "#9aa3ae",
    body: "M98 268 L98 172 Q98 140 130 133 L380 122 Q401 122 416 133 L482 181 L532 189 Q550 193 550 212 L550 268 Z",
    windows: ["M120 146 Q122 142 134 141 L210 138 L210 182 L114 184 Z", "M220 138 L300 135 L300 182 L220 182 Z", "M310 134 L378 132 Q392 132 402 140 L452 182 L310 182 Z"],
    pillar: 305, wheels: [190, 456], r: 37,
  },
];

function carSvg(c) {
  const [w1, w2] = c.wheels;
  const wy = 268 - 4; // центр колеса чуть выше низа кузова
  const wheel = (x) => `
    <g transform="translate(${x} ${wy})">
      <circle r="${c.r}" fill="#1b1d22"/>
      <circle r="${c.r * 0.62}" fill="#c9ced6"/>
      <circle r="${c.r * 0.46}" fill="#8e96a1"/>
      ${[0, 72, 144, 216, 288].map((a) => `<rect x="-2.5" y="${-c.r * 0.6}" width="5" height="${c.r * 0.6}" rx="2" fill="#c9ced6" transform="rotate(${a})"/>`).join("")}
      <circle r="${c.r * 0.14}" fill="#444a52"/>
    </g>`;
  const front = Math.max(...c.body.match(/\d+(?=\s)/g).map(Number).filter((n) => n > 400 && n < 600));
  const rear = Math.min(...c.body.match(/M(\d+)/).slice(1).map(Number));

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${W} ${H}" role="img" aria-label="${c.name}">
  ${scenes[c.scene]()}
  <defs>
    <linearGradient id="paint" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="${c.color}"/><stop offset=".7" stop-color="${c.color}"/><stop offset="1" stop-color="${c.dark}"/>
    </linearGradient>
    <linearGradient id="glass" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#cfe6ff"/><stop offset="1" stop-color="#6d8fb3"/>
    </linearGradient>
    <clipPath id="b"><path d="${c.body}"/></clipPath>
  </defs>
  <ellipse cx="${(rear + front) / 2}" cy="${GROUND + 2}" rx="${(front - rear) / 2 + 14}" ry="12" fill="#000" opacity=".28"/>
  ${c.extra ?? ""}
  <path d="${c.body}" fill="url(#paint)"/>
  <g clip-path="url(#b)">
    <rect x="0" y="214" width="${W}" height="3" fill="#fff" opacity=".28"/>
    <rect x="${c.pillar - 1}" y="150" width="2" height="110" fill="${c.dark}" opacity=".45"/>
    <circle cx="${w1}" cy="${wy}" r="${c.r + 7}" fill="#111"/>
    <circle cx="${w2}" cy="${wy}" r="${c.r + 7}" fill="#111"/>
    <rect x="${front - 26}" y="204" width="24" height="12" rx="5" fill="#fff6c9"/>
    <rect x="${rear}" y="206" width="12" height="16" rx="3" fill="#ff3b30"/>
    <rect x="0" y="250" width="${W}" height="20" fill="#000" opacity=".18"/>
  </g>
  ${c.windows.map((d) => `<path d="${d}" fill="url(#glass)" stroke="${c.dark}" stroke-width="2" stroke-linejoin="round"/>`).join("")}
  <rect x="${c.pillar + 18}" y="200" width="16" height="4" rx="2" fill="${c.dark}"/>
  ${wheel(w1)}${wheel(w2)}
</svg>
`;
}

cars.forEach((c, i) => {
  const png = new Resvg(carSvg(c), { fitTo: { mode: "width", value: PNG_WIDTH } }).render().asPng();
  writeFileSync(new URL(`car-${i + 1}.png`, outDir), png);
  console.log(`car-${i + 1}.png  ${c.name.padEnd(10)} ${c.scene.padEnd(10)} ${(png.length / 1024).toFixed(0)} KB`);
});
