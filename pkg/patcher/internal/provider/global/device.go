package global

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/arisu-archive/bluearchive-data-sync/pkg/adb"
)

type ADBDeviceManager struct {
	device     adb.Device
	fileHelper *DeviceFileHelper
}

func NewADBDeviceManager(device adb.Device) *ADBDeviceManager {
	manager := &ADBDeviceManager{
		device: device,
	}
	manager.fileHelper = NewDeviceFileHelper(manager)
	return manager
}

func (d *ADBDeviceManager) PullFile(remotePath string) (io.ReadCloser, error) {
	buffer := bytes.NewBuffer(nil)
	if err := d.device.Pull(remotePath, buffer); err != nil {
		return nil, fmt.Errorf("failed to pull file %s: %w", remotePath, err)
	}
	return io.NopCloser(buffer), nil
}

func (d *ADBDeviceManager) PushFile(localReader io.Reader, remotePath string, perm int) error {
	if err := d.device.Push(localReader, remotePath, time.Now(), os.FileMode(perm)); err != nil {
		return fmt.Errorf("failed to push file to %s: %w", remotePath, err)
	}
	return nil
}

func (d *ADBDeviceManager) CreateDirectory(path string) error {
	if _, err := d.device.RunShellCommand("mkdir", "-p", path); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

func (d *ADBDeviceManager) IsAppInstalled(packageName string) bool {
	return d.device.IsInstalled(packageName)
}

func (d *ADBDeviceManager) GetAppVersion(packageName string) (string, error) {
	version, err := d.device.GetPackageVersion(packageName)
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %w", packageName, err)
	}
	return version, nil
}

func (d *ADBDeviceManager) InstallApp(apkPaths []string) error {
	if err := d.device.InstallMultipleAPK(apkPaths...); err != nil {
		return fmt.Errorf("failed to install APKs: %w", err)
	}
	return nil
}

func (d *ADBDeviceManager) DownloadAndInstallApp(reader io.Reader) error {
	tempPath := "/data/local/tmp/output.xapk"

	// Push app to device
	if err := d.fileHelper.PushToDevicePath(reader, tempPath, 0o644); err != nil {
		return fmt.Errorf("failed to push app: %w", err)
	}

	// Extract app
	if _, err := d.device.RunShellCommand("unzip", "-o", tempPath, "-d", "/data/local/tmp"); err != nil {
		return fmt.Errorf("failed to extract app: %w", err)
	}

	// Find APK files
	apkFiles, err := d.device.RunShellCommand("ls", "/data/local/tmp/*.apk")
	if err != nil {
		return fmt.Errorf("failed to list APK files: %w", err)
	}

	// Install APKs
	if err := d.InstallApp(strings.Fields(apkFiles)); err != nil {
		return fmt.Errorf("failed to install APKs: %w", err)
	}

	return nil
}
