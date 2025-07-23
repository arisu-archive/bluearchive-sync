package patcher

import (
	"context"

	"github.com/arisu-archive/assets-dumper/pkg/resourceapi"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/adb"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/xdelta"
)

// Options contains configuration for patcher implementations.
type Options struct {
	Server       resourceapi.Server
	PreloadOnly  bool
	CachePath    string
	Device       adb.Device
	XdeltaClient *xdelta.Client
}

// Patcher defines the interface for syncing operations.
type Patcher interface {
	Apply(ctx context.Context) error
}
