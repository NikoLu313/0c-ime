//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/0c-biggo/0c-ime/bridge/ipc"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/internal/settings"
	"github.com/Microsoft/go-winio"
)

// Exercise the saved layout through the actual Windows server transport. All
// paths and the named pipe are private to the test, including learned words.
func TestSavedHsuLayoutStartsWindowsServer(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "0c-server.exe")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	data := filepath.Join(dir, "data")
	if err := os.Mkdir(data, 0700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"lexicon.tsv": "你\tㄋㄧˇ\t1\n好\tㄏㄠˇ\t1\n你好\tㄋㄧˇ ㄏㄠˇ\t1\n",
		"lm.tsv":      "你\t0.1\n好\t0.1\n你好\t0.2\n",
	} {
		if err := os.WriteFile(filepath.Join(data, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	config := settings.Default()
	config.KeyboardLayout = settings.LayoutHsu
	config.Selection = settings.Statistical
	config.MixedInput = true // saved preference must survive the layout's restriction
	configPath := filepath.Join(dir, settings.FileName)
	if err := settings.Save(configPath, config); err != nil {
		t.Fatal(err)
	}
	pipe := fmt.Sprintf(`\\.\pipe\0c-test-hsu-%d-%d`, os.Getpid(), time.Now().UnixNano())
	logPath := filepath.Join(dir, "server.log")
	log, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	server := exec.Command(binary, "-data", data, "-user", filepath.Join(dir, "user.json"), "-socket", pipe, "-system-python", "")
	server.Stdout, server.Stderr = log, log
	if err := server.Start(); err != nil {
		log.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		server.Process.Kill()
		server.Wait()
		log.Close()
		if t.Failed() {
			body, _ := os.ReadFile(logPath)
			t.Logf("server log:\n%s", body)
		}
	})
	timeout := 200 * time.Millisecond
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if conn, err := winio.DialPipe(pipe, &timeout); err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("引擎沒有開啟隔離測試的 named pipe")
		}
	}
	conn, err := winio.DialPipe(pipe, &timeout)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	call := func(env ipc.Envelope) ipc.Reply {
		t.Helper()
		if err := ipc.WriteFrame(conn, env); err != nil {
			t.Fatal(err)
		}
		var reply ipc.Reply
		if err := ipc.ReadFrame(conn, &reply); err != nil {
			t.Fatal(err)
		}
		if !reply.OK {
			t.Fatalf("IPC request failed: %+v", reply)
		}
		return reply
	}
	info := call(ipc.Envelope{Op: ipc.OpHealth}).Info
	if info["keyboard_layout"] != "hsu" || info["mixed_input"] != "false" || info["emoji_shortcut"] != "equals" {
		t.Fatalf("儲存的許氏排列未正確套用：%+v", info)
	}
	id := call(ipc.Envelope{Op: ipc.OpOpen, Version: proto.Version}).Session
	handle := func(req proto.Request) *proto.Snapshot {
		t.Helper()
		snap := call(ipc.Envelope{Op: ipc.OpHandle, Session: id, Req: &req}).Snapshot
		if snap == nil {
			t.Fatal("missing snapshot")
		}
		return snap
	}
	if snap := handle(proto.Request{Type: proto.Key, Key: "1"}); snap.Handled {
		t.Fatalf("許氏排列閒置時數字應交還應用程式：%+v", snap)
	}
	for _, key := range "nefhwf" { // ㄋㄧˇ ㄏㄠˇ
		handle(proto.Request{Type: proto.Key, Key: string(key)})
	}
	if snap := handle(proto.Request{Type: proto.Commit}); snap.Commit != "你好" {
		t.Fatalf("許氏排列應輸入你好：%+v", snap)
	}
	if snap := handle(proto.Request{Type: proto.Key, Key: "="}); snap.Candidates == nil || len(snap.Candidates.Items) < 2 {
		t.Fatalf("許氏排列應能使用 Emoji 選單：%+v", snap)
	}
	if saved, err := settings.Load(configPath); err != nil || !saved.MixedInput || saved.KeyboardLayout != settings.LayoutHsu {
		t.Fatalf("引擎不可覆寫使用者保留的設定：%+v %v", saved, err)
	}
}
