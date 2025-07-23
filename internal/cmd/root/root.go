package root

import (
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/arisu-archive/bluearchive-data-sync/internal/cmd/sync"
)

type rootCommand struct {
	cmd     *cobra.Command
	exit    func(code int)
	verbose bool
}

func Execute(version string, exit func(code int), in io.Reader, out, err io.Writer, args []string) {
	newRootCommand(version, exit, in, out, err).Execute(args)
}

func (r *rootCommand) Execute(args []string) {
	r.cmd.SetArgs(args)
	if err := r.cmd.Execute(); err != nil {
		slog.ErrorContext(r.cmd.Context(), "ba-sync failed.", slog.Any("error", err))
		r.exit(1)
	}
}

func newRootCommand(version string, exit func(code int), in io.Reader, out, err io.Writer) *rootCommand {
	root := &rootCommand{
		exit: exit,
	}

	cmd := &cobra.Command{
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		Use:               "ba-sync <command> [flags]",
		Short:             "ba-sync is a tool for syncing asset data to the Android device",
		Example:           "ba-sync sync --serial <serial-number>",
		Version:           version,
		SilenceErrors:     false,
		SilenceUsage:      false,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if root.verbose {
				slog.SetDefault(slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug})))
				slog.Debug("verbose mode enabled")
			}
		},
		PersistentPostRunE: func(c *cobra.Command, _ []string) error {
			slog.InfoContext(c.Context(), "ba-sync finished")
			return nil
		},
	}

	cmd.PersistentFlags().BoolVarP(&root.verbose, "verbose", "v", false, "verbose mode")
	cmd.AddCommand(sync.NewCommand(in, out, err).Command())
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(err)

	root.cmd = cmd
	return root
}
