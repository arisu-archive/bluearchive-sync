package global

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/arisu-archive/assets-dumper/pkg/resourceapi"
	"github.com/arisu-archive/assets-dumper/pkg/resources"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/adbx"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/patcher/internal/shared"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/xdelta"
)

type Options struct {
	Forced       bool
	PreloadOnly  bool
	CachePath    string
	Device       adbx.Device
	XdeltaClient *xdelta.Client
	Concurrency  int
}

type Patcher struct {
	config         *PatcherConfig
	cache          CacheManager
	device         DeviceManager
	versionManager VersionManager
	patchProcessor FileProcessor
	newProcessor   FileProcessor
}

func NewPatcher(opts Options) *Patcher {
	client, _ := resources.NewClient(resourceapi.ServerGlobal)
	return NewPatcherWithAssetClient(opts, client)
}

func NewPatcherWithAssetClient(opts Options, assetClient resourceapi.Client) *Patcher {
	config := &PatcherConfig{
		Forced:       opts.Forced,
		PreloadOnly:  opts.PreloadOnly,
		CachePath:    opts.CachePath,
		Device:       opts.Device,
		AssetClient:  assetClient,
		XdeltaClient: opts.XdeltaClient,
		Concurrency:  opts.Concurrency,
	}

	if config.Concurrency <= 0 {
		config.Concurrency = 16
	}

	// Create components
	cache := NewFileCacheManager(config.CachePath)
	device := NewADBDeviceManager(config.Device)
	versionManager := NewDeviceVersionManager(device, config.AssetClient, config.PreloadOnly)
	patchProcessor := NewPatchFileProcessor(config.AssetClient, config.XdeltaClient, cache, device)
	newProcessor := NewFreshFileProcessor(config.AssetClient, cache, device)

	return &Patcher{
		config:         config,
		cache:          cache,
		device:         device,
		versionManager: versionManager,
		patchProcessor: patchProcessor,
		newProcessor:   newProcessor,
	}
}

func (p *Patcher) Apply(ctx context.Context) error {
	// Ensure game compatibility
	if err := p.ensureGameCompatibility(ctx); err != nil {
		return fmt.Errorf("game compatibility check failed: %w", err)
	}

	// Apply patches
	if err := p.applyPatches(ctx); err != nil {
		return fmt.Errorf("patch application failed: %w", err)
	}

	return nil
}

func (p *Patcher) ensureGameCompatibility(ctx context.Context) error {
	// Check if game is installed
	if !p.device.IsAppInstalled(GameApplicationPackageName) {
		return fmt.Errorf("%w: %s", shared.ErrGameNotInstalled, GameApplicationPackageName)
	}

	// Check versions
	currentVersion, err := p.device.GetAppVersion(GameApplicationPackageName)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	latestVersion, err := p.config.AssetClient.GetLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	slog.Debug("version check", "current", currentVersion, "latest", latestVersion)

	// Update if needed
	if currentVersion != latestVersion {
		slog.Debug("updating game application")
		if err := p.updateGameApplication(ctx); err != nil {
			return fmt.Errorf("failed to update game: %w", err)
		}
	}

	return nil
}

func (p *Patcher) updateGameApplication(ctx context.Context) error {
	appReader, _, err := p.config.AssetClient.DownloadApplication(ctx)
	if err != nil {
		return fmt.Errorf("failed to download application: %w", err)
	}
	defer appReader.Close()

	if err := p.device.DownloadAndInstallApp(appReader); err != nil {
		return fmt.Errorf("failed to install application: %w", err)
	}

	return nil
}

func (p *Patcher) applyPatches(ctx context.Context) error {
	// Get current state
	versions, err := p.versionManager.GetCurrentVersions()
	if err != nil {
		return fmt.Errorf("failed to get current versions: %w", err)
	}
	slog.Debug("current versions", "versions", versions)
	p.config.AssetClient = p.config.AssetClient.WithPatchVersion(versions["Preload"])

	fileHashes, err := p.versionManager.GetFileHashes()
	if err != nil {
		return fmt.Errorf("failed to get file hashes: %w", err)
	}

	// Identify files to process
	files, err := p.identifyFiles(ctx, versions["Preload"], fileHashes)
	if err != nil {
		return fmt.Errorf("failed to identify files: %w", err)
	}

	slog.Info("file analysis", "total", len(files), "patch", countByType(files, FileTypePatch), "new", countByType(files, FileTypeNew))
	if len(files) == 0 {
		slog.Warn("no files to process")
		return nil
	}

	// Process files
	latestPatchVersion, err := p.config.AssetClient.GetLatestPatchVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest patch version: %w", err)
	}
	if err := p.processFiles(ctx, files, latestPatchVersion); err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	// Update metadata
	if err := p.updateMetadata(ctx, fileHashes); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	slog.Info("patch process completed")
	return nil
}

