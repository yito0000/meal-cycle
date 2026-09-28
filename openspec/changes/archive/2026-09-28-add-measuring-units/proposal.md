## Why

レシピの材料には「醤油 大さじ 2」「ごま油 小さじ 1」のように、調理で量るための単位が多く入る。
ところが S07 は、これを「玉ねぎ 2 個」と同じ買う量として扱い、必要量の列に「3 大さじ」と並べている。
醤油は瓶で買って瓶で冷蔵庫に登録するので、在庫は「醤油 1 本」になる。
単位が違う在庫は数えない規則のため、醤油の瓶があっても行はずっと「在庫なし」のまま残る。
一括登録のダイアログにも「醤油 / 3 / 大さじ」が初期値で入り、そのまま登録すると在庫が実態と合わなくなる。

大さじで量る材料は、「塩 少々」と同じく買い物の場面では量に意味が無い。
知りたいのは量ではなく、家にあるかどうかである。
S07 にはすでに量 0 の材料 (塩少々) を「—」で表示する規則があり、大さじや小さじの材料もこの規則で扱えば足りる。

もう一つ、量と単位の並べ方も合っていない。
S03 のレシピ詳細は「1 大さじ」と表示するが、レシピの表記では「大さじ 1」と単位を前に置くのが普通である。

## What Changes

- **量る単位** を定義する。既定は「大さじ」「小さじ」「カップ」の 3 つとし、`internal/config/defaults.go` に置く
- S07 で、量る単位の材料と量 0 の材料をまとめて **量を持たない材料** として扱う
  - 量を持たない材料は、単位を問わず食材名だけで 1 行に合算する (「醤油 大さじ 2」と「醤油 小さじ 1」は「醤油」の 1 行になる)
  - 必要量の列は「—」とする
  - 在庫の状態は、同じ食材名の在庫が単位を問わず 0 より多ければ「在庫あり」、無ければ「在庫なし」とする。「一部在庫あり」にはならない
  - 同じ食材名でも、量る単位以外の単位で量のある材料 (「醤油 1 本」) は、これまでどおり別の行になる
  - 一括登録のダイアログでは、量を持たない材料の量と単位を空にする
- 量と単位を表示するとき、量る単位は量の前に置く (「大さじ 1」「小さじ 0.5」)。それ以外の単位はこれまでどおり量の後に置く
  - 対象は S03 の材料欄と S05 の量の列。S07 では量る単位の材料は必要量が「—」になるので、量と単位を並べる場面が無い
  - 量 0 の材料は、これまでどおり量を出さずに単位だけを出す

### Non-goals

- **単位の換算**：大さじ 3 を小さじ 9 に直すような換算はしない。量を持たない材料は量を足し合わせないので、換算は要らない。
- **表記ゆれの吸収**：「大匙」「tbsp」「大さじ」を同じ単位とみなす処理はしない。量る単位の一覧に足せば同じ扱いになる。
- **常備品の区別**：調味料を別の欄に分けたり、既定で隠したりはしない。
- **S04 の入力欄の変更**：量と単位は、これまでどおり別々の欄に入れる。
- **量る単位の一覧を画面で変えること**：S09 (設定画面) の proposal で扱う。

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `shopping`：量を持たない材料の合算、在庫の状態、一括登録の初期値を変える。
- `recipe`：S03 の材料欄で、量る単位を量の前に置く。
- `fridge`：S05 の量の列で、量る単位を量の前に置く。

## Impact

### 新設ファイル

- `frontend/src/lib/format-quantity.ts`：量と単位を 1 つの文字列にする関数と、量る単位の一覧 (Go 側の mirror)

### 更新ファイル

- `internal/config/defaults.go`：`MeasuringUnits` を追加
- `internal/config/defaults_test.go`：`MeasuringUnits` の既定値を検証
- `internal/domain/shopping/aggregate.go`：量を持たない材料の合算キーと在庫の状態
- `internal/domain/shopping/aggregate_test.go`：量を持たない材料のケースを追加
- `internal/service/shopping.go`：`Aggregate` に `cfg.MeasuringUnits` を渡す
- `internal/service/shopping_test.go`：量る単位の材料のケースを追加
- `frontend/src/components/shopping/shopping-table.tsx`：必要量の列を `formatQuantity` で表示 (表示は変わらない。規則を 1 か所に揃えるため)
- `frontend/src/components/recipe/recipe-detail-drawer.tsx`：材料欄を `formatQuantity` で表示
- `frontend/src/components/fridge/fridge-item-row.tsx`：量の列を `formatQuantity` で表示
- `frontend/e2e/shopping.spec.ts`：量を持たない材料のシナリオを追加
- `frontend/e2e/shopping-purchase.spec.ts`：量る単位の材料の一括登録のシナリオを追加
- `frontend/e2e/recipe-detail.spec.ts`：量る単位の表示順のシナリオを追加
- `frontend/e2e/fridge.spec.ts`：量る単位の表示順のシナリオを追加
- `docs/product.md`：§1 の可変値の表に量る単位を追加、§4 S07 の材料テーブルの項を更新
- `openspec/specs/{shopping,recipe,fridge}/spec.md`：archive 時に delta を合成 (手で直接は触らない)

### 生成物 (再生成のみ)

- なし。`shopping.Item` の形は変えないので、Wails のバインディングは変わらない

### migration

- なし

### 前提と順序

- `add-shopping-stock-filter` とは独立に apply できる。どちらも `docs/product.md` の §4 S07 を触るので、後から入れる側で差分を合わせる

### 後続 proposal への引き継ぎ

- S09 (設定画面)：`MeasuringUnits` を設定項目にし、フロントの mirror を `ConfigService` からの取得に差し替える
- 量を持たない材料に、レシピの量を参考として添える案 (「—」の代わりに「大さじ 2 + 小さじ 1」) は、不満が出たら検討する
