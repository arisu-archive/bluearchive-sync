package global

import (
	"context"
	"io"
	"os"

	"github.com/arisu-archive/assets-dumper/pkg/resourceapi"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/adbx"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/xdelta"
)

const (
	GameApplicationPackageName = "com.nexon.bluearchive"
	BaseAndroidPath            = "/sdcard/Android/data/com.nexon.bluearchive"
	AndroidDataPath            = BaseAndroidPath + "/files/PUB"
)

// FileProcessor defines the interface for processing files
type FileProcessor interface {
	ProcessFile(ctx context.Context, file *FileInfo, opts *ProcessOptions) error
}

// CacheManager defines the interface for file caching
type CacheManager interface {
	Get(patchVersion string, path string) (io.ReadCloser, error)
	Put(patchVersion string, path string, data io.Reader) error
	Exists(patchVersion string, path string) bool
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

// VersionManager defines the interface for version management
type VersionManager interface {
	GetCurrentVersions() (map[string]int64, error)
	GetFileHashes() (map[string]string, error)
	UpdateVersions(versions map[string]int64) error
	UpdateFileHashes(hashes map[string]string) error
	UpdateToLatestVersion(ctx context.Context) error
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
	PatchVersion string
	Concurrency  int
	FileMode     os.FileMode
}

type PatcherConfig struct {
	Forced       bool
	PreloadOnly  bool
	CachePath    string
	Device       adbx.Device
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
