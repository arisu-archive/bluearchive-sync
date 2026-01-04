package sync

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/arisu-archive/assets-dumper/pkg/resourceapi"

	"github.com/arisu-archive/bluearchive-data-sync/internal/cmd"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/adbx"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/patcher"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/xdelta"
)

type Command struct {
	xdeltaClient *xdelta.Client
	cmd          *cobra.Command
	opts         options
}

func NewCommand(stdin io.Reader, stdout, stderr io.Writer) *Command {
	sc := &Command{
		xdeltaClient: &xdelta.Client{
			Stdin:  stdin,
			Stdout: stdout,
			Stderr: stderr,
		},
	}
	c := &cobra.Command{
		Use:               "sync",
		Aliases:           []string{"s"},
		Short:             "Sync asset data to the Android device",
		Long:              "Sync asset data to the Android device",
		SilenceErrors:     true,
		SilenceUsage:      true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              cmd.RunE("sync", sc.run),
	}
	c.Flags().StringVarP(&sc.opts.serial, "serial", "s", "", "Serial number of the Android device")
	c.Flags().StringVarP(&sc.opts.adbHost, "host", "a", "", "Host of the ADB client")
	c.Flags().StringVarP(&sc.opts.cachePath, "cache-path", "c", cacheFolderPath(), "Path to the cache directory")
	c.Flags().StringVarP(&sc.opts.server, "server", "r", "global", "Server to use for the resource data")
	c.Flags().BoolVar(&sc.opts.preloadOnly, "preload", false, "Only sync preload data")
	c.Flags().BoolVar(&sc.opts.forced, "forced", false, "Force sync all data")
	c.Flags().IntVar(&sc.opts.concurrency, "concurrency", 16, "Concurrency level for the patcher")
	c.MarkFlagsMutuallyExclusive("serial", "host")
	c.MarkFlagsOneRequired("serial", "host")

	sc.cmd = c
	return sc
}

func (c *Command) Command() *cobra.Command {
	return c.cmd
}

func (c *Command) run(cmd *cobra.Command, _ []string) error {
	// Preflight: ensure xdelta3 is available early
	if _, err := c.xdeltaClient.Command(cmd.Context(), "config"); err != nil {
		if errors.Is(err, xdelta.ErrXdelta3NotFound) {
			return fmt.Errorf("xdelta3 not found: %w. Please install xdelta3 (or xdelta) and ensure it is on PATH", err)
		}
		return fmt.Errorf("failed to initialize xdelta3: %w", err)
	}

	adbClient, err := adbx.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create adb client: %w", err)
	}
	device, err := c.getAndroidDevice(adbClient)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}
	if err := os.MkdirAll(c.opts.cachePath, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	p, err := patcher.NewPatcher(patcher.Options{
		Server:       resourceapi.GetServer(c.opts.server),
		Forced:       c.opts.forced,
		PreloadOnly:  c.opts.preloadOnly,
		CachePath:    c.opts.cachePath,
		XdeltaClient: c.xdeltaClient,
		Device:       device,
		Concurrency:  c.opts.concurrency,
	})
	if err != nil {
		return fmt.Errorf("failed to create patcher: %w", err)
	}
	if err := p.Apply(cmd.Context()); err != nil {
		return fmt.Errorf("failed to apply patcher: %w", err)
	}
	return nil
}

func (c *Command) getAndroidDevice(adbClient *adbx.Client) (adbx.Device, error) {
	serial := c.opts.serial
	if c.opts.adbHost != "" {
		if err := adbClient.Connect(c.opts.adbHost); err != nil {
			return adbx.Device{}, fmt.Errorf("failed to connect to adb host: %w", err)
		}
		serial = c.opts.adbHost
	}
	device, err := adbClient.GetDevice(serial)
	if err != nil {
		return adbx.Device{}, fmt.Errorf("failed to get device: %w", err)
	}
	return device, nil
}

func cacheFolderPath() string {
	cache, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Errorf("failed to get cache folder path: %w", err))
	}
	return filepath.Join(cache, ".bluearchive-data-sync")
}
