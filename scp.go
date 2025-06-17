package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

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

// knownHosts stores already verified host fingerprints
var (
	knownHosts   = make(map[string]string)
	knownHostsMu sync.Mutex
)

var hostKeyVerificationCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
	fingerprint := ssh.FingerprintSHA256(key)

	knownHostsMu.Lock()
	storedFingerprint, exists := knownHosts[hostname]
	knownHostsMu.Unlock()
	if exists && storedFingerprint == fingerprint {
		return nil
	}

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
		// Save the verified fingerprint
		knownHostsMu.Lock()
		knownHosts[hostname] = fingerprint
		knownHostsMu.Unlock()
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

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	if err := session.Start(fmt.Sprintf("scp -f %s", remotePath)); err != nil {
		return fmt.Errorf("failed to start scp command: %w", err)
	}

	writer := bufio.NewWriter(stdin)
	reader := bufio.NewReader(stdout)

	// send initial null byte
	if err := writeByte(writer, 0); err != nil {
		return fmt.Errorf("failed to write initial null byte: %w", err)
	}

	// read file metadata line (C0664 999999999 test.txt)
	//                          └─┬─┘ └───┬───┘ └───┬───┘
	//                            │       │         │
	//                           mode    size    filename
	line, err := reader.ReadString('\n')
	if err != nil {
		slurp, _ := io.ReadAll(stderr)
		return fmt.Errorf("failed to read file metadata: %w (%s)", err, string(slurp))
	}

	//var mode string
	var size int64
	//var filename string
	fields := strings.SplitN(strings.TrimSpace(line), " ", 3)
	if len(fields) != 3 {
		return fmt.Errorf("unexpected SCP metadata format: %q", line)
	}

	// @todo respect mode?
	//mode = strings.TrimPrefix(fields[0], "C")
	size, err = strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid file size: %w", err)
	}

	//filename = fields[2]

	// send acknowledgment
	if err := writeByte(writer, 0); err != nil {
		return fmt.Errorf("failed to acknowledge metadata: %w", err)
	}

	// create limited reader for exact file content
	limited := io.LimitReader(reader, size)

	err = saveRemoteFile(remotePath, localBasePath, basePath, limited)
	if err != nil {
		return err
	}

	// read and discard single byte (remote confirmation)
	if b, err := reader.ReadByte(); err != nil || b != 0 {
		return fmt.Errorf("unexpected trailing byte: %v", b)
	}

	// send final null byte
	if err := writeByte(writer, 0); err != nil {
		return fmt.Errorf("failed to send final null byte: %w", err)
	}

	return session.Wait()
}

func (s *SCPConnector) Close() error {
	if s.creds != nil {
		s.creds.Clear()
	}
	return s.client.Close()
}

func writeByte(w *bufio.Writer, b byte) error {
	if _, err := w.Write([]byte{b}); err != nil {
		return err
	}
	return w.Flush()
}
