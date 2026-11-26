// 전수검사 데이터 분석 프로그램 - Windows 실행 파일
// Made By J
//
// 동작 방식
//  1. 프로그램 화면(app.html)을 exe 안에 넣어 두고, 이 PC 안에서만(127.0.0.1) 열리는 작은 서버로 띄운다.
//  2. 윈도우에 기본으로 깔린 Edge(없으면 Chrome)를 주소창 없는 '앱 창'으로 열어 화면을 보여준다.
//  3. 검사 데이터는 exe 옆 data\inspection_data.json 파일에 저장하고, 하루 한 번 data\backup 에 사본을 남긴다.
//  4. 프로그램 창을 닫으면 서버도 같이 끝난다. 인터넷 연결은 쓰지 않는다.
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

//go:embed app.html
var appHTML []byte

//go:embed app.ico
var appIcon []byte

const appID = "JINSPECT-1"

// 실행 파일(런처) 버전. 저장소 version.json 의 launcher 값이 이보다 높으면 exe 를 새로 받는다.
const launcherVersion = "1.4.0"

// 업데이트를 받아 오는 곳: 저장소 JJun-2/HBS 안의 '전수검사분석' 폴더
// (관리자가 여기에 올리면 사용자 프로그램이 켤 때 자동으로 받음. 배포용 설정에서 바꿀 수 있음)
const updDefault = "https://raw.githubusercontent.com/JJun-2/HBS/main/%EC%A0%84%EC%88%98%EA%B2%80%EC%82%AC%EB%B6%84%EC%84%9D/"

// 데이터 파일의 설정(배포용 설정)에서 업데이트 주소와 사용 여부를 읽는다
func updSettings() (string, bool) {
	base, auto := updDefault, true
	b, err := os.ReadFile(dataFile)
	if err != nil {
		return base, auto
	}
	var d struct {
		Meta struct {
			Settings map[string]interface{} `json:"settings"`
		} `json:"meta"`
	}
	if json.Unmarshal(b, &d) != nil {
		return base, auto
	}
	if v, ok := d.Meta.Settings["updBase"].(string); ok && strings.HasPrefix(v, "https://") {
		if !strings.HasSuffix(v, "/") {
			v += "/"
		}
		base = v
	}
	if v, ok := d.Meta.Settings["autoUpdate"].(bool); ok {
		auto = v
	}
	return base, auto
}

var (
	dataDir   string
	dataFile  string
	backupDir string
	mu        sync.Mutex
	lastPing  = time.Now()
	pingMu    sync.Mutex

	htmlMu     sync.RWMutex
	curHTML    []byte
	curVersion string
	appCache   string
	token      string
	origin     string
	exePending string
	updatedTo  string
	updNotes   string
)

type verInfo struct {
	Version  string   `json:"version"`
	Notes    []string `json:"notes"`
	App      string   `json:"app"`
	AppSha   string   `json:"app_sha256"`
	Launcher string   `json:"launcher"`
	Exe      string   `json:"exe"`
	ExeSha   string   `json:"exe_sha256"`
}

func fetchURL(url string, timeout time.Duration, max int64) ([]byte, error) {
	c := http.Client{Timeout: timeout}
	resp, err := c.Get(url + "?t=" + strconv.FormatInt(time.Now().UnixNano(), 10))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

func shaHex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// 프로그램을 켤 때 저장소에서 새 버전을 확인해 화면(app.html)을 먼저 바꿔 둔다.
// 실행 파일 자체가 바뀐 경우는 뒤에서 받아 교체하고, 다음 실행부터 적용된다.
func startupUpdate() {
	updBase, auto := updSettings()
	if !auto {
		return
	}
	b, err := fetchURL(updBase+"version.json", 4*time.Second, 1<<20)
	if err != nil {
		return
	}
	var v verInfo
	if json.Unmarshal(b, &v) != nil || v.Version == "" {
		return
	}
	htmlMu.RLock()
	cur := curVersion
	htmlMu.RUnlock()
	if v.App != "" && verCmp(v.Version, cur) > 0 {
		hb, err := fetchURL(updBase+v.App, 15*time.Second, 30<<20)
		if err == nil && (v.AppSha == "" || shaHex(hb) == strings.ToLower(v.AppSha)) && htmlVersion(hb) == v.Version && bytes.Contains(hb, []byte("</html>")) {
			os.MkdirAll(filepath.Dir(appCache), 0755)
			if writeAtomic(appCache, hb) == nil {
				htmlMu.Lock()
				curHTML, curVersion = hb, v.Version
				updatedTo, updNotes = v.Version, strings.Join(v.Notes, "\n")
				htmlMu.Unlock()
			}
		}
	}
	if v.Exe != "" && v.Launcher != "" && verCmp(v.Launcher, launcherVersion) > 0 {
		go func() {
			eb, err := fetchURL(updBase+v.Exe, 5*time.Minute, 150<<20)
			if err != nil || len(eb) < 1024 || eb[0] != 'M' || eb[1] != 'Z' {
				return
			}
			if v.ExeSha != "" && shaHex(eb) != strings.ToLower(v.ExeSha) {
				return
			}
			if replaceExe(eb) == nil {
				exePending = v.Launcher
			}
		}()
	}
}

func replaceExe(b []byte) error {
	exe, _ := os.Executable()
	nw, old := exe+".new", exe+".old"
	if err := os.WriteFile(nw, b, 0755); err != nil {
		return err
	}
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(nw)
		return err
	}
	if err := os.Rename(nw, exe); err != nil {
		os.Rename(old, exe)
		return err
	}
	return nil
}

