// chromemanager/chromemanager.go
package chromemanager

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/shirou/gopsutil/v3/process"
)

type ChromeManager struct {
	BaseProfileDir   string
	BrowserPath      string
	DebugPort        int
	browserProcess   *exec.Cmd
	playwrightInst   *playwright.Playwright
	browser          playwright.Browser
	page             playwright.Page
	processPID       int
}

// NewChromeManager mirrors Python __init__
func NewChromeManager(baseProfileDir, browserPath string, debugPort int) (*ChromeManager, error) {
	if baseProfileDir == "" {
		if runtime.GOOS != "darwin" {
			baseProfileDir = `C:\ChromeProfiles`
		} else {
			home, _ := os.UserHomeDir()
			baseProfileDir = filepath.Join(home, "ChromeProfiles")
		}
	}
	if err := os.MkdirAll(baseProfileDir, 0755); err != nil {
		return nil, err
	}
	if browserPath == "" {
		var err error
		browserPath, err = findBrowserPath()
		if err != nil {
			return nil, err
		}
	}
	if debugPort == 0 {
		debugPort = 9222
	}

	return &ChromeManager{
		BaseProfileDir: baseProfileDir,
		BrowserPath:    browserPath,
		DebugPort:      debugPort,
	}, nil
}

// findBrowserPath = Python _find_browser_path()
func findBrowserPath() (string, error) {
	paths := []string{}
	system := runtime.GOOS

	if system == "darwin" {
		home, _ := os.UserHomeDir()
		paths = []string{
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			filepath.Join(home, "Applications/Brave Browser.app/Contents/MacOS/Brave Browser"),
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			filepath.Join(home, "Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"),
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			filepath.Join(home, "Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			filepath.Join(home, "Applications/Chromium.app/Contents/MacOS/Chromium"),
		}
	} else if system == "windows" {
		paths = []string{
			`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
			`C:\Program Files (x86)\BraveSoftware\Brave-Browser\Application\brave.exe`,
			filepath.Join(os.Getenv("LOCALAPPDATA"), `BraveSoftware\Brave-Browser\Application\brave.exe`),
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			filepath.Join(os.Getenv("LOCALAPPDATA"), `Microsoft\Edge\Application\msedge.exe`),
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			filepath.Join(os.Getenv("LOCALAPPDATA"), `Google\Chrome\Application\chrome.exe`),
			`C:\Program Files\Chromium\Application\chromium.exe`,
			`C:\Program Files (x86)\Chromium\Application\chromium.exe`,
			filepath.Join(os.Getenv("LOCALAPPDATA"), `Chromium\Application\chromium.exe`),
		}
	} else {
		home, _ := os.UserHomeDir()
		paths = []string{
			"/usr/bin/brave-browser", "/usr/bin/brave",
			"/usr/local/bin/brave-browser", "/usr/local/bin/brave",
			filepath.Join(home, ".local/bin/brave-browser"),
			"/usr/bin/microsoft-edge", "/usr/bin/microsoft-edge-stable",
			"/usr/local/bin/microsoft-edge",
			"/usr/bin/google-chrome", "/usr/bin/google-chrome-stable",
			"/usr/local/bin/google-chrome",
			"/usr/bin/chromium", "/usr/bin/chromium-browser",
			"/usr/local/bin/chromium",
			filepath.Join(home, ".local/bin/chromium"),
		}
	}

	fmt.Println("Checking browser paths:")
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			fmt.Printf(" - %s: Found\n", p)
			fmt.Printf("Selected browser at: %s\n", p)
			return p, nil
		}
		fmt.Printf(" - %s: Not found\n", p)
	}

	fmt.Println("Browser executable not found. Please input your browser path.")
	if system == "darwin" {
		fmt.Println("Example: /Applications/Brave Browser.app/Contents/MacOS/Brave Browser")
	} else if system == "windows" {
		fmt.Println(`Example: C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`)
	} else {
		fmt.Println("Example: /usr/bin/brave-browser")
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Enter browser path: ")
		if !scanner.Scan() {
			return "", fmt.Errorf("input error")
		}
		userPath := strings.TrimSpace(scanner.Text())
		userPath = strings.Trim(userPath, `"`)
		if userPath == "" {
			fmt.Println("Path cannot be empty. Please try again.")
			continue
		}
		if _, err := os.Stat(userPath); err != nil {
			fmt.Printf("File not found at: %s\n", userPath)
			retry := promptYesNo("Would you like to try again? (y/n): ")
			if !retry {
				return "", fmt.Errorf("browser path not found. Exiting...")
			}
			continue
		}
		base := strings.ToLower(filepath.Base(userPath))
		if strings.HasSuffix(base, "brave") || strings.HasSuffix(base, "brave.exe") ||
		   strings.HasSuffix(base, "msedge") || strings.HasSuffix(base, "msedge.exe") ||
		   strings.HasSuffix(base, "chrome") || strings.HasSuffix(base, "chrome.exe") ||
		   strings.HasSuffix(base, "chromium") || strings.HasSuffix(base, "chromium.exe") {
			fmt.Printf("Browser path verified: %s\n", userPath)
			return userPath, nil
		}
		fmt.Println("Path should point to Brave, Edge, Chrome, or Chromium executable. Please try again.")
	}
}

func promptYesNo(msg string) bool {
	fmt.Print(msg)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" ||
		       strings.ToLower(strings.TrimSpace(scanner.Text())) == "yes"
	}
	return false
}

// _is_port_open
func (cm *ChromeManager) isPortOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return true
	}
	conn.Close()
	return false
}

