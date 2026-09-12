package mealplan

import "time"

// CopyMode は CopyLeftover / PlanCopy の合流方針。
type CopyMode string

const (
	CopyModeAppend    CopyMode = "append"
	CopyModeOverwrite CopyMode = "overwrite"
)

// CopyDates は startDate を含む days 日ぶんの "YYYY-MM-DD" 配列を返す。
//
// days=0 は空、days<0 は ErrInvalidCopyDays。startDate が "YYYY-MM-DD" 形式でなければ ErrInvalidDate。
// 「連続 N 日」の解釈は「起点日を含む N 日」(design.md D5)。
func CopyDates(startDate string, days int) ([]string, error) {
	if days < 0 {
		return nil, ErrInvalidCopyDays
	}
	if days == 0 {
		return []string{}, nil
	}
	t, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return nil, ErrInvalidDate
	}
	out := make([]string, days)
	for i := 0; i < days; i++ {
		out[i] = t.AddDate(0, 0, i).Format(dateLayout)
	}
	return out, nil
}

// PlanCopy はコピー元の品を target 日の meal に合流させた結果 (実際に挿入すべき Entry 列) を返す。
//
// mode=append は existing の末尾に source を並べる (positions は existing の max+1 から)。
// mode=overwrite は existing を捨てて source だけを 0 から採番する (existing の削除は
// repo 側で行う想定で、返り値には含めない)。
//
// 二重追加防止: append で、existing に含まれる RecipeID と重複する source は返り値に含めない。
// (起点日 = target のときに、コピー元品を「もう 1 回」入れるのを避けるため)。
//
// shareBatch=true (作り置き) は「同じレシピ ID なら全 target で同じ BatchID」を割り振る
// (呼び出し側でレシピ ID → BatchID の map を作って渡してもよいが、PlanCopy は 1 target ずつの
// 呼び出しなので、source の BatchID をそのまま伝搬する形にする)。shareBatch=false (別調理) は
// この target の各 Entry に新しい BatchID を発行する。
func PlanCopy(
	target string, sourceEntries []Entry, existing []Entry,
	mode CopyMode, shareBatch ShareBatch,
) ([]Entry, error) {
	if mode != CopyModeAppend && mode != CopyModeOverwrite {
		return nil, ErrInvalidCopyMode
	}
	var startPos int
	seen := map[string]bool{}
	if mode == CopyModeAppend {
		for _, e := range existing {
			if e.Position+1 > startPos {
				startPos = e.Position + 1
			}
			seen[e.RecipeID] = true
		}
	}
	out := make([]Entry, 0, len(sourceEntries))
	pos := startPos
	for _, s := range sourceEntries {
		if mode == CopyModeAppend && seen[s.RecipeID] {
			continue
		}
		batchID := s.BatchID
		if !shareBatch.IsShared() || batchID == "" {
			batchID = NewBatchID()
		}
		out = append(out, Entry{
			Date:       target,
			Meal:       s.Meal,
			Position:   pos,
			RecipeID:   s.RecipeID,
			RecipeName: s.RecipeName,
			BatchID:    batchID,
		})
		pos++
	}
	return out, nil
}

// IsShared は ShareBatch を bool として読み替えるためのアクセサ。
// alias 型のまま `if shareBatch { ... }` と書くと meal まわりで「なぜ true を渡すのか」が
// 呼び出し側から埋没するため、意図が読める語彙で分岐する。
func (s ShareBatch) IsShared() bool { return bool(s) }
