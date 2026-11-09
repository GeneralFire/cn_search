// Самопроверка fe/src/searchkey.ts: таблица translate совпадает с plate_search_key из миграции,
// ключи и подсветка на парах «запрос → номер». Без зависимостей, вручную:
//   node fe/scripts/searchkey-check.ts   (Node 22.6+ — сам убирает типы TS)
import { readFileSync } from "node:fs";
import { strict as assert } from "node:assert";
import { matchRanges, plateSearchKey, queryWords, SEARCH_KEY_FROM, SEARCH_KEY_TO } from "../src/searchkey.ts";

const sql = readFileSync(new URL("../../be/migrations/0001_init.sql", import.meta.url), "utf8");
const m = sql.match(/plate_search_key\(s text\)[\s\S]*?translate\(s, '([^']*)', '([^']*)'\)/);
assert.ok(m, "в миграции не найден translate() в plate_search_key");
assert.equal(SEARCH_KEY_FROM, m[1], "FROM расходится с миграцией");
assert.equal(SEARCH_KEY_TO, m[2], "TO расходится с миграцией");

// Похожие символы: кириллица/латиница, О/0, дефис.
assert.equal(plateSearchKey("А123ВС 777"), "A123BC 777");
assert.equal(plateSearchKey("О515НХ 52"), "0515HX 52");
assert.equal(plateSearchKey("o515hx"), "0515hx");
assert.equal(plateSearchKey("AB-1234"), "AB 1234");
assert.deepEqual(queryWords("о12 77"), ["012", "77"]); // пример из tsQueryFrom
assert.deepEqual(queryWords("a123bc777"), ["a123bc777"]);
assert.deepEqual(queryWords("  У54!, "), ["y54"]);

// Подсветка: [начало, конец) в исходном номере.
assert.deepEqual(matchRanges("О515НХ 52", "о515"), [[0, 4]]);
assert.deepEqual(matchRanges("О515НХ 52", "O515 5"), [[0, 4], [7, 8]]);
assert.deepEqual(matchRanges("А123ВС 777", "a123bc 77"), [[0, 6], [7, 9]]);
assert.deepEqual(matchRanges("А123ВС 777", "123"), []); // только по началу слов
assert.deepEqual(matchRanges("M-AB 1234", "ab"), [[2, 4]]); // дефис — граница слова
assert.deepEqual(matchRanges("А123ВС 777", ""), []);

console.log("searchkey: ok");