var verRe = regexp.MustCompile(`<meta name="app-version" content="([0-9.]+)"`)

func htmlVersion(b []byte) string {
	m := verRe.FindSubmatch(b)
	if m == nil {
		return ""
	}
	return string(m[1])
}

// a>b 이면 1, 같으면 0, 작으면 -1
func verCmp(a, b string) int {
	x, y := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(x) || i < len(y); i++ {
		var p, q int
		if i < len(x) {
			p, _ = strconv.Atoi(x[i])
		}
		if i < len(y) {
			q, _ = strconv.Atoi(y[i])
		}
		if p != q {
			if p > q {
				return 1
			}
			return -1
		}
	}
	return 0
}

// 내장 화면과 data\app\app.html(업데이트로 받은 화면) 중 새 버전을 쓴다
func loadHTML() {
	curHTML, curVersion = appHTML, htmlVersion(appHTML)
	b, err := os.ReadFile(appCache)
	if err != nil {
		return
	}
	v := htmlVersion(b)
	if v != "" && verCmp(v, curVersion) > 0 && bytes.Contains(b, []byte("</html>")) {
		curHTML, curVersion = b, v
	}
}

func authOK(w http.ResponseWriter, r *http.Request) bool {
	o := r.Header.Get("Origin")
	if r.Header.Get("X-Token") != token || (o != "" && o != origin) {
		http.Error(w, "forbidden", 403)
		return false
	}
	return true
}

func writeAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(path)
		return os.Rename(tmp, path)
	}
	return nil
}

func handleUpdateHTML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !authOK(w, r) {
		return
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 30<<20))
	if err != nil {
		http.Error(w, "read", 400)
		return
	}
	v := htmlVersion(b)
	if v == "" || v != r.Header.Get("X-Version") || !bytes.Contains(b, []byte("</html>")) {
		http.Error(w, "화면 파일 형식이 맞지 않습니다", 400)
		return
	}
	os.MkdirAll(filepath.Dir(appCache), 0755)
	if err := writeAtomic(appCache, b); err != nil {
		http.Error(w, "저장 실패: "+err.Error(), 500)
		return
	}
	htmlMu.Lock()
	curHTML, curVersion = b, v
	htmlMu.Unlock()
	w.Write([]byte(`{"ok":true}`))
}

func handleUpdateExe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !authOK(w, r) {
		return
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 150<<20))
	if err != nil || len(b) < 1024 || b[0] != 'M' || b[1] != 'Z' {
		http.Error(w, "실행 파일 형식이 맞지 않습니다", 400)
		return
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != strings.ToLower(r.Header.Get("X-Sha256")) {
		http.Error(w, "실행 파일 검증 실패", 400)
		return
	}
	if err := replaceExe(b); err != nil {
		http.Error(w, "실행 파일 교체 실패: "+err.Error(), 500)
		return
	}
	exePending = r.Header.Get("X-Version")
	w.Write([]byte(`{"ok":true}`))
}

func msgBox(text string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString("전수검사 데이터 분석 프로그램")
	proc.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x40)
}

func writable(dir string) bool {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	f := filepath.Join(dir, ".write_test")
	if err := os.WriteFile(f, []byte("ok"), 0644); err != nil {
		return false
	}
	os.Remove(f)
	return true
}

func setupDirs() {
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	dataDir = filepath.Join(base, "data")
	if !writable(dataDir) {
		// exe가 쓰기 금지 폴더(Program Files 등)에 있으면 사용자 폴더로
		la := os.Getenv("LOCALAPPDATA")
		if la == "" {
			la = os.TempDir()
		}
		dataDir = filepath.Join(la, "전수검사분석", "data")
		os.MkdirAll(dataDir, 0755)
	}
	dataFile = filepath.Join(dataDir, "inspection_data.json")
	appCache = filepath.Join(dataDir, "app", "app.html")
	backupDir = filepath.Join(dataDir, "backup")
	os.MkdirAll(backupDir, 0755)
}

func dailyBackup() {
	if _, err := os.Stat(dataFile); err != nil {
		return
	}
	name := filepath.Join(backupDir, "inspection_data_"+time.Now().Format("20060102")+".json")
	if _, err := os.Stat(name); err == nil {
		return
	}
	src, err := os.ReadFile(dataFile)
	if err != nil || len(src) < 10 {
		return
	}
	os.WriteFile(name, src, 0644)
	// 최근 60개만 보관
	ents, _ := os.ReadDir(backupDir)
	var names []string
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), "inspection_data_") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for len(names) > 60 {
		os.Remove(filepath.Join(backupDir, names[0]))
		names = names[1:]
	}
}

