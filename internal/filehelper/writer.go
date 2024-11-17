package filehelper

import (
	"fmt"
	"io"
	"io/fs"
	"os"
)

var WriteWithBackup = func(filename string, contents []byte, perm fs.FileMode) error {
	err := BackupFile(filename)
	if err != nil {
		return fmt.Errorf("error making backup: %w", err)
	}

	return os.WriteFile(filename, contents, perm)
}

func BackupFile(filename string) error {
	backupPath := filename + ".bak"
	originalFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening original file: %w", err)
	}
	defer originalFile.Close()

	backupFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("error creating backup file: %w", err)
	}
	defer backupFile.Close()

	_, err = io.Copy(backupFile, originalFile)
	if err != nil {
		return fmt.Errorf("error copying content to backup: %w", err)
	}

	return nil
}
