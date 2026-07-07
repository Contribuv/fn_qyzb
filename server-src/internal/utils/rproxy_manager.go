package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"qyzb-server/internal/database"
)

type RProxyManager struct {
	cmd        *exec.Cmd
	apiPort    int
	apiBaseURL string
	configFile string
	mu         sync.Mutex
	running    bool
}

type ProxyConfig struct {
	Domain      string `json:"domain"`
	Port        int    `json:"port"`
	CertPath    string `json:"cert_path"`
	KeyPath     string `json:"key_path"`
	BackendAddr string `json:"backend_addr"`
	GzipEnabled bool   `json:"gzip_enabled"`
	HstsEnabled bool   `json:"hsts_enabled"`
	Timeout     int    `json:"timeout"`
	ApiPort     int    `json:"api_port"`
}

type ProxyStatus struct {
	Running   bool   `json:"running"`
	Domain    string `json:"domain"`
	Port      int    `json:"port"`
	StartedAt string `json:"started_at"`
}

var rproxyManager *RProxyManager
var rproxyOnce sync.Once

func GetRProxyManager() *RProxyManager {
	rproxyOnce.Do(func() {
		dataDir := database.GetDataDir()
		rproxyManager = &RProxyManager{
			configFile: filepath.Join(dataDir, "rproxy-config.json"),
		}
		cfg := rproxyManager.loadConfig()
		rproxyManager.recoverStatus(cfg)
	})
	return rproxyManager
}

func (m *RProxyManager) getBinaryPath() string {
	if p := os.Getenv("RPROXY_BIN_PATH"); p != "" {
		return p
	}

	baseDir := getAppDir()
	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}

	binName := fmt.Sprintf("qyzb-rproxy-%s", arch)
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	return filepath.Join(baseDir, "..", "rproxy", binName)
}

func getAppDir() string {
	exe, err := os.Executable()
	if err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

func (m *RProxyManager) Start(domain string, port int, certPath, keyPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.Stop()
	}

	available, err := CheckPortAvailable(port)
	if err != nil {
		return fmt.Errorf("端口检测失败: %v", err)
	}
	if !available {
		suggested := SuggestPort(port)
		if suggested > 0 {
			return fmt.Errorf("端口 %d 已被占用，建议尝试端口 %d", port, suggested)
		}
		return fmt.Errorf("端口 %d 已被占用，请更换端口后重试", port)
	}

	backendPort := getBackendPort()

	binPath := m.getBinaryPath()
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return fmt.Errorf("反代二进制不存在: %s", binPath)
	}

	apiPort, err := getFreePort()
	if err != nil {
		return fmt.Errorf("获取可用端口失败: %v", err)
	}

	cmd := exec.Command(binPath, "--api-port", strconv.Itoa(apiPort))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动反代失败: %v", err)
	}

	m.cmd = cmd
	m.apiPort = apiPort
	m.apiBaseURL = fmt.Sprintf("http://127.0.0.1:%d", apiPort)

	time.Sleep(500 * time.Millisecond)

	config := ProxyConfig{
		Domain:      domain,
		Port:        port,
		CertPath:    certPath,
		KeyPath:     keyPath,
		BackendAddr: fmt.Sprintf("http://127.0.0.1:%s", backendPort),
		GzipEnabled: true,
		HstsEnabled: true,
		Timeout:     600,
		ApiPort:     apiPort,
	}

	if err := m.callAPI("/start", "POST", config, nil); err != nil {
		cmd.Process.Kill()
		m.cmd = nil
		m.running = false
		return fmt.Errorf("启动反代失败: %v", err)
	}

	m.running = true
	m.saveConfig(config)

	go m.watchProcess()

	return nil
}

func (m *RProxyManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.cmd == nil {
		return nil
	}

	m.callAPI("/stop", "POST", nil, nil)
	time.Sleep(300 * time.Millisecond)

	if m.cmd.Process != nil {
		m.cmd.Process.Kill()
	}
	m.cmd = nil
	m.running = false
	return nil
}

func (m *RProxyManager) GetStatus() (*ProxyStatus, error) {
	m.mu.Lock()
	running := m.running
	m.mu.Unlock()

	if !running {
		return &ProxyStatus{Running: false}, nil
	}

	var status ProxyStatus
	if err := m.callAPI("/status", "GET", nil, &status); err != nil {
		return &ProxyStatus{Running: false}, nil
	}
	return &status, nil
}

func (m *RProxyManager) GetLogs(limit int) ([]map[string]interface{}, error) {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return []map[string]interface{}{}, nil
	}
	m.mu.Unlock()

	var result struct {
		Logs []map[string]interface{} `json:"logs"`
	}
	if err := m.callAPI(fmt.Sprintf("/logs?limit=%d", limit), "GET", nil, &result); err != nil {
		return nil, err
	}
	return result.Logs, nil
}

func (m *RProxyManager) ReloadCert(certPath, keyPath string) error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return fmt.Errorf("反代未运行")
	}
	m.mu.Unlock()

	req := map[string]string{
		"cert_path": certPath,
		"key_path":  keyPath,
	}
	return m.callAPI("/reload-cert", "POST", req, nil)
}

func (m *RProxyManager) callAPI(path, method string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, m.apiBaseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, result)
	}

	return nil
}

func (m *RProxyManager) watchProcess() {
	if m.cmd != nil {
		m.cmd.Wait()
	}
	m.mu.Lock()
	m.running = false
	m.cmd = nil
	m.mu.Unlock()
}

func (m *RProxyManager) saveConfig(config ProxyConfig) {
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(m.configFile, data, 0644)
}

func (m *RProxyManager) loadConfig() ProxyConfig {
	var config ProxyConfig
	data, err := os.ReadFile(m.configFile)
	if err == nil {
		json.Unmarshal(data, &config)
	}
	return config
}

func (m *RProxyManager) recoverStatus(cfg ProxyConfig) {
	if cfg.ApiPort <= 0 || cfg.Port <= 0 {
		return
	}

	apiURL := fmt.Sprintf("http://127.0.0.1:%d/status", cfg.ApiPort)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		m.mu.Lock()
		m.running = true
		m.apiPort = cfg.ApiPort
		m.apiBaseURL = fmt.Sprintf("http://127.0.0.1:%d", cfg.ApiPort)
		m.mu.Unlock()
		log.Printf("[RProxy] 检测到反代已在运行，端口: %d", cfg.Port)
	}
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func getBackendPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	if p := os.Getenv("wizard_app_port"); p != "" {
		return p
	}
	return "3008"
}

func CheckPortAvailable(port int) (bool, error) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false, nil
	}
	ln.Close()
	return true, nil
}

func SuggestPort(preferredPort int) int {
	for i := 0; i < 20; i++ {
		port := preferredPort + i
		if available, _ := CheckPortAvailable(port); available {
			return port
		}
	}
	return 0
}

func (m *RProxyManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}
