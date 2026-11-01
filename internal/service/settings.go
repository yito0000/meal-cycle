package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/settings"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// configSource は他サービスが現在の設定値を読む口。Wails は公開メソッドをすべて bindings に出すので、
// Config 全体をフロントに見せないよう非公開にする (add-settings-screen design.md D3)。
type configSource interface {
	current() config.Config
}

// settingKey は互いに条件を持つ値を 1 つに束ね、key 単位で検証できるようにする (design.md D2)。
type settingKey struct {
	name     string
	encode   func(v settings.Values) any
	decode   func(raw json.RawMessage, v *settings.Values) error
	validate func(v settings.Values) error
}

var errMissingField = errors.New("settings: missing field")

type expiryAlertJSON struct {
	RedDays    *int `json:"redDays"`
	YellowDays *int `json:"yellowDays"`
}

type weeklyFillJSON struct {
	Days []int  `json:"days"`
	Meal string `json:"meal"`
}

type shoppingPeriodJSON struct {
	DayPresets  []int `json:"dayPresets"`
	DefaultDays *int  `json:"defaultDays"`
}

var settingKeys = []settingKey{
	{
		name:   "language",
		encode: func(v settings.Values) any { return v.Language },
		decode: func(raw json.RawMessage, v *settings.Values) error { return json.Unmarshal(raw, &v.Language) },
		validate: func(v settings.Values) error {
			return settings.ValidateLanguage(v.Language)
		},
	},
	{
		name: "expiryAlert",
		encode: func(v settings.Values) any {
			return expiryAlertJSON{RedDays: &v.ExpiryAlertRedDays, YellowDays: &v.ExpiryAlertYellowDays}
		},
		decode: func(raw json.RawMessage, v *settings.Values) error {
			var j expiryAlertJSON
			if err := json.Unmarshal(raw, &j); err != nil {
				return err
			}
			if j.RedDays == nil || j.YellowDays == nil {
				return errMissingField
			}
			v.ExpiryAlertRedDays, v.ExpiryAlertYellowDays = *j.RedDays, *j.YellowDays
			return nil
		},
		validate: func(v settings.Values) error {
			return settings.ValidateExpiryAlert(v.ExpiryAlertRedDays, v.ExpiryAlertYellowDays)
		},
	},
	{
		name:   "leftoverCopyPresets",
		encode: func(v settings.Values) any { return v.LeftoverCopyPresets },
		decode: func(raw json.RawMessage, v *settings.Values) error {
			return json.Unmarshal(raw, &v.LeftoverCopyPresets)
		},
		validate: func(v settings.Values) error { return settings.ValidateLeftoverCopyPresets(v.LeftoverCopyPresets) },
	},
	{
		name: "shoppingPeriod",
		encode: func(v settings.Values) any {
			return shoppingPeriodJSON{DayPresets: v.ShoppingPeriodDayPresets, DefaultDays: &v.ShoppingDefaultPeriodDays}
		},
		decode: func(raw json.RawMessage, v *settings.Values) error {
			var j shoppingPeriodJSON
			if err := json.Unmarshal(raw, &j); err != nil {
				return err
			}
			if j.DayPresets == nil || j.DefaultDays == nil {
				return errMissingField
			}
			v.ShoppingPeriodDayPresets, v.ShoppingDefaultPeriodDays = j.DayPresets, *j.DefaultDays
			return nil
		},
		validate: func(v settings.Values) error {
			return settings.ValidateShoppingPeriod(v.ShoppingPeriodDayPresets, v.ShoppingDefaultPeriodDays)
		},
	},
	{
		name:   "measuringUnits",
		encode: func(v settings.Values) any { return v.MeasuringUnits },
		decode: func(raw json.RawMessage, v *settings.Values) error { return json.Unmarshal(raw, &v.MeasuringUnits) },
		validate: func(v settings.Values) error {
			return settings.ValidateMeasuringUnits(v.MeasuringUnits)
		},
	},
	{
		name:   "unquantifiedWords",
		encode: func(v settings.Values) any { return v.UnquantifiedWords },
		decode: func(raw json.RawMessage, v *settings.Values) error {
			return json.Unmarshal(raw, &v.UnquantifiedWords)
		},
		validate: func(v settings.Values) error { return settings.ValidateUnquantifiedWords(v.UnquantifiedWords) },
	},
	{
		name:   "recipeImportDefaultServings",
		encode: func(v settings.Values) any { return v.RecipeImportDefaultServings },
		decode: func(raw json.RawMessage, v *settings.Values) error {
			return json.Unmarshal(raw, &v.RecipeImportDefaultServings)
		},
		validate: func(v settings.Values) error {
			return settings.ValidateImportServings(v.RecipeImportDefaultServings)
		},
	},
	{
		name:   "recencyThresholdDays",
		encode: func(v settings.Values) any { return v.RecencyThresholdDays },
		decode: func(raw json.RawMessage, v *settings.Values) error {
			return json.Unmarshal(raw, &v.RecencyThresholdDays)
		},
		validate: func(v settings.Values) error {
			return settings.ValidateRecencyThreshold(v.RecencyThresholdDays)
		},
	},
	{
		name:   "weeklyFill",
		encode: func(v settings.Values) any { return weeklyFillJSON{Days: v.WeeklyFillDays, Meal: v.WeeklyFillMeal} },
		decode: func(raw json.RawMessage, v *settings.Values) error {
			var j weeklyFillJSON
			if err := json.Unmarshal(raw, &j); err != nil {
				return err
			}
			if j.Days == nil || j.Meal == "" {
				return errMissingField
			}
			v.WeeklyFillDays, v.WeeklyFillMeal = j.Days, j.Meal
			return nil
		},
		validate: func(v settings.Values) error { return settings.ValidateWeeklyFill(v.WeeklyFillDays, v.WeeklyFillMeal) },
	},
	{
		name:   "weeklyPresetMaxRunDays",
		encode: func(v settings.Values) any { return v.WeeklyPresetMaxRunDays },
		decode: func(raw json.RawMessage, v *settings.Values) error {
			return json.Unmarshal(raw, &v.WeeklyPresetMaxRunDays)
		},
		validate: func(v settings.Values) error {
			return settings.ValidateWeeklyPresetMaxRunDays(v.WeeklyPresetMaxRunDays)
		},
	},
}

