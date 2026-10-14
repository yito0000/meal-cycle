// 統合ダイアログ「作り置きを広げる」の選択状態の型。
// 選んだセルはそのまま MealPlanService.SpreadEntry に渡す (revise-mealplan-spread-api design.md D1)。

export type SpreadTarget = { date: string; meal: string };

export type SpreadConflictMode = "append" | "overwrite";

export type SpreadSelection = {
  targets: SpreadTarget[];
  shareBatch: boolean;
  conflictMode: SpreadConflictMode;
};

export type SpreadSource = { date: string; meal: string; position: number };
