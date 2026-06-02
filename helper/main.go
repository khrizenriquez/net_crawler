package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const maxDurationMinutes = 120

var channelRE = regexp.MustCompile(`Channel:\s*([0-9]+)`)

type state struct {
	PID       int       `json:"pid"`
	Interface string    `json:"interface"`
	Channel   int       `json:"channel"`
	RawPath   string    `json:"rawPath"`
	StartedAt time.Time `json:"startedAt"`
	EndsAt    time.Time `json:"endsAt"`
}

func main() {
	if len(os.Args) < 2 {
		fatal("usage: duku-capture-helper probe|start|stop|status|restore")
	}
	switch os.Args[1] {
	case "probe":
		probe()
	case "start":
		start(os.Args[2:])
	case "stop":
		stop()
	case "status":
		status()
	case "restore":
		restore()
	default:
		fatal("unsupported action")
	}
}

func appDir() string {
	dir := os.Getenv("DUKU_DATA_DIR")
	if dir == "" || os.Geteuid() == 0 {
		home := ""
		if invokingUser := os.Getenv("SUDO_USER"); invokingUser != "" {
			if account, err := user.Lookup(invokingUser); err == nil {
				home = account.HomeDir
			}
		}
		if home == "" {
			if account, err := user.Current(); err == nil {
				home = account.HomeDir
			}
		}
		dir = filepath.Join(home, ".duku-net-lab")
	}
	return filepath.Clean(dir)
}
func statePath() string    { return filepath.Join(appDir(), "helper-state.json") }
func fatal(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
func must(err error) {
	if err != nil {
		fatal(err.Error())
	}
}
func isAllowedInterface(value string) bool { return value == "en0" }
func isAllowedChannel(value int) bool      { return value >= 1 && value <= 233 }
func inside(base, candidate string) bool {
	base, _ = filepath.Abs(base)
	candidate, _ = filepath.Abs(candidate)
	rel, err := filepath.Rel(base, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func validateRawOutputPath(staging, raw string) error {
	staging, _ = filepath.Abs(staging)
	raw, _ = filepath.Abs(raw)
	if filepath.Ext(raw) != ".pcap" || filepath.Dir(raw) != staging {
		return errors.New("output path must be a new .pcap directly inside the staging directory")
	}
	for _, path := range []string{filepath.Dir(staging), staging} {
		info, err := os.Lstat(path)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinked data path is not allowed: %s", path)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if _, err := os.Lstat(raw); err == nil {
		return errors.New("output capture path must not already exist")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
func readState() (state, error) {
	var out state
	data, err := os.ReadFile(statePath())
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(data, &out)
	return out, err
}
func saveState(value state) error {
	if err := os.MkdirAll(appDir(), 0o700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(value, "", "  ")
	return os.WriteFile(statePath(), data, 0o600)
}
func processRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	return err == nil && process.Signal(syscall.Signal(0)) == nil
}

func probe() {
	path, err := exec.LookPath("tcpdump")
	if err != nil {
		fatal("tcpdump is required")
	}
	channel, channelErr := currentChannel()
	response := map[string]any{"status": "degraded", "tcpdump": path, "interface": "en0", "currentChannel": channel, "channelControl": false, "maxDurationMinutes": maxDurationMinutes, "note": "macOS removed the airport channel-control utility. V1 capture is limited to the current channel; multi-channel rotation remains capability-blocked."}
	if channelErr != nil {
		response["channelProbeError"] = channelErr.Error()
	}
	_ = json.NewEncoder(os.Stdout).Encode(response)
}
func start(args []string) {
	if len(args) != 4 {
		fatal("usage: start <interface> <channel> <duration-minutes> <raw-output-path>")
	}
	iface := args[0]
	if !isAllowedInterface(iface) {
		fatal("only en0 is allowed in v1")
	}
	channel, err := strconv.Atoi(args[1])
	must(err)
	if !isAllowedChannel(channel) {
		fatal("channel outside supported range")
	}
	current, err := currentChannel()
	must(err)
	if current != channel {
		fatal(fmt.Sprintf("requested channel %d does not match current Wi-Fi channel %d; this macOS version cannot switch channels", channel, current))
	}
	duration, err := strconv.Atoi(args[2])
	must(err)
	if duration < 1 || duration > maxDurationMinutes {
		fatal("duration must be between 1 and 120 minutes")
	}
	raw := filepath.Clean(args[3])
	staging := filepath.Join(appDir(), "staging")
	must(validateRawOutputPath(staging, raw))
	if current, err := readState(); err == nil && processRunning(current.PID) {
		fatal("capture is already running")
	}
	must(os.MkdirAll(staging, 0o700))
	must(validateRawOutputPath(staging, raw))
	capture, err := os.OpenFile(raw, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	must(err)
	seconds := strconv.Itoa(duration * 60)
	cmd := exec.Command("/usr/sbin/tcpdump", "-I", "-i", iface, "-G", seconds, "-W", "1", "-w", "-")
	cmd.Stdout, cmd.Stderr = capture, os.Stderr
	if err := cmd.Start(); err != nil {
		_ = capture.Close()
		_ = os.Remove(raw)
		must(err)
	}
	_ = capture.Close()
	now := time.Now()
	must(saveState(state{PID: cmd.Process.Pid, Interface: iface, Channel: channel, RawPath: raw, StartedAt: now, EndsAt: now.Add(time.Duration(duration) * time.Minute)}))
	fmt.Printf("capture started pid=%d channel=%d\n", cmd.Process.Pid, channel)
}

func currentChannel() (int, error) {
	output, err := exec.Command("/usr/sbin/system_profiler", "SPAirPortDataType").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("read Wi-Fi channel: %w", err)
	}
	return parseChannel(string(output))
}
func parseChannel(output string) (int, error) {
	match := channelRE.FindStringSubmatch(output)
	if len(match) != 2 {
		return 0, errors.New("current Wi-Fi channel is unavailable")
	}
	return strconv.Atoi(match[1])
}
func stop() {
	current, err := readState()
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("capture is not running")
		restore()
		return
	}
	must(err)
	if processRunning(current.PID) {
		process, _ := os.FindProcess(current.PID)
		_ = process.Signal(syscall.SIGTERM)
	}
	_ = os.Remove(statePath())
	restore()
	fmt.Println("capture stopped")
}
func status() {
	current, err := readState()
	if err != nil || !processRunning(current.PID) {
		_ = os.Remove(statePath())
		fmt.Println(`{"status":"idle"}`)
		return
	}
	if time.Now().After(current.EndsAt) {
		stop()
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "running", "capture": current})
}
func restore() {
	cmd := exec.Command("/usr/sbin/networksetup", "-setairportpower", "en0", "on")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "restore warning: %v: %s\n", err, strings.TrimSpace(string(output)))
	}
}
