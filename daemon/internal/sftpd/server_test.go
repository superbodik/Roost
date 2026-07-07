package sftpd

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	sftppkg "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/yourorg/panel-daemon/internal/docker"
)

func TestSFTPEndToEnd(t *testing.T) {
	dataDir := t.TempDir()
	dockerManager, err := docker.NewManager("", dataDir)
	if err != nil {
		t.Fatalf("docker.NewManager: %v", err)
	}

	testUUID := uuid.New()
	volPath := dockerManager.ServerVolumePath(testUUID)
	if err := os.MkdirAll(volPath, 0755); err != nil {
		t.Fatalf("mkdir volPath: %v", err)
	}
	if err := os.WriteFile(filepath.Join(volPath, "existing.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	clientPub, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(clientPub)
	if err != nil {
		t.Fatalf("ssh.NewPublicKey: %v", err)
	}
	fingerprint := ssh.FingerprintSHA256(sshPub)

	panelStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["username"] != "alice.abcd1234" || req["fingerprint"] != fingerprint {
			json.NewEncoder(w).Encode(map[string]any{"allowed": false, "reason": "mismatch"})
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-daemon-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"allowed": true, "read_only": false, "server_uuid": testUUID.String(),
		})
	}))
	defer panelStub.Close()

	hostKeyPath := filepath.Join(t.TempDir(), "host_key")
	srv, err := NewServer(dockerManager, panelStub.URL, "test-daemon-token", hostKeyPath)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	go srv.ListenAndServe("127.0.0.1:22022")
	time.Sleep(200 * time.Millisecond)

	clientSigner, err := ssh.NewSignerFromKey(clientPriv)
	if err != nil {
		t.Fatalf("NewSignerFromKey: %v", err)
	}
	clientConfig := &ssh.ClientConfig{
		User:            "alice.abcd1234",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(clientSigner)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	conn, err := ssh.Dial("tcp", "127.0.0.1:22022", clientConfig)
	if err != nil {
		t.Fatalf("ssh.Dial failed: %v", err)
	}
	defer conn.Close()

	sftpClient, err := sftppkg.NewClient(conn)
	if err != nil {
		t.Fatalf("sftp.NewClient failed: %v", err)
	}
	defer sftpClient.Close()

	entries, err := sftpClient.ReadDir("/")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Name() == "existing.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected existing.txt in listing, got %+v", entries)
	}

	if err := sftpClient.Mkdir("/newdir"); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(volPath, "newdir")); err != nil {
		t.Fatalf("newdir was not created on disk: %v", err)
	}

	f, err := sftpClient.Create("/newdir/uploaded.txt")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := f.Write([]byte("payload")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	f.Close()

	data, err := os.ReadFile(filepath.Join(volPath, "newdir", "uploaded.txt"))
	if err != nil || string(data) != "payload" {
		t.Fatalf("uploaded file mismatch: %v %q", err, data)
	}

	if err := sftpClient.Remove("/existing.txt"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(volPath, "existing.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected existing.txt to be deleted, stat err: %v", err)
	}

	if _, err := sftpClient.ReadDir("/../../etc"); err == nil {
		t.Fatalf("expected path traversal outside server root to be rejected")
	}
}