func (p *Patcher) identifyFiles(ctx context.Context, patchVersion int64, fileHashes map[string]string) ([]*FileInfo, error) {
	filter := "**/**"
	if p.config.PreloadOnly {
		filter = "Preload/**"
	}
	resources, err := p.config.AssetClient.ListResources(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	var files []*FileInfo
	for _, resource := range resources {
		file := &FileInfo{
			Type:         FileTypeNew,
			Path:         resource.Path,
			Hash:         resource.Hash,
			Size:         resource.Size,
			PatchVersion: patchVersion,
		}

		if !p.config.Forced {
			// Determine file type
			if currentHash, exists := fileHashes[resource.Path]; exists && currentHash == resource.Hash {
				file.Type = FileTypeSkip
			} else if exists {
				file.Type = FileTypePatch
			} else {
				file.Type = FileTypeNew
			}
		}

		// Update hash cache
		fileHashes[resource.Path] = resource.Hash
		if file.Type != FileTypeSkip {
			files = append(files, file)
		}
	}

	return files, nil
}

func (p *Patcher) processFiles(ctx context.Context, files []*FileInfo, patchVersion string) error {
	opts := &ProcessOptions{
		PatchVersion: patchVersion,
		Concurrency:  p.config.Concurrency,
		FileMode:     0o664,
	}

	// Separate files by type
	patchFiles := filterByType(files, FileTypePatch)
	newFiles := filterByType(files, FileTypeNew)

	// Process patch files with concurrency
	if len(patchFiles) > 0 {
		slog.Info("processing patch files", "count", len(patchFiles))
		patchOpts := *opts
		patchOpts.FileMode = 0o664
		stats, err := ProcessFilesWithConcurrency(ctx, patchFiles, p.patchProcessor, &patchOpts)
		if err != nil {
			return fmt.Errorf("failed to process patch files: %w", err)
		}

		// Handle files that failed patching - convert to new files
		for _, file := range patchFiles {
			if file.Type == FileTypeNew {
				newFiles = append(newFiles, file)
			}
		}

		p.logStats("patch", stats)
	}

	// Process new files with concurrency
	if len(newFiles) > 0 {
		slog.Info("processing new files", "count", len(newFiles))
		newOpts := *opts
		newOpts.FileMode = 0o664
		stats, err := ProcessFilesWithConcurrency(ctx, newFiles, p.newProcessor, &newOpts)
		if err != nil {
			return fmt.Errorf("failed to process new files: %w", err)
		}

		p.logStats("new", stats)
	}

	return nil
}

func (p *Patcher) updateMetadata(ctx context.Context, fileHashes map[string]string) error {
	// Update to latest version
	if err := p.versionManager.UpdateToLatestVersion(ctx); err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}

	// Update file hashes
	if err := p.versionManager.UpdateFileHashes(fileHashes); err != nil {
		return fmt.Errorf("failed to update file hashes: %w", err)
	}

	return nil
}

func (p *Patcher) logStats(operation string, stats *ProcessingStats) {
	if len(stats.Errors) > 0 {
		slog.Error("processing completed with errors",
			"operation", operation,
			"processed", stats.FilesProcessed,
			"errors", len(stats.Errors),
			"bytes", stats.BytesProcessed)

		for _, err := range stats.Errors {
			slog.Error("processing error", "error", err)
		}
	} else {
		slog.Info("processing completed successfully",
			"operation", operation,
			"processed", stats.FilesProcessed,
			"bytes", stats.BytesProcessed)
	}
}

// Helper functions
func countByType(files []*FileInfo, fileType FileType) int {
	count := 0
	for _, file := range files {
		if file.Type == fileType {
			count++
		}
	}
	return count
}

func filterByType(files []*FileInfo, fileType FileType) []*FileInfo {
	var result []*FileInfo
	for _, file := range files {
		if file.Type == fileType {
			result = append(result, file)
		}
	}
	return result
}