// loadConfig は読めない key と条件を満たさない key を捨て、その項目だけ既定値で動かす (design.md D6)。
// 未知の key は将来の版が足した key なので無視する。
func loadConfig(rows map[string]json.RawMessage) config.Config {
	defaults := config.Defaults()
	values := settings.FromConfig(defaults)
	for _, k := range settingKeys {
		raw, ok := rows[k.name]
		if !ok {
			continue
		}
		candidate := values
		if err := k.decode(raw, &candidate); err != nil {
			log.Printf("meal-cycle: ignore setting %s: %v", k.name, err)
			continue
		}
		candidate = candidate.Normalize()
		if err := k.validate(candidate); err != nil {
			log.Printf("meal-cycle: ignore setting %s: %v", k.name, err)
			continue
		}
		values = candidate
	}
	return values.ApplyTo(defaults)
}

// diffRows は既定値と違う key だけを返す。全項目を保存すると、既定値を見直した版に上げても古い既定値が残り続けるため。
func diffRows(values settings.Values) (map[string]json.RawMessage, error) {
	defaults := settings.FromConfig(config.Defaults())
	rows := make(map[string]json.RawMessage)
	for _, k := range settingKeys {
		got, err := json.Marshal(k.encode(values))
		if err != nil {
			return nil, err
		}
		want, err := json.Marshal(k.encode(defaults))
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(got, want) {
			rows[k.name] = got
		}
	}
	return rows, nil
}

// SettingsService は S09 設定画面をフロントに公開し、他サービスに現在の設定値を渡す。
type SettingsService struct {
	repo *sqlite.SettingsRepo
	mu   sync.RWMutex
	cfg  config.Config
	now  func() time.Time
}

// NewSettingsService は保存した値を読めなくても既定値で動かす。設定が読めないだけで献立や在庫まで使えなくしないため
// (design.md D6)。
func NewSettingsService(db *sql.DB) *SettingsService {
	s := &SettingsService{repo: sqlite.NewSettingsRepo(db), cfg: config.Defaults(), now: time.Now}
	rows, err := s.repo.Load(context.Background())
	if err != nil {
		log.Printf("meal-cycle: load settings: %v (using defaults)", err)
		return s
	}
	s.cfg = loadConfig(rows)
	return s
}

func (s *SettingsService) current() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *SettingsService) Get(_ context.Context) (settings.Values, error) {
	return settings.FromConfig(s.current()), nil
}

func (s *SettingsService) Save(ctx context.Context, values settings.Values) (settings.Values, error) {
	normalized := values.Normalize()
	if err := normalized.Validate(); err != nil {
		return settings.Values{}, err
	}
	rows, err := diffRows(normalized)
	if err != nil {
		return settings.Values{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.repo.ReplaceAll(ctx, rows, s.now()); err != nil {
		return settings.Values{}, err
	}
	s.cfg = normalized.ApplyTo(config.Defaults())
	return settings.FromConfig(s.cfg), nil
}

func (s *SettingsService) Reset(ctx context.Context) (settings.Values, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.repo.DeleteAll(ctx); err != nil {
		return settings.Values{}, err
	}
	s.cfg = config.Defaults()
	return settings.FromConfig(s.cfg), nil
}

// Constraints は S09 の画面が表示に使う規則を返す。版の中で変わらないので、保存した値には左右されない。
func (s *SettingsService) Constraints(_ context.Context) (settings.Constraints, error) {
	return settings.CurrentConstraints(), nil
}
