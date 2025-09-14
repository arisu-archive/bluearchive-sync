package global

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"sync"

	"github.com/arisu-archive/assets-dumper/pkg/resourceapi"
	"github.com/arisu-archive/assets-dumper/pkg/resources"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/patcher/internal/shared"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/xdelta"
)

// tempFileManager handles creation and cleanup of temporary files
type tempFileManager struct {
	files []string
}

func newTempFileManager() *tempFileManager {
	return &tempFileManager{}
}

func (tm *tempFileManager) create(prefix string) (*os.File, error) {
	file, err := os.CreateTemp("", prefix)
	if err != nil {
		return nil, err
	}
	tm.files = append(tm.files, file.Name())
	return file, nil
}

func (tm *tempFileManager) cleanup() {
	for _, file := range tm.files {
		os.Remove(file)
	}
}

// BaseProcessor provides common functionality for all processors
type BaseProcessor struct {
	cache  CacheManager
	device DeviceManager
}

func NewBaseProcessor(cache CacheManager, device DeviceManager) *BaseProcessor {
	return &BaseProcessor{
		cache:  cache,
		device: device,
	}
}

func (b *BaseProcessor) ProcessWithCache(ctx context.Context, file *FileInfo, opts *ProcessOptions, processFn func() error) error {
	// Check cache first
	if b.cache.Exists(opts.PatchVersion, file.Path) {
		slog.Debug("using cached file", "path", file.Path)
		return b.pushCachedFile(opts.PatchVersion, file.Path, opts.FileMode)
	}

	// Process the file
	if err := processFn(); err != nil {
		return err
	}

	// Push to device
	return b.pushCachedFile(opts.PatchVersion, file.Path, opts.FileMode)
}

func (b *BaseProcessor) pushCachedFile(patchVersion string, filePath string, fileMode os.FileMode) error {
	reader, err := b.cache.Get(patchVersion, filePath)
	if err != nil {
		return fmt.Errorf("failed to get cached file: %w", err)
	}
	defer reader.Close()

	return b.device.PushFile(reader, path.Join(AndroidDataPath, "Resource", filePath), int(fileMode))
}

func (b *BaseProcessor) CacheFile(patchVersion string, filePath string, reader io.Reader) error {
	return b.cache.Put(patchVersion, filePath, reader)
}

type PatchFileProcessor struct {
	*BaseProcessor
	assetClient  resourceapi.Client
	xdeltaClient *xdelta.Client
}

func NewPatchFileProcessor(assetClient resourceapi.Client, xdeltaClient *xdelta.Client, cache CacheManager, device DeviceManager) *PatchFileProcessor {
	return &PatchFileProcessor{
		BaseProcessor: NewBaseProcessor(cache, device),
		assetClient:   assetClient,
		xdeltaClient:  xdeltaClient,
	}
}

func (p *PatchFileProcessor) ProcessFile(ctx context.Context, file *FileInfo, opts *ProcessOptions) error {
	return p.ProcessWithCache(ctx, file, opts, func() error {
		return p.applyPatch(ctx, file, opts)
	})
}

func (p *PatchFileProcessor) applyPatch(ctx context.Context, file *FileInfo, opts *ProcessOptions) error {
	tmpManager := newTempFileManager()
	defer tmpManager.cleanup()

	// Pull source file
	sourceReader, err := p.device.PullFile(path.Join(AndroidDataPath, "Resource", file.Path))
	if err != nil {
		return fmt.Errorf("failed to pull source file: %w", err)
	}
	defer sourceReader.Close()

	// Create and populate source temp file
	sourceFile, err := tmpManager.create("source-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer sourceFile.Close()

	if _, err := io.Copy(sourceFile, sourceReader); err != nil {
		return fmt.Errorf("failed to copy source file: %w", err)
	}
	sourceFile.Close()

	// Download patch
	patchReader, _, err := p.assetClient.WithPatchVersion(file.PatchVersion).DownloadPatch(ctx, file.Path+".xd3")
	if err != nil {
		return fmt.Errorf("failed to download patch: %w", err)
	}
	defer patchReader.Close()

	// Create and populate patch temp file
	patchFile, err := tmpManager.create("patch-*.xd3")
	if err != nil {
		return fmt.Errorf("failed to create patch temp file: %w", err)
	}
	defer patchFile.Close()

	if _, err := io.Copy(patchFile, patchReader); err != nil {
		return fmt.Errorf("failed to copy patch file: %w", err)
	}
	patchFile.Close()

	// Create output temp file
	outputFile, err := tmpManager.create("output-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create output temp file: %w", err)
	}
	outputFile.Close()

	if err := p.xdeltaClient.Decode(ctx, sourceFile.Name(), outputFile.Name(), patchFile.Name()); err != nil {
		return fmt.Errorf("failed to apply delta: %w", err)
	}

	// Cache the result
	outputReader, err := os.Open(outputFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer outputReader.Close()

	return p.CacheFile(opts.PatchVersion, file.Path, outputReader)
}

type FreshFileProcessor struct {
	*BaseProcessor
	assetClient resourceapi.Client
}

func NewFreshFileProcessor(assetClient resourceapi.Client, cache CacheManager, device DeviceManager) *FreshFileProcessor {
	return &FreshFileProcessor{
		BaseProcessor: NewBaseProcessor(cache, device),
		assetClient:   assetClient,
	}
}

func (n *FreshFileProcessor) ProcessFile(ctx context.Context, file *FileInfo, opts *ProcessOptions) error {
	return n.ProcessWithCache(ctx, file, opts, func() error {
		return n.downloadFile(ctx, file, opts)
	})
}

func (n *FreshFileProcessor) downloadFile(ctx context.Context, file *FileInfo, opts *ProcessOptions) error {
	// Download file
	reader, _, err := n.assetClient.DownloadResource(ctx, file.Path)
	if err != nil {
		return fmt.Errorf("failed to download resource: %w", err)
	}
	defer reader.Close()

	// Cache the file
	return n.CacheFile(opts.PatchVersion, file.Path, reader)
}

// ProcessFilesWithConcurrency processes files with controlled concurrency
func ProcessFilesWithConcurrency(ctx context.Context, files []*FileInfo, processor FileProcessor, opts *ProcessOptions) (*ProcessingStats, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}

	stats := &ProcessingStats{}
	semaphore := make(chan struct{}, opts.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, file := range files {
		wg.Add(1)
		go func(f *FileInfo) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			slog.Debug("processing file", "path", f.Path, "type", f.Type)

			err := processor.ProcessFile(ctx, f, opts)

			mu.Lock()
			if err != nil {
				// Check if it's a recoverable error
				if errors.Is(err, resources.ErrPatchVersionNotFound) {
					// Mark as new file and continue
					f.Type = FileTypeNew
					slog.Warn("patch not found, marking as new file", "path", f.Path)
				} else if errors.Is(err, shared.ErrFileNotFound) {
					f.Type = FileTypeNew
					slog.Warn("source missing, marking as new file", "path", f.Path)
				} else {
					stats.Errors = append(stats.Errors, fmt.Errorf("failed to process %s: %w", f.Path, err))
				}
			} else {
				stats.FilesProcessed++
				stats.BytesProcessed += f.Size
			}
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	return stats, nil
}
