package global

import (
	"context"
	"io"
	"os"
	"path"

	"github.com/arisu-archive/assets-dumper/pkg/resourceapi"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/adb"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/xdelta"
)

// FileProcessor defines the interface for processing files
type FileProcessor interface {
	ProcessFile(ctx context.Context, file *FileInfo, opts *ProcessOptions) error
}

// CacheManager defines the interface for file caching
type CacheManager interface {
	Get(path string) (io.ReadCloser, error)
	Put(path string, data io.Reader) error
	Exists(path string) bool
}

// DeviceManager defines the interface for device operations
type DeviceManager interface {
	PullFile(remotePath string) (io.ReadCloser, error)
	PushFile(localReader io.Reader, remotePath string, perm int) error
	CreateDirectory(path string) error
	IsAppInstalled(packageName string) bool
	GetAppVersion(packageName string) (string, error)
	InstallApp(apkPaths []string) error
	DownloadAndInstallApp(appReader io.Reader) error
}

// DeviceFileHelper provides common device file operations
type DeviceFileHelper struct {
	device DeviceManager
}

func NewDeviceFileHelper(device DeviceManager) *DeviceFileHelper {
	return &DeviceFileHelper{device: device}
}

func (h *DeviceFileHelper) PushToDevicePath(reader io.Reader, devicePath string, perm os.FileMode) error {
	// Create directory if needed
	if err := h.device.CreateDirectory(path.Dir(devicePath)); err != nil {
		return err
	}

	// Push file
	return h.device.PushFile(reader, devicePath, int(perm))
}

func (h *DeviceFileHelper) PushToAndroidPath(reader io.Reader, relativePath string, perm os.FileMode) error {
	devicePath := path.Join(AndroidDataPath, relativePath)
	return h.PushToDevicePath(reader, devicePath, perm)
}

// VersionManager defines the interface for version management
type VersionManager interface {
	GetCurrentVersions() (map[string]int64, error)
	GetFileHashes() (map[string]string, error)
	UpdateVersions(versions map[string]int64) error
	UpdateFileHashes(hashes map[string]string) error
	UpdateToLatestVersion(ctx context.Context, key string) error
}

// Data structures
type FileInfo struct {
	Path         string
	Hash         string
	Size         int64
	Type         FileType
	PatchVersion int64
}

type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypePatch
	FileTypeNew
	FileTypeSkip
)

type ProcessOptions struct {
	CacheDir     string
	PatchVersion int64
	Concurrency  int
	FileMode     os.FileMode
}

type PatcherConfig struct {
	PreloadOnly  bool
	CachePath    string
	Device       adb.Device
	AssetClient  resourceapi.Client
	XdeltaClient *xdelta.Client
	Concurrency  int
}

type ProcessingStats struct {
	FilesProcessed int
	FilesSkipped   int
	BytesProcessed int64
	Errors         []error
}
