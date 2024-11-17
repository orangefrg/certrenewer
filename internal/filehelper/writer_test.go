package filehelper

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupFile(t *testing.T) {
	// Create a temporary file to act as the original file
	originalFile, err := os.CreateTemp("", "original-*.txt")
	require.NoError(t, err)
	defer os.Remove(originalFile.Name())

	// Write some content to the original file
	_, err = originalFile.WriteString("This is a test content")
	require.NoError(t, err)
	require.NoError(t, originalFile.Close())

	// Call BackupFile to create a backup
	err = BackupFile(originalFile.Name())
	require.NoError(t, err)

	// Check if the backup file exists
	backupFileName := originalFile.Name() + ".bak"
	backupFile, err := os.Open(backupFileName)
	require.NoError(t, err)
	defer os.Remove(backupFileName)
	defer backupFile.Close()

	// Verify the content of the backup file
	backupContent, err := io.ReadAll(backupFile)
	require.NoError(t, err)
	require.Equal(t, "This is a test content", string(backupContent))
}

func TestBackupFile_FileNotFound(t *testing.T) {
	// Call BackupFile with a non-existent file
	err := BackupFile("non-existent-file.txt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "error opening original file")
}

func TestBackupFile_CreateBackupError(t *testing.T) {
	// Create a temporary file to act as the original file
	originalFile, err := os.CreateTemp("", "original-*.txt")
	require.NoError(t, err)
	defer os.Remove(originalFile.Name())

	// Write some content to the original file
	_, err = originalFile.WriteString("This is a test content")
	require.NoError(t, err)
	require.NoError(t, originalFile.Close())

	// Simulate an error by creating a directory with the backup file name
	backupFileName := originalFile.Name() + ".bak"
	err = os.Mkdir(backupFileName, 0755)
	require.NoError(t, err)
	defer os.Remove(backupFileName)

	// Call BackupFile to create a backup
	err = BackupFile(originalFile.Name())
	require.Error(t, err)
	require.Contains(t, err.Error(), "error creating backup file")
}

func TestWriteWithBackup(t *testing.T) {
	// Create a temporary file to act as the original file
	originalFile, err := os.CreateTemp("", "original-*.txt")
	require.NoError(t, err)
	defer os.Remove(originalFile.Name())

	// Write some content to the original file
	_, err = originalFile.WriteString("This is a test content")
	require.NoError(t, err)
	require.NoError(t, originalFile.Close())

	// Call WriteWithBackup to write new content and create a backup
	newContent := []byte("This is new content")
	err = WriteWithBackup(originalFile.Name(), newContent, 0644)
	require.NoError(t, err)

	// Check if the backup file exists
	backupFileName := originalFile.Name() + ".bak"
	backupFile, err := os.Open(backupFileName)
	require.NoError(t, err)
	defer os.Remove(backupFileName)
	defer backupFile.Close()

	// Verify the content of the backup file
	backupContent, err := io.ReadAll(backupFile)
	require.NoError(t, err)
	require.Equal(t, "This is a test content", string(backupContent))

	// Verify the content of the original file
	originalFile, err = os.Open(originalFile.Name())
	require.NoError(t, err)
	defer originalFile.Close()

	originalContent, err := io.ReadAll(originalFile)
	require.NoError(t, err)
	require.Equal(t, "This is new content", string(originalContent))
}
