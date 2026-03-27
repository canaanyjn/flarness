package nativebridge

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
)

// LogcatBridge captures Android logcat output and feeds it into the Flarness log pipeline.
type LogcatBridge struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	deviceID string
	onLog    func(model.LogEntry)
	stopCh   chan struct{}
	stopped  bool
}

func NewLogcatBridge(deviceID string, onLog func(model.LogEntry)) *LogcatBridge {
	return &LogcatBridge{
		deviceID: deviceID,
		onLog:    onLog,
		stopCh:   make(chan struct{}),
	}
}

func (lb *LogcatBridge) Start() error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	adbPath, err := exec.LookPath("adb")
	if err != nil {
		return fmt.Errorf("adb not found in PATH: %w", err)
	}

	args := []string{}
	if lb.deviceID != "" {
		args = append(args, "-s", lb.deviceID)
	}

	clearArgs := append([]string{}, args...)
	clearArgs = append(clearArgs, "logcat", "-c")
	exec.Command(adbPath, clearArgs...).Run()

	streamArgs := append([]string{}, args...)
	streamArgs = append(streamArgs, "logcat", "-v", "threadtime",
		"flutter:V", "FlutterJNI:V", "dart:V", "DartVM:V",
		"AndroidRuntime:E", "ActivityManager:W",
		"*:S",
	)

	lb.cmd = exec.Command(adbPath, streamArgs...)
	lb.cmd.Env = os.Environ()

	stdout, err := lb.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("logcat stdout pipe: %w", err)
	}

	if err := lb.cmd.Start(); err != nil {
		return fmt.Errorf("logcat start: %w", err)
	}

	go lb.readLoop(bufio.NewScanner(stdout))
	go lb.waitForExit()
	return nil
}

func (lb *LogcatBridge) readLoop(scanner *bufio.Scanner) {
	for scanner.Scan() {
		select {
		case <-lb.stopCh:
			return
		default:
		}

		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "-----") {
			continue
		}

		entry := lb.parseLine(line)
		if lb.onLog != nil {
			lb.onLog(entry)
		}
	}
}

func (lb *LogcatBridge) parseLine(line string) model.LogEntry {
	entry := model.LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     model.LevelInfo,
		Source:    model.SourceEngine,
		Message:   line,
	}

	parts := strings.Fields(line)
	if len(parts) >= 6 {
		levelChar := parts[4]
		switch levelChar {
		case "V", "D":
			entry.Level = model.LevelDebug
		case "I":
			entry.Level = model.LevelInfo
		case "W":
			entry.Level = model.LevelWarning
		case "E", "F":
			entry.Level = model.LevelError
		}

		tag := strings.TrimSuffix(parts[5], ":")
		if len(parts) > 6 {
			msg := strings.Join(parts[6:], " ")
			msg = strings.TrimPrefix(msg, ": ")
			entry.Message = fmt.Sprintf("[%s] %s", tag, msg)
		} else {
			entry.Message = fmt.Sprintf("[%s]", tag)
		}

		tagLower := strings.ToLower(tag)
		switch {
		case tagLower == "flutter" || tagLower == "dart":
			entry.Source = model.SourceApp
		case tagLower == "flutterjni" || tagLower == "dartvm":
			entry.Source = model.SourceEngine
		case tagLower == "androidruntime":
			entry.Source = model.SourceFramework
			entry.Level = model.LevelError
		}
	}

	return entry
}

func (lb *LogcatBridge) waitForExit() {
	if lb.cmd != nil {
		lb.cmd.Wait()
	}
}

func (lb *LogcatBridge) Stop() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.stopped {
		return
	}
	lb.stopped = true
	close(lb.stopCh)

	if lb.cmd != nil && lb.cmd.Process != nil {
		lb.cmd.Process.Kill()
	}
}

func (lb *LogcatBridge) IsRunning() bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.stopped || lb.cmd == nil || lb.cmd.Process == nil {
		return false
	}

	select {
	case <-lb.stopCh:
		return false
	default:
		return true
	}
}
