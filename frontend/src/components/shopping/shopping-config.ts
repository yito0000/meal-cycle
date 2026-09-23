// internal/config/defaults.go (ShoppingPeriodDayPresets / ShoppingDefaultPeriodDays) のフロント側 mirror。
// S09 で ConfigService が Defaults() を返すまで手動で同期する。source of truth は defaults_test.go。
export const SHOPPING_DEFAULTS = {
  periodDayPresets: [3, 4, 7, 14, 28] as const,
  defaultPeriodDays: 7,
};
