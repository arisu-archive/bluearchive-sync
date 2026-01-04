package version

import (
	"bytes"
	"fmt"
	"io"
	"path"

	"github.com/spf13/cobra"

	"github.com/arisu-archive/bluearchive-data-sync/internal/cmd"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/adbx"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/javax"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/xdelta"
)

type Command struct {
	xdeltaClient *xdelta.Client
	cmd          *cobra.Command
	opts         options
}

func NewCommand(stdin io.Reader, stdout, stderr io.Writer) *Command {
	sc := &Command{}
	c := &cobra.Command{
		Use:               "version",
		Aliases:           []string{"s"},
		Short:             "Show version map information",
		Long:              "Show version map information",
		SilenceErrors:     true,
		SilenceUsage:      true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              cmd.RunE("version", sc.run),
	}
	c.Flags().StringVarP(&sc.opts.serial, "serial", "s", "", "Serial number of the Android device")
	c.Flags().StringVarP(&sc.opts.adbHost, "host", "a", "", "Host of the ADB client")
	c.MarkFlagsMutuallyExclusive("serial", "host")
	c.MarkFlagsOneRequired("serial", "host")

	sc.cmd = c
	return sc
}

func (c *Command) Command() *cobra.Command {
	return c.cmd
}

func (c *Command) run(cmd *cobra.Command, _ []string) error {
	adbClient, err := adbx.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create adb client: %w", err)
	}
	device, err := c.getAndroidDevice(adbClient)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}
	versions, err := getCurrentVersions(device)
	if err != nil {
		return fmt.Errorf("failed to get current versions: %w", err)
	}
	for key, version := range versions {
		fmt.Fprintf(cmd.OutOrStdout(), "%s: %d\n", key, version)
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

const (
	GameApplicationPackageName = "com.nexon.bluearchive"
	BaseAndroidPath            = "/sdcard/Android/data/com.nexon.bluearchive"
	AndroidDataPath            = BaseAndroidPath + "/files/PUB"
)

func getCurrentVersions(device adbx.Device) (map[string]int64, error) {
	buffer := bytes.NewBuffer(nil)
	versionPath := path.Join(AndroidDataPath, "Patch", "patch.version.map")
	if err := device.Pull(versionPath, buffer); err != nil {
		return nil, fmt.Errorf("failed to pull file %s: %w", versionPath, err)
	}

	decoder, err := javax.NewDecoder(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	versionMap, err := decoder.ReadObject()
	if err != nil {
		return nil, fmt.Errorf("failed to decode version map: %w", err)
	}

	hashmap, ok := versionMap.(*javax.HashMap)
	if !ok {
		return nil, fmt.Errorf("version map is not a HashMap")
	}

	versions := make(map[string]int64)
	for k, v := range hashmap.Data {
		versions[k.(string)] = int64(v.(int32))
	}

	return versions, nil
}
