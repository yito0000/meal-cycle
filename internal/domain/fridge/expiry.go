package fridge

import "time"

// DateLayout は ExpiresAt の書式。
const DateLayout = "2006-01-02"

// Evaluate は賞味期限 expiresAt と今日 today から期限状態と残り日数を返す。
// 期限なし (空文字) と解釈できない値は (ExpiryNone, nil)。
// 残り日数は today のロケーションでの日付同士の差で、時刻は見ない
// (UTC で差を取ると JST の 0〜9 時に 1 日ずれるため)。
func Evaluate(expiresAt string, today time.Time, red, yellow int) (ExpiryStatus, *int) {
	if expiresAt == "" {
		return ExpiryNone, nil
	}
	exp, err := time.Parse(DateLayout, expiresAt)
	if err != nil {
		return ExpiryNone, nil
	}
	y, m, d := today.Date()
	// 両方を UTC の暦日に載せ替えると、夏時間などで 1 日が 24h でない地域でも割り切れる。
	todayDate := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	days := int(exp.Sub(todayDate).Hours() / 24)

	switch {
	case days < 0:
		return ExpiryExpired, &days
	case days <= red:
		return ExpiryUrgent, &days
	case days <= yellow:
		return ExpirySoon, &days
	default:
		return ExpiryFresh, &days
	}
}
