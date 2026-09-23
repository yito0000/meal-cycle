## Why

現行の作り置きコピー UX は 2 つの入り口に分かれている:

- **横コピー (同時間帯 × N 日)**: `⋯ → この品を連続 N 日 → 3 日` → コピー方法ダイアログ (作り置き / 別調理)
- **縦コピー (別時間帯 × 起点日 ±1 日)**: `⋯ → 別の時間帯にも作り置きを追加` → 対象日 + 対象時間帯
  ダイアログ (add-mealplan-cross-meal-leftover で追加)

同じ「作り置きを他のセルにも広げる」操作なのに、方向が違うだけで別メニュー / 別ダイアログを開かされる。
ペルソナ (作り置き中心) は「月・火は夕食、水は昼食」のような柔軟な組み合わせを表現したいが、現状は
横コピー → 縦コピーの 2 段階操作になる。「1 度の操作で複数の (対象日, 対象時間帯) にまとめて広げる」
形に統一したい。

具体例:

- 月曜・夕食の肉じゃがを「火曜・夕食」「水曜・昼食」「木曜・夕食」に広げたい
- 現状: 「連続 2 日」で月火 → 「別の時間帯にも追加」で水昼 → 「別の時間帯にも追加」で木夕 (3 段)
- 目標: 1 つのダイアログで (対象日 × 対象時間帯) をチェックボックス格子から複数選択 → 1 回で反映

## What Changes

- **`⋯ → 作り置きを広げる` に一本化**: 「この品を連続 N 日」と「別の時間帯にも作り置きを追加」の
  メニュー項目を 1 つに統合する。開くダイアログも 1 つ
- **統合ダイアログ (`spread-batch-dialog.tsx`)**: (対象日 × 対象時間帯) の格子 UI。行 = 対象日
  (起点日 ±4 日、7〜9 行)、列 = 対象時間帯 (朝 / 昼 / 夕)。各セルはチェックボックス。既存割当があるセルは
  背景色で示唆
- **既定の初期選択**: 起点日以降の同 meal で連続 `min(LeftoverCopyPresets)` 日 (= 2 日) をチェック済み
  で開く (「連続 2 日」の代替)。ユーザーは追加チェック / チェック解除で自由に組み合わせられる
- **作り置き / 別調理の選択**: ダイアログ下部の `作り置き / 別調理` セグメント (既存 `CopyModeDialog`
  の内容を統合)。既定は「作り置き」
- **衝突解決**: チェック済みセルのうち既存割当があるものは、下部の「衝突の扱い」セグメント
  (`追加 / 上書き`) で決める。既存の `CopyConflictDialog` の内容を統合
- **既存 API は据え置き**: サーバは `CopyLeftover` + `CopyToOtherMeal` を組み合わせて反映する
  (Repo/Service 変更なし)。UI 側は「同 meal 対象日群」と「別 meal 対象セル群」を選択から分けて
  `CopyLeftover` と `CopyToOtherMeal` に振り分ける
- **旧メニュー / ダイアログの削除**: `⋯ → この品を連続 N 日` サブメニュー、`⋯ → 別の時間帯にも作り置きを追加`、
  `CopyModeDialog`、`copy-to-meal-dialog.tsx` を削除する

## Non-goals

- **時間帯全体のコピー (`セル右下 ⋯ → 時間帯全体を連続 N 日コピー`)** の UX 統合: 別 proposal で扱う。
  「1 品を広げる」と「セル全体を広げる」は概念が違うので、まずは 1 品側の統合に絞る
- **サーバ API の統合 (CopyLeftover と CopyToOtherMeal を 1 つにする)**: Repo / Service は今のまま
  維持。UI 側の変更に閉じる
- **ドラッグ&ドロップ**: 別 proposal (`add-mealplan-drag-move`)
- **カレンダー UI 直接操作 (対象セルをクリックで選択)**: MVP 対象外。まずはダイアログ内格子から始める

## Capabilities

### Modified Capabilities

- `mealplan`:
  - **MODIFIED**: 「事後編集による作り置きの外し / まとめ」Requirement — 「この品を連続 N 日」
    「別の時間帯にも作り置きを追加」の項目を「作り置きを広げる」に統合する。表示条件も更新
  - **ADDED**: 「作り置きを 1 つのダイアログで広げる」Requirement — 統合ダイアログの挙動
    (格子 UI / 既定選択 / 作り置き別調理選択 / 衝突解決) を規定
  - **REMOVED**: 旧「作り置きコピー (連続 N 日)」および「作り置きを別時間帯にも追加」の Requirement を
    廃止 (統合 Requirement に置き換え)

## Impact

- **新設ファイル**:
  - `frontend/src/components/mealplan/spread-batch-dialog.tsx` — 統合ダイアログ本体
  - `frontend/src/lib/spread-batch-selection.ts` — 選択状態 → CopyLeftover / CopyToOtherMeal
    呼び出しへの分解ロジック (pure function、テストしやすい形)
- **削除ファイル**:
  - `frontend/src/components/mealplan/copy-to-meal-dialog.tsx`
  - `frontend/src/components/mealplan/copy-mode-dialog.tsx`
  - `frontend/src/components/mealplan/copy-conflict-dialog.tsx` は「時間帯全体コピー」でも使うので残す
    (削除は Non-goal で扱った別 proposal)
- **更新ファイル**:
  - `frontend/src/components/mealplan/cell-item-menu.tsx` — 「この品を連続 N 日」サブメニュー削除、
    「別の時間帯にも作り置きを追加」削除、「作り置きを広げる」を追加 (props も整理)
  - `frontend/src/components/mealplan/week-calendar.tsx` — `pendingCopy` / `copyToMeal` 状態を
    `spreadBatch` に統合、mutation 呼び出しを spread-batch-selection ヘルパ経由に
  - `frontend/src/locales/{ja,en}.json` — 新 key 追加、旧 key 削除
  - `docs/product.md` §S01 — 「作り置きを広げる」統合 UX を反映
  - `docs/scenarios/mealplan.feature` — シナリオ再編 (連続 N 日 / 別時間帯を統合ダイアログの
    シナリオに書き換え)
  - `frontend/e2e/mealplan-batch.spec.ts` — テスト名を新シナリオに合わせる
- **サーバ変更**: なし
- **migration**: なし (UI のみ)

### 後続 proposal への引き継ぎ

- **時間帯全体コピー統合**: 「セル右下 `⋯` → 時間帯全体を連続 N 日コピー」も同じダイアログに
  流し込むか、別 UX にするかは別 proposal で判断する
- **サーバ API 統合**: `CopyLeftover` + `CopyToOtherMeal` を将来的に 1 つのメソッドに寄せるかは、
  実装後の使用感を見てから判断
