package patcher

import (
	"fmt"

	"github.com/arisu-archive/assets-dumper/pkg/resourceapi"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/patcher/internal/provider/global"
)

var _ Patcher = (*global.Patcher)(nil)

func NewPatcher(opts Options) (Patcher, error) {
	switch opts.Server {
	case resourceapi.ServerGlobal:
		return global.NewPatcher(global.Options{
			PreloadOnly:  opts.PreloadOnly,
			CachePath:    opts.CachePath,
			Device:       opts.Device,
			XdeltaClient: opts.XdeltaClient,
		}), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidServer, opts.Server)
	}
}
