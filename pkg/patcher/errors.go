package patcher

import (
	"errors"

	"github.com/arisu-archive/bluearchive-data-sync/pkg/patcher/internal/shared"
)

var (
	ErrInvalidServer    = errors.New("invalid server")
	ErrGameNotInstalled = shared.ErrGameNotInstalled
)
