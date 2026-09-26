package quota

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileCookiePersistence(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "p1")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	if loadProfileCookies(profile) != nil {
		t.Fatal("无 Cookie 文件时应返回 nil")
	}
	if err := os.WriteFile(filepath.Join(profile, profileCookieFile), []byte(`[{"name":"serviceToken","value":"x"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loadProfileCookies(profile); len(got) != 1 || got[0]["name"] != "serviceToken" {
		t.Fatalf("Cookie 读取不正确: %+v", got)
	}

	for _, name := range []string{"DevToolsActivePort", "SingletonLock", "SingletonCookie", "SingletonSocket"} {
		if err := os.WriteFile(filepath.Join(profile, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cleanProfileLocks(profile)
	for _, name := range []string{"DevToolsActivePort", "SingletonLock", "SingletonCookie", "SingletonSocket"} {
		if _, err := os.Stat(filepath.Join(profile, name)); !os.IsNotExist(err) {
			t.Fatalf("锁文件未清理: %s", name)
		}
	}
	if len(loadProfileCookies(profile)) != 1 {
		t.Fatal("清理锁文件不应删除 Cookie 文件")
	}
}

func TestBrowserSessionPersistsCookie(t *testing.T) {
	if _, err := browserExecutable(); err != nil {
		t.Skip(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "quota-test", Value: "signed-in", Path: "/", MaxAge: 3600})
			fmt.Fprint(w, "<html>logged in</html>")
		case "/home":
			fmt.Fprint(w, "<html>home</html>")
		case "/usage":
			if c, err := r.Cookie("quota-test"); err == nil && c.Value == "signed-in" {
				fmt.Fprint(w, `{"used":25}`)
			} else {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	script := `(async()=>{const r=await fetch('/usage',{credentials:'include'});return JSON.stringify({status:r.status,body:await r.json()})})()`
	manager := NewBrowserSessionManager(root)
	first, err := manager.Run(17, server.URL+"/login", script)
	if err != nil {
		manager.Close()
		t.Fatalf("首次查询失败: %v", err)
	}
	if !strings.Contains(first, `"used":25`) {
		manager.Close()
		t.Fatalf("首次额度无效: %s", first)
	}
	manager.Close()
	manager = NewBrowserSessionManager(root)
	defer manager.Close()
	second, err := manager.Run(17, server.URL+"/home", script)
	if err != nil {
		t.Fatalf("重启后查询失败: %v", err)
	}
	if !strings.Contains(second, `"used":25`) {
		t.Fatalf("登录会话未保留: %s", second)
	}
}
