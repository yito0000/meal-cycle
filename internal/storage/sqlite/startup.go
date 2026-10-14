package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

type StartupStage string

const (
	StageOpen    StartupStage = "open"
	StageBackup  StartupStage = "backup"
	StageMigrate StartupStage = "migrate"
)

// フロントは Stage のコードから見出しを引き、Err の文字列は詳細としてそのまま出す (design.md D2)。
type StartupError struct {
	Stage StartupStage
	Err   error
}

func (e *StartupError) Error() string {
	return fmt.Sprintf("%s: %v", e.Stage, e.Err)
}

func (e *StartupError) Unwrap() error {
	return e.Err
}

// OpenAndMigrate は DB を開き、更新があれば控えを取ってからスキーマを最新にする (design.md D6)。
// 失敗は *StartupError で返す。
func OpenAndMigrate(path string) (*sql.DB, error) {
	// 初回かどうかは Open がファイルを作る前にしか判定できない (design.md D4)。
	_, statErr := os.Stat(path)
	existed := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return nil, &StartupError{Stage: StageOpen, Err: statErr}
	}

	db, err := Open(path)
	if err != nil {
		return nil, &StartupError{Stage: StageOpen, Err: err}
	}

	if existed {
		current, pending, err := pendingMigrations(db)
		if err != nil {
			_ = db.Close()
			return nil, &StartupError{Stage: StageMigrate, Err: err}
		}
		if pending {
			if _, err := backupBeforeMigrate(db, path, current, time.Now()); err != nil {
				_ = db.Close()
				return nil, &StartupError{Stage: StageBackup, Err: err}
			}
			// 古い控えが消せなくても更新前の控えは取れているので、起動は止めない。
			if err := pruneBackups(BackupDir(path), backupKeep); err != nil {
				log.Printf("meal-cycle: prune backups: %v", err)
			}
		}
	}

	if err := Migrate(db); err != nil {
		_ = db.Close()
		return nil, &StartupError{Stage: StageMigrate, Err: err}
	}
	return db, nil
}
