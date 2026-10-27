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

	// 常備品のコードは "staple" を挟み、在庫の部分文字列判定 ("name empty" 等) に誤って当たらないようにする。
	ErrStapleNameEmpty      = errors.New("fridge: staple name empty")
	ErrStapleNameTooLong    = errors.New("fridge: staple name too long")
	ErrStapleDuplicate      = errors.New("fridge: staple duplicate")
	ErrStapleInvalidState   = errors.New("fridge: staple invalid state")
	ErrStapleNotFound       = errors.New("fridge: staple not found")
	ErrStapleInvalidRestore = errors.New("fridge: staple invalid restore")

	ErrConsumeAmountOutOfRange = errors.New("fridge: consume amount out of range")
	ErrConsumeDuplicateStock   = errors.New("fridge: consume duplicate stock")
)