func handleDB(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	mu.Lock()
	defer mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		b, err := os.ReadFile(dataFile)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("null"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	case http.MethodPost:
		if !authOK(w, r) {
			return
		}
		b, err := io.ReadAll(io.LimitReader(r.Body, 500<<20))
		if err != nil || !json.Valid(b) {
			http.Error(w, "bad data", 400)
			return
		}
		dailyBackup()
		tmp := dataFile + ".tmp"
		if err := os.WriteFile(tmp, b, 0644); err != nil {
			http.Error(w, "저장 실패: "+err.Error(), 500)
			return
		}
		if err := os.Rename(tmp, dataFile); err != nil {
			// 윈도우에서 대상 파일이 잠겨 있을 때 대비
			os.Remove(dataFile)
			if err2 := os.Rename(tmp, dataFile); err2 != nil {
				http.Error(w, "저장 실패: "+err2.Error(), 500)
				return
			}
		}
		w.Write([]byte(`{"ok":true}`))
	default:
		http.Error(w, "method", 405)
	}
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	htmlMu.RLock()
	v := curVersion
	defer htmlMu.RUnlock()
	json.NewEncoder(w).Encode(map[string]string{"app": appID, "dataFile": dataFile, "backupDir": backupDir,
		"launcher": launcherVersion, "appVersion": v, "exePending": exePending, "updatedTo": updatedTo, "updateNotes": updNotes})
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	pingMu.Lock()
	lastPing = time.Now()
	pingMu.Unlock()
	w.Write([]byte(appID))
}

func handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !authOK(w, r) {
		return
	}
	exec.Command("explorer.exe", dataDir).Start()
	w.Write([]byte(`{"ok":true}`))
}

func findBrowser() string {
	var cands []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		b := os.Getenv(env)
		if b == "" {
			continue
		}
		cands = append(cands,
			filepath.Join(b, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(b, "Google", "Chrome", "Application", "chrome.exe"))
	}
	// Edge 먼저
	sort.SliceStable(cands, func(i, j int) bool {
		return strings.Contains(cands[i], "Edge") && !strings.Contains(cands[j], "Edge")
	})
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func openWindow(url string) (*exec.Cmd, bool) {
	br := findBrowser()
	if br == "" {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		return nil, false
	}
	la := os.Getenv("LOCALAPPDATA")
	if la == "" {
		la = os.TempDir()
	}
	profile := filepath.Join(la, "전수검사분석", "window")
	cmd := exec.Command(br,
		"--app="+url,
		"--user-data-dir="+profile,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-sync",
		"--window-size=1440,920",
	)
	if err := cmd.Start(); err != nil {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		return nil, false
	}
	return cmd, true
}

// 이미 실행 중이면 그 프로그램에 새 창만 띄운다
func alreadyRunning(port int) bool {
	c := http.Client{Timeout: 700 * time.Millisecond}
	resp, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d/api/ping", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b) == appID
}

func main() {
	setupDirs()
	if exe, err := os.Executable(); err == nil {
		os.Remove(exe + ".old") // 지난번 업데이트로 남은 이전 실행 파일 정리
	}
	loadHTML()
	tb := make([]byte, 16)
	rand.Read(tb)
	token = hex.EncodeToString(tb)
	var ln net.Listener
	var port int
	for p := 47811; p < 47841; p++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			ln, port = l, p
			break
		}
		if alreadyRunning(p) {
			openWindow(fmt.Sprintf("http://127.0.0.1:%d/", p))
			return
		}
	}
	if ln == nil {
		msgBox("프로그램을 시작하지 못했습니다. (사용 가능한 내부 포트 없음)\nPC를 다시 시작한 뒤 실행해 보세요.")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		htmlMu.RLock()
		page := curHTML
		htmlMu.RUnlock()
		w.Write(bytes.Replace(page, []byte("__JTOKEN__"), []byte(token), 1))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(appIcon)
	})
	mux.HandleFunc("/api/db", handleDB)
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/ping", handlePing)
	mux.HandleFunc("/api/open-folder", handleOpenFolder)
	mux.HandleFunc("/api/update/html", handleUpdateHTML)
	mux.HandleFunc("/api/update/exe", handleUpdateExe)
	origin = fmt.Sprintf("http://127.0.0.1:%d", port)
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	// 새 버전 확인 (인터넷이 안 되면 최대 몇 초 안에 건너뜀)
	done := make(chan struct{})
	go func() { startupUpdate(); close(done) }()
	select {
	case <-done:
	case <-time.After(6 * time.Second):
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	start := time.Now()
	cmd, ok := openWindow(url)
	if ok {
		cmd.Wait()
		// 브라우저가 바로 끝났으면(다른 창에 넘긴 경우) 화면 신호로 종료 판단
		if time.Since(start) > 8*time.Second {
			return
		}
	}
	// 화면이 열려 있는 동안 주기적으로 ping 이 온다. 10분 넘게 끊기면 종료.
	for {
		time.Sleep(30 * time.Second)
		pingMu.Lock()
		idle := time.Since(lastPing)
		pingMu.Unlock()
		if idle > 10*time.Minute {
			return
		}
	}
}
