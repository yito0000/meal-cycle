package shopping

import "errors"

// エラーコードは Wails bindings 経由でフロントに届く。S07 はエラー種別で表示を分けない。
var ErrInvalidPeriodDays = errors.New("shopping: invalid period days")
