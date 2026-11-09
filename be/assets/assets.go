// Package assets содержит статику, вшитую в бинарник.
package assets

import "embed"

// Images — PNG-картинки машин: cars/car-1.png … cars/car-5.png.
// Генерируются скриптом fe/scripts/gen-car-images.mjs.
//
//go:embed cars/*.png
var Images embed.FS

// PhotoCount — сколько картинок лежит в cars/.
const PhotoCount = 5
