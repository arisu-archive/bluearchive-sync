package adbx

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/arisu-archive/bluearchive-data-sync/pkg/gadb"
)

type Device struct {
	gadb.Device
}

func (d *Device) IsInstalled(appName string) bool {
	output, err := d.Device.RunShellCommand("pm", "list", "packages", appName, "|", "cut", "-d", ":", "-f", "2")
	if err != nil {
		fmt.Println(err)
		return false
	}
	for line := range strings.SplitSeq(output, "\n") {
		if strings.TrimSpace(line) == appName {
			return true
		}
	}
	return false
}

func (d *Device) GetPackageVersion(appName string) (string, error) {
	output, err := d.Device.RunShellCommand("dumpsys", "package", appName, "|", "grep", "versionName")
	if err != nil {
		return "", fmt.Errorf("failed to get package version: %w", err)
	}

	output = strings.TrimSpace(output)
	if !strings.HasPrefix(output, "versionName=") {
		return "", fmt.Errorf("failed to get package version: %s", output)
	}
	return strings.TrimPrefix(output, "versionName="), nil
}

func (d *Device) InstallAPK(apkPath string) error {
	args := []string{"install", "-r", apkPath}
	_, err := d.Device.RunShellCommand("pm", args...)
	if err != nil {
		return fmt.Errorf("failed to install apk: %w", err)
	}
	return nil
}

func (d *Device) InstallMultipleAPK(apkPath ...string) error {
	totalSize := 0
	for _, apk := range apkPath {
		size, err := d.Device.RunShellCommand("stat", "-c", "%s", apk)
		if err != nil {
			return fmt.Errorf("failed to get apk size: %w", err)
		}
		sizeInt, err := strconv.Atoi(strings.TrimSpace(size))
		if err != nil {
			return fmt.Errorf("failed to parse apk size: %w", err)
		}
		totalSize += sizeInt
	}
	// Create a session.
	output, err := d.Device.RunShellCommand("pm", "install-create", "-S", strconv.Itoa(totalSize))
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Output: Success: created install session [391044253]
	output = strings.TrimSpace(output)
	if !strings.HasPrefix(output, "Success: created install session [") {
		return fmt.Errorf("failed to create session: %s", output)
	}

	// Example output: "Success: created install session [391044253]"
	start := strings.Index(output, "[")
	end := strings.Index(output, "]")
	if start == -1 || end == -1 || end <= start+1 {
		return fmt.Errorf("failed to parse session id: %s", output)
	}
	sessionID := output[start+1 : end]
	// Write the APK to the session.
	for i, apk := range apkPath {
		size, err := d.Device.RunShellCommand("stat", "-c", "%s", apk)
		if err != nil {
			return fmt.Errorf("failed to get apk size: %w", err)
		}
		size = strings.TrimSpace(size)
		_, err = d.Device.RunShellCommand("pm", "install-write", "-S", size, sessionID, strconv.Itoa(i), apk)
		if err != nil {
			return fmt.Errorf("failed to write apk to session: %w", err)
		}
	}

	// Commit the session.
	_, err = d.Device.RunShellCommand("pm", "install-commit", sessionID)
	if err != nil {
		return fmt.Errorf("failed to commit session: %w", err)
	}
	return nil
}
