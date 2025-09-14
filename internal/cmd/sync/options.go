package sync

type options struct {
	serial      string
	adbHost     string
	cachePath   string
	server      string
	preloadOnly bool
	forced      bool
	concurrency int
}
