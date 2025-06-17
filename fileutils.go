package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func resolveRelativePath(remotePath, basePath, localBasePath string) (string, error) {
	relativePath := strings.TrimPrefix(remotePath, basePath)
	relativePath = strings.TrimPrefix(relativePath, "/")

	localPath := filepath.Join(localBasePath, relativePath)

	absolutePath, err := filepath.Abs(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return absolutePath, nil
}

func saveRemoteFile(remotePath, localBasePath string, basePath string, reader io.Reader) error {
	localPath, err := resolveRelativePath(remotePath, basePath, localBasePath)
	if err != nil {
		log.Fatalf("Error resolving path: %v", err)
	}

	// @todo receive mode from caller
	// @todo implement param to respect mode from remote or override
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	destFile, err := os.Create(localPath)
	if err != nil {

		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		_ = destFile.Close()
	}()

	_, err = io.Copy(destFile, reader)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

func promptToContinue(job *Job) bool {
	if len(job.Items) == 0 {
		log.Println("No files to download.")
		return false
	}

	firstFile := job.Items[0]

	localPath, err := resolveRelativePath(firstFile.Path, job.SourceURL.Path, job.TargetDir)
	if err != nil {
		log.Fatalf("Error resolving path: %v", err)
	}

	fmt.Printf("\nFile %s://%s@%s%s will become %s\n", job.SourceURL.Scheme, job.SourceURL.User.Username(), job.SourceURL.Host, firstFile.Path, localPath)
	fmt.Print("Do you want to continue? (y/n): ")

	var response string
	_, _ = fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
}
