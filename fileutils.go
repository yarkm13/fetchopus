package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func saveRemoteFile(remotePath, localBasePath string, basePath string, reader io.Reader) error {
	relativePath := strings.TrimPrefix(remotePath, basePath)
	relativePath = strings.TrimPrefix(relativePath, "/")

	localPath := filepath.Join(localBasePath, relativePath)

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	destFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, reader)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}
