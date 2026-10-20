package applog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	cases := []struct {
		name       string
		existing   string
		oldBackup  string
		limit      int64
		wantLog    string
		wantBackup string
	}{
		{name: "creates file when missing", limit: 10, wantLog: "new"},
		{name: "appends when under limit", existing: "old,", limit: 10, wantLog: "old,new"},
		{name: "appends when exactly at limit", existing: "0123456789", limit: 10, wantLog: "0123456789new"},
		{name: "rotates when over limit", existing: "0123456789X", limit: 10, wantLog: "new", wantBackup: "0123456789X"},
		{name: "overwrites old backup when rotating", existing: "0123456789X", oldBackup: "stale", limit: 10, wantLog: "new", wantBackup: "0123456789X"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "app.log")
			if tc.existing != "" {
				writeFile(t, path, tc.existing)
			}
			if tc.oldBackup != "" {
				writeFile(t, path+".1", tc.oldBackup)
			}

			f, err := open(path, tc.limit)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString("new"); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}

			if got := readFile(t, path); got != tc.wantLog {
				t.Errorf("log = %q, want %q", got, tc.wantLog)
			}
			wantBackup := tc.wantBackup
			if wantBackup == "" {
				wantBackup = tc.oldBackup
			}
			if got := readFile(t, path+".1"); got != wantBackup {
				t.Errorf("backup = %q, want %q", got, wantBackup)
			}
		})
	}
}

func writeFile(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