// _kill_child_processes
func (cm *ChromeManager) killChildProcesses(pid int) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		fmt.Printf("Process %d no longer exists\n", pid)
		return
	}
	children, _ := p.Children()
	for _, child := range children {
		child.Kill()
		fmt.Printf("Killed child process: %d\n", child.Pid)
	}
	p.Kill()
	fmt.Printf("Killed parent process: %d\n", pid)
}

// get_profile_path
func (cm *ChromeManager) GetProfilePath(name string) string {
	return filepath.Join(cm.BaseProfileDir, name)
}

// profile_exists
func (cm *ChromeManager) ProfileExists(name string) bool {
	_, err := os.Stat(cm.GetProfilePath(name))
	return err == nil
}

// setup_profile
func (cm *ChromeManager) SetupProfile(profileName, url, waitMessage string, headless bool) error {
	if waitMessage == "" {
		waitMessage = "Perform manual actions, then close the browser to save."
	}
	userDataDir := cm.GetProfilePath(profileName)
	os.MkdirAll(userDataDir, 0755)

	if !cm.isPortOpen(cm.DebugPort) {
		return fmt.Errorf("Port %d is in use. Choose another port.", cm.DebugPort)
	}

	args := []string{
		cm.BrowserPath,
		fmt.Sprintf("--remote-debugging-port=%d", cm.DebugPort),
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		"--no-first-run",
		"--no-default-browser-check",
	}
	if url != "" {
		args = append(args, url)
	}
	if headless {
		args = append(args, "--headless=new")
	}

	fmt.Printf("Starting browser for profile '%s'\n", profileName)
	fmt.Println(waitMessage)

	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		return err
	}
	cm.browserProcess = cmd
	cm.processPID = cmd.Process.Pid

	cmd.Wait()
	cm.CloseBrowser()
	fmt.Printf("Profile '%s' saved.\n", profileName)
	return nil
}

// connect_to_browser
func (cm *ChromeManager) ConnectToBrowser(profileName, url string, headless bool, timeout int) (playwright.Page, error) {
	if !cm.ProfileExists(profileName) {
		return nil, fmt.Errorf("Profile '%s' does not exist. Create it first.", profileName)
	}
	if !cm.isPortOpen(cm.DebugPort) {
		return nil, fmt.Errorf("Port %d is in use. Choose another port.", cm.DebugPort)
	}

	userDataDir := cm.GetProfilePath(profileName)
	args := []string{
		cm.BrowserPath,
		fmt.Sprintf("--remote-debugging-port=%d", cm.DebugPort),
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		"--no-first-run",
		"--no-default-browser-check",
	}
	if headless {
		args = append(args, "--headless=new")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	cm.browserProcess = cmd
	cm.processPID = cmd.Process.Pid
	time.Sleep(3 * time.Second)
	fmt.Printf("Browser started for profile '%s' (PID: %d).\n", profileName, cm.processPID)

	pw, err := playwright.Run()
	if err != nil {
		cm.CloseBrowser()
		return nil, err
	}
	cm.playwrightInst = pw

	browser, err := pw.Chromium.ConnectOverCDP(fmt.Sprintf("http://127.0.0.1:%d", cm.DebugPort))
	if err != nil {
		cm.CloseBrowser()
		return nil, err
	}
	cm.browser = browser

	contexts := browser.Contexts()
	var page playwright.Page
	if len(contexts) > 0 && len(contexts[0].Pages()) > 0 {
		page = contexts[0].Pages()[0]
	} else {
		page, err = browser.NewPage()
		if err != nil {
			cm.CloseBrowser()
			return nil, err
		}
	}
	cm.page = page

	if url != "" {
		_, err = page.Goto(url, playwright.PageGotoOptions{Timeout: playwright.Float(float64(timeout))})
		if err != nil {
			return nil, err
		}
		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateLoad, Timeout: playwright.Float(float64(timeout))})
	}
	return page, nil
}

// close_browser
func (cm *ChromeManager) CloseBrowser() {
	if cm.page != nil {
		cm.page.Close()
		fmt.Println("Closed page")
		cm.page = nil
	}
	if cm.browser != nil {
		cm.browser.Close()
		fmt.Println("Closed Playwright browser instance")
		cm.browser = nil
	}
	if cm.playwrightInst != nil {
		cm.playwrightInst.Stop()
		fmt.Println("Stopped Playwright instance")
		cm.playwrightInst = nil
	}
	if cm.browserProcess != nil && cm.processPID > 0 {
		cm.killChildProcesses(cm.processPID)
		cm.browserProcess = nil
		cm.processPID = 0
	}
	fmt.Println("Browser closed.")
}
