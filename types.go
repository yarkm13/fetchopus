package main

import (
	"net/url"
)

// Connector interface for remote file operations
type Connector interface {
	ListFilesRecursively(base string) ([]string, error)
	DownloadFile(remotePath, localPath string) error
	Close() error
}

// ConnectorFactory interface for creating connectors
type ConnectorFactory interface {
	Accept(u *url.URL) bool
	Create(u *url.URL, password []byte) (Connector, error)
	Name() string
}
