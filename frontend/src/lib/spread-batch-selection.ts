// 統合ダイアログ「作り置きを広げる」の選択状態の型。
// 選んだセルはそのまま MealPlanService.SpreadEntry に渡す (revise-mealplan-spread-api design.md D1)。

export type SpreadTarget = { date: string; meal: string };

export type SpreadConflictMode = "append" | "overwrite";

export type SpreadSelection = {
  targets: SpreadTarget[];
  shareBatch: boolean;
  conflictMode: SpreadConflictMode;
};

// id で起点の品を指し、date / meal は格子の起点セルの表示に使う (add-mealplan-entry-id design.md D7)。
export type SpreadSource = { id: string; date: string; meal: string };
