// Package applog は配布用ビルドで消えてしまう stderr の代わりに、ログをファイルへ残す。
package applog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const (
	appDirName  = "meal-cycle"
	fileName    = "meal-cycle.log"
	maxFileSize = 1 << 20
)

// DefaultPath は DB と同じディレクトリのログファイルパスを返す (例: %APPDATA%\meal-cycle\meal-cycle.log)。
func DefaultPath() (string, error) {
	p, err := xdg.DataFile(filepath.Join(appDirName, fileName))
	if err != nil {
		return "", fmt.Errorf("resolve log path: %w", err)
	}
	return p, nil
}

// Open は path を追記モードで開く。起動ごとに追記して際限なく育たないよう、
// 開く時点で maxFileSize を超えていれば path.1 へ退避し (古い path.1 は上書き)、新しいファイルから始める。
func Open(path string) (*os.File, error) {
	return open(path, maxFileSize)
}

func open(path string, limit int64) (*os.File, error) {
	if fi, err := os.Stat(path); err == nil && fi.Size() > limit {
		if err := os.Rename(path, path+".1"); err != nil {
			return nil, fmt.Errorf("rotate log: %w", err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log: %w", err)
	}
	return f, nil
}
