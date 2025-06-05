package main

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

type FTPConnectorFactory struct{}

func (f *FTPConnectorFactory) Accept(u *url.URL) bool {
	return u.Scheme == "ftp"
}

func (f *FTPConnectorFactory) Create(u *url.URL, password []byte) (Connector, error) {
	return NewFTPConnector(u, password)
}

func (f *FTPConnectorFactory) Name() string {
	return "ftp"
}

type FTPConnector struct {
	client *ftp.ServerConn
	creds  *Credentials // Store credentials for possible reconnection
}

func NewFTPConnector(u *url.URL, password []byte) (*FTPConnector, error) {
	c, err := ftp.Dial(u.Host)
	if err != nil {
		return nil, err
	}

	err = c.Login(u.User.Username(), string(password))
	if err != nil {
		c.Quit() // Close connection on login failure
		return nil, err
	}

	// Create a new copy of the password to avoid issues with slice bounds
	passwordCopy := make([]byte, len(password))
	copy(passwordCopy, password)

	creds := &Credentials{
		username: u.User.Username(),
		password: passwordCopy,
	}

	return &FTPConnector{
		client: c,
		creds:  creds,
	}, nil
}

func (f *FTPConnector) ListFilesRecursively(base string) ([]string, error) {
	log.Printf("Preparing files list for %s", base)
	var files []string

	// Use map to prevent cycles or revisits
	visited := make(map[string]bool)

	// Clean base
	base = path.Clean(base)

	var walk func(current string) error
	walk = func(current string) error {
		// Avoid infinite recursion
		if visited[current] {
			log.Printf("Skipping already visited path: %s", current)
			return nil
		}
		visited[current] = true

		log.Printf("Listing: %s", current)
		entries, err := f.client.List(current)
		if err != nil {
			return err
		}

		for _, e := range entries {
			if e.Name == "." || e.Name == ".." {
				continue
			}

			full := path.Join(current, e.Name)
			cleaned := path.Clean(full)

			if !strings.HasPrefix(cleaned, base) {
				log.Printf("Skipping path outside base: %s", cleaned)
				continue
			}

			log.Printf("   Found %s (%v)", cleaned, e.Type)
			if e.Type == ftp.EntryTypeFile {
				files = append(files, cleaned)
			} else if e.Type == ftp.EntryTypeFolder {
				if err := walk(cleaned); err != nil {
					return err
				}
			}
		}

		return nil
	}

	err := walk(base)
	return files, err
}

func (f *FTPConnector) DownloadFile(remotePath, localPath string) error {
	var err error
	for attempts := 0; attempts < 3; attempts++ {
		err = f.downloadFileOnce(remotePath, localPath)
		if err == nil {
			return nil
		}
		time.Sleep(time.Second * time.Duration(attempts+1))
	}
	return fmt.Errorf("failed after 3 attempts: %w", err)
}

func (f *FTPConnector) downloadFileOnce(remotePath, localPath string) error {
	r, err := f.client.Retr(remotePath)
	if err != nil {
		return err
	}
	defer r.Close()
	os.MkdirAll(filepath.Dir(localPath), 0755)
	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, r)
	return err
}

func (f *FTPConnector) Close() error {
	if f.creds != nil {
		f.creds.Clear()
	}
	return f.client.Quit()
}
