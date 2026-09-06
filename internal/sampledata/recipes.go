package sampledata

import "github.com/yito0000/meal-cycle/internal/domain/recipe"

// Recipes は S02/S04 の表示確認用の 6 品。カテゴリ・タグ数・材料数・
// 手順長さ・備考の有無をわざとバラつかせている (scratchpad/sample-recipes.md
// と同じ内容)。
func Recipes() []recipe.Recipe {
	return []recipe.Recipe{
		{
			Name:     "鶏の照り焼き",
			Servings: 2,
			Tags:     []string{"主菜", "和食", "鶏肉"},
			Ingredients: []recipe.RecipeIngredient{
				{Name: "鶏もも肉", Amount: 300, Unit: "g"},
				{Name: "醤油", Amount: 2, Unit: "大さじ"},
				{Name: "みりん", Amount: 2, Unit: "大さじ"},
				{Name: "酒", Amount: 1, Unit: "大さじ"},
				{Name: "砂糖", Amount: 1, Unit: "大さじ"},
				{Name: "サラダ油", Amount: 1, Unit: "小さじ"},
			},
			Steps: `## 下ごしらえ
- 鶏もも肉は厚みを均一にするため観音開きにする
- 醤油・みりん・酒・砂糖を合わせておく

## 焼く
1. フライパンに油をひき、皮目から中火で 5 分焼く
2. 裏返して 3 分焼き、余分な脂をキッチンペーパーで拭き取る
3. 合わせ調味料を加え、煮絡めながら照りを出す

## 仕上げ
- 食べやすい大きさに切って盛り付け`,
			Notes: `- タレは **煮詰めすぎると焦げる** ので中火で手早く
- 冷めても美味しいので弁当にも可`,
		},
		{
			Name:     "ほうれん草のごま和え",
			Servings: 2,
			Tags:     []string{"副菜", "和食", "作り置き"},
			Ingredients: []recipe.RecipeIngredient{
				{Name: "ほうれん草", Amount: 1, Unit: "束"},
				{Name: "すりごま", Amount: 2, Unit: "大さじ"},
				{Name: "醤油", Amount: 1, Unit: "大さじ"},
				{Name: "砂糖", Amount: 1, Unit: "小さじ"},
			},
			Steps: `1. ほうれん草を塩ゆでし、冷水にとって水気を絞る
2. 3 cm 幅に切る
3. すりごま・醤油・砂糖を混ぜ、ほうれん草と和える`,
			Notes: "",
		},
		{
			Name:     "豆腐とわかめの味噌汁",
			Servings: 2,
			Tags:     []string{"汁物", "和食"},
			Ingredients: []recipe.RecipeIngredient{
				{Name: "絹豆腐", Amount: 150, Unit: "g"},
				{Name: "乾燥わかめ", Amount: 2, Unit: "g"},
				{Name: "だし汁", Amount: 400, Unit: "ml"},
				{Name: "味噌", Amount: 1.5, Unit: "大さじ"},
			},
			Steps: `- だし汁を沸かし、さいの目に切った豆腐を入れる
- わかめを加えてひと煮立ち
- 火を止めてから味噌を溶き入れる`,
			Notes: "- 味噌は **煮立てない** (風味が飛ぶ)",
		},
		{
			Name:     "鮭のムニエル",
			Servings: 2,
			Tags:     []string{"主菜", "洋食", "魚"},
			Ingredients: []recipe.RecipeIngredient{
				{Name: "生鮭", Amount: 2, Unit: "切れ"},
				{Name: "塩", Amount: 0, Unit: "少々"},
				{Name: "こしょう", Amount: 0, Unit: "少々"},
				{Name: "薄力粉", Amount: 0, Unit: "適量"},
				{Name: "バター", Amount: 20, Unit: "g"},
				{Name: "レモン", Amount: 0.25, Unit: "個"},
			},
			Steps: `1. 鮭に塩こしょうを振り、10 分置いて出てきた水気を拭く
2. 薄力粉を薄くまぶす
3. 中火で温めたフライパンにバターを溶かし、皮目から焼く
4. 両面がきつね色になったら皿に盛り、レモンを添える`,
			Notes: "- 粉は **焼く直前** にまぶす。早すぎるとベタつく",
		},
		{
			Name:     "トマトとモッツァレラのカプレーゼ",
			Servings: 2,
			Tags:     []string{"副菜", "イタリアン"},
			Ingredients: []recipe.RecipeIngredient{
				{Name: "トマト", Amount: 1, Unit: "個"},
				{Name: "モッツァレラチーズ", Amount: 100, Unit: "g"},
				{Name: "バジル", Amount: 0, Unit: "適量"},
				{Name: "オリーブオイル", Amount: 1, Unit: "大さじ"},
				{Name: "塩", Amount: 0, Unit: "少々"},
				{Name: "黒こしょう", Amount: 0, Unit: "少々"},
			},
			Steps: `- トマトとモッツァレラを 7 mm 厚さの輪切りにする
- 交互に並べ、バジルを散らす
- オリーブオイルを回しかけ、塩・黒こしょうで味を整える`,
			Notes: "",
		},
		{
			// タグ無しでの一覧表示確認用
			Name:     "豚汁",
			Servings: 4,
			Tags:     nil,
			Ingredients: []recipe.RecipeIngredient{
				{Name: "豚こま切れ肉", Amount: 150, Unit: "g"},
				{Name: "大根", Amount: 100, Unit: "g"},
				{Name: "人参", Amount: 50, Unit: "g"},
				{Name: "ごぼう", Amount: 50, Unit: "g"},
				{Name: "こんにゃく", Amount: 100, Unit: "g"},
				{Name: "長ねぎ", Amount: 0.5, Unit: "本"},
				{Name: "だし汁", Amount: 800, Unit: "ml"},
				{Name: "味噌", Amount: 3, Unit: "大さじ"},
				{Name: "ごま油", Amount: 1, Unit: "小さじ"},
			},
			Steps: `## 下ごしらえ
- 大根・人参はいちょう切り
- ごぼうはささがきにして水にさらす
- こんにゃくは手でちぎって下ゆで

## 煮る
1. 鍋にごま油を熱し、豚肉を炒める
2. 野菜とこんにゃくを加えてさらに炒める
3. だし汁を加え、アクを取りながら中火で 10 分煮る
4. 火を止め、味噌を溶き入れる
5. 器に盛り、小口切りの長ねぎを散らす`,
			Notes: `- 具材が多いので **4 人分** で登録している
- 翌日温め直すと味が染みてさらに美味`,
		},
	}
}
