package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/yito0000/meal-cycle/internal/domain/settings"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// フロントは文言の部分一致でメッセージを引くので、変えるときは settings の database のエラーの表も直す。
var (
	ErrDatabaseExportFailed = errors.New("database: export failed")
	ErrBackupNotFound       = errors.New("database: backup not found")
)

const exportStampLayout = "20060102-150405"

// DatabaseService は S09 のデータの節 (書き出しと復元の予約) をフロントに公開する。
// 設定の値の保存とは関心が違い、DB のパスとファイルの picker が要るので SettingsService と分ける (add-settings-database-file design.md D4)。
// MCP サーバーには登録しない。
type DatabaseService struct {
	db     *sql.DB
	dbPath string
	picker FilePicker
	now    func() time.Time
	quit   func()
}

// quit が nil ならアプリを終了する。E2E サーバーはプロセスを止めないよう何もしない関数を渡す。
func NewDatabaseService(db *sql.DB, dbPath string, picker FilePicker, quit func()) *DatabaseService {
	if quit == nil {
		quit = func() { application.Get().Quit() }
	}
	return &DatabaseService{db: db, dbPath: dbPath, picker: picker, now: time.Now, quit: quit}
}

// Export は保存ダイアログで選んだ場所に DB 全体を書き出す。ダイアログを閉じたら何もしない。
func (s *DatabaseService) Export(ctx context.Context, dialogTitle string) (settings.DatabaseExportResult, error) {
	name := fmt.Sprintf("meal-cycle-%s.db", s.now().Format(exportStampLayout))
	dest, err := s.picker.SaveFile(dialogTitle, name)
	if err != nil {
		return settings.DatabaseExportResult{}, fmt.Errorf("%w: %v", ErrDatabaseExportFailed, err)
	}
	if dest == "" {
		return settings.DatabaseExportResult{Cancelled: true}, nil
	}
	if err := sqlite.ExportTo(s.db, dest); err != nil {
		return settings.DatabaseExportResult{}, fmt.Errorf("%w: %v", ErrDatabaseExportFailed, err)
	}
	return settings.DatabaseExportResult{Path: dest}, nil
}

func (s *DatabaseService) ListBackups(ctx context.Context) ([]settings.BackupFile, error) {
	return sqlite.ListBackups(s.dbPath)
}

// ReserveFromBackup は ListBackups が返した名前だけを受け付ける。任意のパスを渡されて backups の外を読まないようにする。
func (s *DatabaseService) ReserveFromBackup(ctx context.Context, name string) (settings.PendingRestore, error) {
	list, err := sqlite.ListBackups(s.dbPath)
	if err != nil {
		return settings.PendingRestore{}, err
	}
	for _, b := range list {
		if b.Name == name {
			if err := sqlite.ReserveRestore(s.dbPath, filepath.Join(sqlite.BackupDir(s.dbPath), name), name); err != nil {
				return settings.PendingRestore{}, err
			}
			return sqlite.PendingRestore(s.dbPath)
		}
	}
	return settings.PendingRestore{}, ErrBackupNotFound
}

// ReserveFromFile はファイルの選択で選んだファイルで予約する。選択を閉じたら何もしない。
func (s *DatabaseService) ReserveFromFile(ctx context.Context, dialogTitle string) (settings.ReserveResult, error) {
	src, err := s.picker.OpenFile(dialogTitle)
	if err != nil {
		return settings.ReserveResult{}, err
	}
	if src == "" {
		return settings.ReserveResult{Cancelled: true}, nil
	}
	if err := sqlite.ReserveRestore(s.dbPath, src, filepath.Base(src)); err != nil {
		return settings.ReserveResult{}, err
	}
	p, err := sqlite.PendingRestore(s.dbPath)
	if err != nil {
		return settings.ReserveResult{}, err
	}
	return settings.ReserveResult{Pending: p}, nil
}

func (s *DatabaseService) Pending(ctx context.Context) (settings.PendingRestore, error) {
	return sqlite.PendingRestore(s.dbPath)
}

func (s *DatabaseService) CancelPending(ctx context.Context) error {
	return sqlite.CancelRestore(s.dbPath)
}

// Quit はアプリを終了する。Wails v3 には自分を起動し直す API が無いので、起動は利用者に任せる (proposal.md Non-goals)。
func (s *DatabaseService) Quit(ctx context.Context) {
	s.quit()
}
