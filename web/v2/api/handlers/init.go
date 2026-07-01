package handlers

import (
	"github.com/sirupsen/logrus"
	"github.com/sundowndev/phoneinfoga/v2/lib/filter"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
	"sync"
)

var once sync.Once
var RemoteLibrary *remote.Library

// Store is the lookup-persistence backend, injected by the serve command via InitStore.
// It is nil when persistence is not configured (e.g. the CLI, or tests that never call
// InitStore); persistence-aware handlers must nil-check it and degrade gracefully.
var Store store.Store

func Init(filterEngine filter.Filter) {
	once.Do(func() {
		RemoteLibrary = remote.NewLibrary(filterEngine)
		remote.InitScanners(RemoteLibrary)
		logrus.Debug("Scanners and plugins initialized")
	})
}

// InitStore injects the lookup persistence store. Unlike Init it is a plain setter (no
// sync.Once) so it can be re-injected in tests; the serve command calls it exactly once.
func InitStore(s store.Store) {
	Store = s
}
