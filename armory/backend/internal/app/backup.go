package app

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const keepBackups = 14

func runBackups(ctx context.Context, db *sql.DB, dataDir string) {
	dirs := []string{filepath.Join(dataDir, "backups")}
	if extra := extraBackupDir(dataDir); extra != "" {
		dirs = append(dirs, extra)
	}
	for {
		for _, dir := range dirs {
			if err := backupTo(ctx, db, dir); err != nil {
				log.Printf("backup to %s failed: %v", dir, err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(24 * time.Hour):
		}
	}
}

func extraBackupDir(dataDir string) string {
	if v := os.Getenv("ARMORY_BACKUP_DIR"); v != "" {
		return v
	}
	b, err := os.ReadFile(filepath.Join(dataDir, "backup-folder.txt"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func backupTo(ctx context.Context, db *sql.DB, dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	target := filepath.Join(dir, "armory-"+time.Now().Format("2006-01-02")+".db")
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	tmp := target + ".tmp"
	os.Remove(tmp)
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		return err
	}
	log.Printf("backup saved to %s", target)
	prune(dir)
	return nil
}

func prune(dir string) {
	files, err := filepath.Glob(filepath.Join(dir, "armory-*.db"))
	if err != nil || len(files) <= keepBackups {
		return
	}
	sort.Strings(files)
	for _, f := range files[:len(files)-keepBackups] {
		os.Remove(f)
	}
}
