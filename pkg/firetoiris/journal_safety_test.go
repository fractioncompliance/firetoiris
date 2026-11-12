package firetoiris

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJournalProcessHelper(t *testing.T) {
	if os.Getenv("FIRETOIRIS_TEST_HELPER") != "1" {
		return
	}
	j, err := OpenJournal(os.Getenv("FIRETOIRIS_TEST_JOURNAL"))
	if err != nil {
		t.Fatal(err)
	}
	j.State = State{Version: Version}
	if err = j.save(); err != nil {
		t.Fatal(err)
	}
	fmt.Println("ready")
	time.Sleep(time.Minute)
}

func TestJournalLockRecoversAfterProcessTermination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestJournalProcessHelper$")
	cmd.Env = append(os.Environ(), "FIRETOIRIS_TEST_HELPER=1", "FIRETOIRIS_TEST_JOURNAL="+path)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	scanner := bufio.NewScanner(pipe)
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "ready" {
		t.Fatal("helper failed")
	}
	if j, err := OpenJournal(path); err == nil {
		j.Close()
		t.Fatal("concurrent process acquired lock")
	}
	if err = cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	j, err := OpenJournal(path)
	if err != nil {
		t.Fatalf("crash left stale lock: %v", err)
	}
	defer j.Close()
	if j.State.Version != Version {
		t.Fatal("durable state missing")
	}
}
