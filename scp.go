package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"path"
	"strings"

	"golang.org/x/crypto/ssh"
)

type SCPConnectorFactory struct{}

func (f *SCPConnectorFactory) Accept(u *url.URL) bool { return u.Scheme == "scp" }

func (f *SCPConnectorFactory) Create(u *url.URL, password []byte) (Connector, error) {
	return NewSCPConnector(u, password)
}

func (f *SCPConnectorFactory) Name() string { return "scp" }

type SCPConnector struct {
	client *ssh.Client
	creds  *Credentials
}

var hostKeyVerificationCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
	fingerprint := ssh.FingerprintSHA256(key)

	fmt.Printf("\nThe authenticity of host '%s' can't be established.\n", hostname)
	fmt.Printf("%s key fingerprint is %s\n", key.Type(), fingerprint)
	fmt.Print("Are you sure you want to continue connecting (yes/no)? ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "yes" || response == "y" {
		return nil
	}

	return fmt.Errorf("host key verification rejected by user")
}

func NewSCPConnector(u *url.URL, password []byte) (*SCPConnector, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(string(password))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	} else {
		fmt.Printf("private key decoded successfully\n")
	}

	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: u.User.Username(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyVerificationCallback,
	}

	client, err := ssh.Dial("tcp", u.Host, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	passwordCopy := make([]byte, len(password))
	copy(passwordCopy, password)

	creds := &Credentials{
		username: u.User.Username(),
		password: passwordCopy,
	}

	return &SCPConnector{
		client: client,
		creds:  creds,
	}, nil
}

func (s *SCPConnector) ListFilesRecursively(base string) ([]string, error) {
	var files []string
	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	fmt.Println("Listing files in ", base, "...")
	cmd := fmt.Sprintf("find %s -type f", base)
	output, err := session.Output(cmd)
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			cleaned := path.Clean(line)
			if strings.HasPrefix(cleaned, base) {
				files = append(files, cleaned)
			}
		}
	}

	return files, nil
}

func (s *SCPConnector) DownloadFile(remotePath, localBasePath string, basePath string) error {
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		session.Stdout = pw
		if err := session.Run(fmt.Sprintf("scp -f %s", remotePath)); err != nil {
			log.Printf("SCP command failed: %v", err)
		}
	}()

	return saveRemoteFile(remotePath, localBasePath, basePath, pr)
}

func (s *SCPConnector) Close() error {
	if s.creds != nil {
		s.creds.Clear()
	}
	return s.client.Close()
}
