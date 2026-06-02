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
const localMaxDurationMinutes = 1
const captureSnapshotBytes = 4096
const localMaxCaptureBytes = 128 * 1024 * 1024
const supervisorAction = "__supervise"

var channelRE = regexp.MustCompile(`Channel:\s*([0-9]+)`)

type state struct {
	PID         int       `json:"pid"`
	Mode        string    `json:"mode,omitempty"`
	Interface   string    `json:"interface"`
	Channel     int       `json:"channel"`
	RawPath     string    `json:"rawPath"`
	PartialPath string    `json:"partialPath,omitempty"`
	StartedAt   time.Time `json:"startedAt"`
	EndsAt      time.Time `json:"endsAt"`
}

func main() {
	if len(os.Args) < 2 {
		fatal("usage: duku-capture-helper probe|start|start-local|stop|status|restore")
	}
	switch os.Args[1] {
	case "probe":
		probe()
	case "start":
		startRadio(os.Args[2:])
	case "start-local":
		startLocal(os.Args[2:])
	case "stop":
		stop()
	case "status":
		status()
	case "restore":
		restore()
	case supervisorAction:
		supervise(os.Args[2:])
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
func partialCapturePath(final string) string {
	return final + ".partial"
}
func validateCaptureOutputName(path string, local bool) error {
	name := filepath.Base(path)
	if local {
		if !strings.HasSuffix(name, ".local.pcap") {
			return errors.New("local-host output path must end with .local.pcap")
		}
		return nil
	}
	if !strings.HasSuffix(name, ".pcap") || strings.HasSuffix(name, ".local.pcap") {
		return errors.New("radio output path must end with .pcap and must not use the .local.pcap suffix")
	}
	return nil
}
func tcpdumpArgs(iface string, seconds int, monitor bool) []string {
	args := []string{}
	if monitor {
		args = append(args, "-I")
	}
	return append(args, "-i", iface, "-s", strconv.Itoa(captureSnapshotBytes), "-G", strconv.Itoa(seconds), "-W", "1", "-w", "-")
}
func validateCaptureDuration(duration int, monitor bool) error {
	max := maxDurationMinutes
	if !monitor {
		max = localMaxDurationMinutes
	}
	if duration < 1 || duration > max {
		return fmt.Errorf("duration must be between 1 and %d minutes", max)
	}
	return nil
}
func invokingUserIDs() (int, int, error) {
	uid, err := strconv.Atoi(os.Getenv("SUDO_UID"))
	if err != nil || uid < 0 {
		return 0, 0, errors.New("valid SUDO_UID is required")
	}
	gid, err := strconv.Atoi(os.Getenv("SUDO_GID"))
	if err != nil || gid < 0 {
		return 0, 0, errors.New("valid SUDO_GID is required")
	}
	return uid, gid, nil
}
func setPrivateCaptureOwner(path string, uid, gid int) error {
	if uid < 0 || gid < 0 {
		return errors.New("capture owner ids must be non-negative")
	}
	if err := os.Chown(path, uid, gid); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
func captureStartedMessage(pid, channel int) string {
	return fmt.Sprintf("capture started pid=%d channel=%d", pid, channel)
}
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
	for _, path := range []string{raw, partialCapturePath(raw)} {
		if _, err := os.Lstat(path); err == nil {
			return errors.New("output capture path must not already exist")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
func publishCapture(partial, final string) error {
	partial, _ = filepath.Abs(partial)
	final, _ = filepath.Abs(final)
	staging, _ := filepath.Abs(filepath.Join(appDir(), "staging"))
	if filepath.Dir(final) != staging || filepath.Ext(final) != ".pcap" || partial != partialCapturePath(final) {
		return errors.New("capture publication paths must be a .pcap and matching .pcap.partial inside staging")
	}
	for _, path := range []string{filepath.Dir(staging), staging} {
		info, err := os.Lstat(path)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinked data path is not allowed: %s", path)
		}
		if err != nil {
			return err
		}
	}
	if _, err := os.Lstat(final); err == nil {
		if _, partialErr := os.Lstat(partial); errors.Is(partialErr, os.ErrNotExist) {
			return nil
		}
		return errors.New("refusing to overwrite an existing finalized capture")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	info, err := os.Lstat(partial)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if _, finalErr := os.Lstat(final); finalErr == nil {
				return nil
			}
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("symlinked partial capture is not allowed")
	}
	if err := os.Rename(partial, final); err != nil {
		if _, finalErr := os.Lstat(final); finalErr == nil {
			if _, partialErr := os.Lstat(partial); errors.Is(partialErr, os.ErrNotExist) {
				return nil
			}
		}
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
func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for processRunning(pid) && time.Now().Before(deadline) {
		time.Sleep(250 * time.Millisecond)
	}
	return !processRunning(pid)
}
func stopCaptureProcess(pid int) bool {
	process, _ := os.FindProcess(pid)
	_ = process.Signal(syscall.SIGTERM)
	return waitForProcessExit(pid, 5*time.Second)
}
func captureByteLimit(final string) int64 {
	if strings.HasSuffix(filepath.Base(final), ".local.pcap") {
		return localMaxCaptureBytes
	}
	return 0
}
func waitForCaptureExit(pid int, partial, final string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	limit := captureByteLimit(final)
	for processRunning(pid) && time.Now().Before(deadline) {
		if limit > 0 {
			if info, err := os.Lstat(partial); err == nil && info.Size() >= limit {
				return stopCaptureProcess(pid)
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return !processRunning(pid)
}
func finishCapture(current state) error {
	if current.PartialPath != "" {
		if err := publishCapture(current.PartialPath, current.RawPath); err != nil {
			return err
		}
	}
	saved, err := readState()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if saved.PID != current.PID {
		return nil
	}
	if err := os.Remove(statePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
func launchSupervisor(pid int, partial, final string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(executable, supervisorAction, strconv.Itoa(pid), partial, final)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
func supervise(args []string) {
	if len(args) != 3 || os.Geteuid() != 0 {
		fatal("internal supervisor is unavailable")
	}
	pid, err := strconv.Atoi(args[0])
	must(err)
	if pid < 1 {
		fatal("invalid capture pid")
	}
	partial, final := filepath.Clean(args[1]), filepath.Clean(args[2])
	if partial != partialCapturePath(final) {
		fatal("invalid capture publication paths")
	}
	if !waitForCaptureExit(pid, partial, final, time.Duration(maxDurationMinutes+1)*time.Minute) {
		if !stopCaptureProcess(pid) {
			fatal("capture watchdog could not stop tcpdump")
		}
	}
	must(finishCapture(state{PID: pid, RawPath: final, PartialPath: partial}))
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
func startRadio(args []string) {
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
	raw := filepath.Clean(args[3])
	must(validateCaptureOutputName(raw, false))
	startCapture(iface, channel, duration, raw, true)
}
func startLocal(args []string) {
	if len(args) != 3 {
		fatal("usage: start-local <interface> <duration-minutes> <raw-output-path>")
	}
	iface := args[0]
	if !isAllowedInterface(iface) {
		fatal("only en0 is allowed in v1")
	}
	duration, err := strconv.Atoi(args[1])
	must(err)
	raw := filepath.Clean(args[2])
	must(validateCaptureOutputName(raw, true))
	channel, _ := currentChannel()
	startCapture(iface, channel, duration, raw, false)
}
func startCapture(iface string, channel, duration int, raw string, monitor bool) {
	must(validateCaptureDuration(duration, monitor))
	staging := filepath.Join(appDir(), "staging")
	must(validateRawOutputPath(staging, raw))
	if current, err := readState(); err == nil && processRunning(current.PID) {
		fatal("capture is already running")
	}
	must(os.MkdirAll(staging, 0o700))
	must(validateRawOutputPath(staging, raw))
	partial := partialCapturePath(raw)
	uid, gid, err := invokingUserIDs()
	must(err)
	capture, err := os.OpenFile(partial, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	must(err)
	if err := setPrivateCaptureOwner(partial, uid, gid); err != nil {
		_ = capture.Close()
		_ = os.Remove(partial)
		must(err)
	}
	cmd := exec.Command("/usr/sbin/tcpdump", tcpdumpArgs(iface, duration*60, monitor)...)
	cmd.Stdout, cmd.Stderr = capture, os.Stderr
	if err := cmd.Start(); err != nil {
		_ = capture.Close()
		_ = os.Remove(partial)
		must(err)
	}
	_ = capture.Close()
	tcpdumpPID := cmd.Process.Pid
	now := time.Now()
	mode := "local-host"
	if monitor {
		mode = "authorized-radio"
	}
	currentState := state{PID: tcpdumpPID, Mode: mode, Interface: iface, Channel: channel, RawPath: raw, PartialPath: partial, StartedAt: now, EndsAt: now.Add(time.Duration(duration) * time.Minute)}
	if err := saveState(currentState); err != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = os.Remove(partial)
		must(err)
	}
	if err := launchSupervisor(tcpdumpPID, partial, raw); err != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = os.Remove(statePath())
		_ = os.Remove(partial)
		must(err)
	}
	_ = cmd.Process.Release()
	fmt.Println(captureStartedMessage(tcpdumpPID, channel))
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
	if !waitForProcessExit(current.PID, 5*time.Second) {
		fatal("capture did not stop within five seconds")
	}
	must(finishCapture(current))
	restore()
	fmt.Println("capture stopped")
}
func status() {
	current, err := readState()
	if err != nil {
		_ = os.Remove(statePath())
		fmt.Println(`{"status":"idle"}`)
		return
	}
	if !processRunning(current.PID) {
		must(finishCapture(current))
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
