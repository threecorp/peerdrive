package sync

import (
	"log"

	"github.com/radovskyb/watcher"
	"github.com/threecorp/peerdrive/pkg/dev"
)

// so tiny: need more handling
func logFatal(err error) {
	if err == nil {
		return
	}

	log.Printf("logFatal: %+v\n", err) // log.Fatalln(err)
}

// remove domeday
func paths(syncDir string, ev watcher.Event) (string, string) {
	return dev.RelativePath(syncDir, ev.Path), dev.RelativePath(syncDir, ev.OldPath)
}
