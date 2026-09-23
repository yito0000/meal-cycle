## Context

- 動機は [proposal.md](proposal.md) 参照
- 完全ローカルの Wails アプリで、WebView は Chromium 系。localStorage は各 OS で永続化される
  (Windows: `%APPDATA%\...\WebView2`, macOS: `~/Library/Application Support/...`, Linux:
  `~/.local/share/...`)
- 既存の永続化対象は SQLite DB のみ。UI 状態は永続化していない

## Goals / Non-Goals

**Goals:**

- 時間帯表示 (`visibleMeals`) と期間長 (`periodWeeks`) の初期値を「前回終了時の状態」に揃える
- 値が壊れていても既定値にフォールバックしてアプリを落とさない
- S09 設定画面が入るときに ConfigService へ移管しやすい形にする (呼び出し側は 1 箇所のフック)

**Non-Goals:**

- SQLite への永続化 (S09 で行う。今回は localStorage で最小コスト)
- `periodStart` の永続化 (「今週」基準リセットが仕様)
- 設定画面 UI の追加 (既存の切替 UI をそのまま使う)

## Decisions

### D1. 永続化バックエンドの選定

**選択肢:**

- (a) **localStorage** ← 採用 — Pros: 実装 30 分、DB 変更なし、パース失敗時のフォールバックだけで
  動作。Cons: WebView の origin ごとに独立。将来 SQLite に移すときは移行スクリプトが要る
- (b) SQLite (`ui_preferences` テーブル新設) — Pros: source of truth が 1 系統。Cons: MVP に対して
  重すぎる。migration / repo / service / bindings まで一式必要
- (c) `internal/config/defaults.go` に置いて Wails からファイル永続化 — Pros: 完全ローカルらしい統一
  Cons: config は「不変な既定値」レイヤであり、可変な UI 状態を置くと責務が混ざる

**採用: (a)**。S09 設定画面が入るタイミングで (b) に移す前提。呼び出し側は 1 つのフック
(`useMealPlanPreference`) にまとめておき、後で内部実装だけ差し替えられるようにする。

### D2. フックの API 形状

**選択肢:**

- (a) 汎用 `useLocalStoragePreference<T>(key, default, validate)` — Pros: 他ページでも使い回せる。
  Cons: 汎用化により初回の read/write 判定がジェネリックで少し複雑
- (b) 対象別の `useVisibleMealsPreference()` / `usePeriodWeeksPreference()` — Pros: シグネチャが
  シンプル、型付けが自然。Cons: 増えるたびに hook を足す
- (c) mealplan 専用ヘルパ `useMealPlanPreference<T>(key, default, validate)` ← 採用 —
  Pros: (a) の汎用性と (b) の限定性の中間。mealplan 配下の複数 key を扱いつつ、他 capability に漏れない

**採用: (c)**。今回追加する key は 2 つ (`visibleMeals`, `periodWeeks`)。他 capability で必要になった
ときに一般化する。key prefix は `mealcycle.mealplan.` で衝突を避ける (WebView2 の origin は 1 つで
複数アプリを共存させる想定は無いが、prefix はコスト無しの防御)。

### D3. 値の validate 戦略

**選択肢:**

- (a) `validate(raw): T` を各 key ごとに書く ← 採用 — Pros: 型情報が呼び出し側にある、値異常
  (`periodWeeks: 3` 等) を弾ける
- (b) スキーマライブラリ (zod 等) を導入 — Cons: 依存追加のコストが大きい

**採用: (a)**。visibleMeals は `Meal` enum のみ許容、periodWeeks は AllowedPeriodWeeks (1/2/4) のみ
許容する軽量な検証。

### D4. localStorage エラー時の挙動

**選択肢:**

- (a) throw を上に投げる — Cons: アプリが落ちる
- (b) try/catch で握り潰して既定値を返す ← 採用 — Pros: private mode / storage 無効環境でも動作
- (c) 握り潰し + console.warn — Cons: 完全ローカルアプリでコンソール警告に意味がない

**採用: (b)**。read/write 両方で try/catch。ログも残さず、あくまで「設定が保存できない」だけの
軽微な劣化として扱う。

## Risks / Trade-offs

- [S09 で SQLite に移すとき既存 localStorage のユーザーは初期値に戻る] → S09 実装時に
  「初回起動で localStorage を読んで DB に write」する move 処理を入れる。今の proposal で
  この責務は取らない
- [Wails WebView が古い版に切り替わると localStorage が消える可能性] → 発生頻度は低い。消えた
  場合は既定値に戻るだけで害はない
- [複数 Wails アプリが同一マシンにインストールされた場合の origin 混在] → 今後 meal-cycle 以外の
  Wails アプリが同一 WebView2 profile を共有すると理論上ぶつかる。`mealcycle.mealplan.` の
  prefix で回避

## Migration Plan

- localStorage への保存だけなので既存データに触れず、既存ユーザーには「初回起動は既定値、以降
  切替後は保存される」挙動になる
- ロールバック: `frontend/src/lib/mealplan-preferences.ts` を消し、`meal-plan.tsx` を `useState`
  ベースに戻せば元に戻る
- S09 に移すときは `useMealPlanPreference` の中身だけ ConfigService 呼び出しに差し替える
