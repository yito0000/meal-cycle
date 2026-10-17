package settings

import "errors"

// フロントは "settings: " 以降の文言で欄とメッセージを引くので、変更時は error-messages.ts も直す。
var (
	ErrInvalidLanguage      = errors.New("settings: invalid language")
	ErrRedDaysOutOfRange    = errors.New("settings: red days out of range")
	ErrYellowDaysOutOfRange = errors.New("settings: yellow days out of range")
	ErrRedNotBelowYellow    = errors.New("settings: red days not below yellow days")

	ErrLeftoverPresetsCount     = errors.New("settings: leftover presets count")
	ErrLeftoverPresetOutOfRange = errors.New("settings: leftover preset out of range")
	ErrLeftoverPresetDuplicate  = errors.New("settings: leftover preset duplicate")

	ErrShoppingPresetsCount          = errors.New("settings: shopping presets count")
	ErrShoppingPresetOutOfRange      = errors.New("settings: shopping preset out of range")
	ErrShoppingPresetDuplicate       = errors.New("settings: shopping preset duplicate")
	ErrShoppingPresetRequiredMissing = errors.New("settings: shopping preset required missing")
	ErrShoppingDefaultNotInPresets   = errors.New("settings: shopping default not in presets")

	ErrMeasuringUnitsCount    = errors.New("settings: measuring units count")
	ErrMeasuringUnitLength    = errors.New("settings: measuring unit length")
	ErrMeasuringUnitDuplicate = errors.New("settings: measuring unit duplicate")

	ErrUnquantifiedWordsCount    = errors.New("settings: unquantified words count")
	ErrUnquantifiedWordLength    = errors.New("settings: unquantified word length")
	ErrUnquantifiedWordDuplicate = errors.New("settings: unquantified word duplicate")

	ErrImportServingsOutOfRange = errors.New("settings: import servings out of range")
)
