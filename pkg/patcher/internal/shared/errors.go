package shared

import "errors"

var (
	ErrGameNotInstalled = errors.New("game is not installed")
	ErrFileNotFound     = errors.New("file not found")
)
