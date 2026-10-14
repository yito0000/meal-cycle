package mealplan

import "github.com/google/uuid"

// ShareBatch は連続 N 日コピー時の「作り置き / 別調理」意図。true なら 1 バッチ共有、
// false なら 1 行ごとに独立バッチ。boolean を裸で渡すと呼び出し側の意味が曖昧になるため
// alias を切って、シグネチャ上で意図が読めるようにする。
type ShareBatch bool

const (
	ShareBatchLeftover ShareBatch = true  // 作り置き (材料 1 セット)
	ShareBatchDistinct ShareBatch = false // 別調理 (材料 N セット)
)

// NewBatchID は UUID v7 を lower-case hex (ハイフンあり) で返す。
// recipe.ID も uuid.NewV7 で発行しているため、両テーブルの ID 生成方針を揃える。
func NewBatchID() string {
	id, err := uuid.NewV7()
	if err != nil {
		// uuid.NewV7 は crypto/rand の失敗以外では err を返さない。desktop 環境で
		// crypto/rand が落ちる状況は想定外のため panic に落とす。
		panic("mealplan: uuid.NewV7 failed: " + err.Error())
	}
	return id.String()
}

// NewEntryID は献立の行の ID を、作り置きの ID と同じ UUID v7 で発行する (add-mealplan-entry-id design.md D1)。
func NewEntryID() string {
	return NewBatchID()
}
