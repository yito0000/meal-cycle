package fridge

import "errors"

// エラーコードは Wails bindings 経由でフロントに届き、i18n key にマップされる。
// フロントは "fridge: " 以降の文言で判定するので、変更時は error-messages.ts も直す。
var (
	ErrStockNameEmpty        = errors.New("fridge: name empty")
	ErrStockNameTooLong      = errors.New("fridge: name too long")
	ErrStockAmountOutOfRange = errors.New("fridge: amount out of range")
	ErrStockUnitTooLong      = errors.New("fridge: unit too long")
	ErrStockInvalidExpiry    = errors.New("fridge: invalid expiry date")
	ErrStockInvalidRestore   = errors.New("fridge: invalid restore")
	ErrStockNotFound         = errors.New("fridge: stock item not found")
	ErrStockAlreadyExists    = errors.New("fridge: stock item exists")
)
