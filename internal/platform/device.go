package platform

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var (
	iosUUIDRe       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	androidSerialRe = regexp.MustCompile(`^(?:[a-z0-9._-]{8,}|(?:\d{1,3}\.){3}\d{1,3}:\d{2,5})$`)
)

type DeviceType int

const (
	DeviceTypeWeb DeviceType = iota
	DeviceTypeMacOS
	DeviceTypeLinux
	DeviceTypeWindows
	DeviceTypeAndroid
	DeviceTypeIOS
	DeviceTypeUnknown
)

func (t DeviceType) String() string {
	switch t {
	case DeviceTypeWeb:
		return "web"
	case DeviceTypeMacOS:
		return "macos"
	case DeviceTypeLinux:
		return "linux"
	case DeviceTypeWindows:
		return "windows"
	case DeviceTypeAndroid:
		return "android"
	case DeviceTypeIOS:
		return "ios"
	default:
		return "unknown"
	}
}

func Classify(device string) DeviceType {
	d := strings.ToLower(strings.TrimSpace(device))
	switch {
	case d == "chrome" || d == "web-server" || d == "edge" || strings.Contains(d, "web"):
		return DeviceTypeWeb
	case d == "macos":
		return DeviceTypeMacOS
	case d == "linux":
		return DeviceTypeLinux
	case d == "windows":
		return DeviceTypeWindows
	case strings.HasPrefix(d, "emulator-") || d == "android" || strings.Contains(d, "pixel") || strings.Contains(d, "sdk gphone") || strings.Contains(d, "android"):
		return DeviceTypeAndroid
	case strings.Contains(d, "iphone") || strings.Contains(d, "ipad") || d == "ios":
		return DeviceTypeIOS
	case iosUUIDRe.MatchString(d):
		return DeviceTypeIOS
	case androidSerialRe.MatchString(d):
		return DeviceTypeAndroid
	default:
		return DeviceTypeUnknown
	}
}

func IsWeb(device string) bool     { return Classify(device) == DeviceTypeWeb }
func IsAndroid(device string) bool { return Classify(device) == DeviceTypeAndroid }
func IsIOS(device string) bool     { return Classify(device) == DeviceTypeIOS }
func IsMacOS(device string) bool   { return Classify(device) == DeviceTypeMacOS }
func IsNativeDesktop(device string) bool {
	t := Classify(device)
	return t == DeviceTypeMacOS || t == DeviceTypeLinux || t == DeviceTypeWindows
}
func IsMobile(device string) bool {
	t := Classify(device)
	return t == DeviceTypeAndroid || t == DeviceTypeIOS
}

type DetectedDevice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

func DetectDevices() []DetectedDevice {
	var devices []DetectedDevice

	if runtime.GOOS == "darwin" {
		devices = append(devices, DetectedDevice{ID: "macos", Name: "macOS (desktop)", Platform: "macos"})
	}

	if androids := detectAndroidDevices(); len(androids) > 0 {
		devices = append(devices, androids...)
	}

	if runtime.GOOS == "darwin" {
		if sims := detectIOSSimulators(); len(sims) > 0 {
			devices = append(devices, sims...)
		}
	}

	if runtime.GOOS == "linux" {
		devices = append(devices, DetectedDevice{ID: "linux", Name: "Linux (desktop)", Platform: "linux"})
	}

	if _, err := exec.LookPath("google-chrome"); err == nil {
		devices = append(devices, DetectedDevice{ID: "chrome", Name: "Chrome (web)", Platform: "web"})
	} else if _, err := exec.LookPath("chromium"); err == nil {
		devices = append(devices, DetectedDevice{ID: "chrome", Name: "Chromium (web)", Platform: "web"})
	} else if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"); err == nil {
			devices = append(devices, DetectedDevice{ID: "chrome", Name: "Chrome (web)", Platform: "web"})
		}
	}

	return devices
}

func PickDefaultDevice() string {
	devices := DetectDevices()
	if len(devices) == 0 {
		return "chrome"
	}

	priorityOrder := []string{"macos", "linux", "windows"}
	for _, p := range priorityOrder {
		for _, d := range devices {
			if d.Platform == p {
				return d.ID
			}
		}
	}
	for _, d := range devices {
		if d.Platform == "android" {
			return d.ID
		}
	}
	for _, d := range devices {
		if d.Platform == "ios" {
			return d.ID
		}
	}
	return devices[0].ID
}

func detectAndroidDevices() []DetectedDevice {
	adbPath, err := exec.LookPath("adb")
	if err != nil {
		return nil
	}

	output, err := exec.Command(adbPath, "devices", "-l").Output()
	if err != nil {
		return nil
	}

	var devices []DetectedDevice
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of") || strings.HasPrefix(line, "*") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "device" {
			id := parts[0]
			name := id
			for _, p := range parts[2:] {
				if strings.HasPrefix(p, "model:") {
					name = strings.TrimPrefix(p, "model:")
					break
				}
			}
			devices = append(devices, DetectedDevice{
				ID:       id,
				Name:     fmt.Sprintf("Android (%s)", name),
				Platform: "android",
			})
		}
	}
	return devices
}

func detectIOSSimulators() []DetectedDevice {
	xcrunPath, err := exec.LookPath("xcrun")
	if err != nil {
		return nil
	}

	output, err := exec.Command(xcrunPath, "simctl", "list", "devices", "--json").Output()
	if err != nil {
		return nil
	}

	var payload struct {
		Devices map[string][]struct {
			UDID        string `json:"udid"`
			Name        string `json:"name"`
			IsAvailable bool   `json:"isAvailable"`
			State       string `json:"state"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil
	}

	var devices []DetectedDevice
	for _, group := range payload.Devices {
		for _, dev := range group {
			if !dev.IsAvailable || !strings.EqualFold(dev.State, "Booted") {
				continue
			}
			devices = append(devices, DetectedDevice{
				ID:       dev.UDID,
				Name:     fmt.Sprintf("iOS Simulator (%s)", dev.Name),
				Platform: "ios",
			})
		}
	}
	return devices
}
