package dev

import (
	"os"
	"path"
	"strings"
	"time"
)

func FileSize(name string) int64 {
	fi, err := os.Stat(name)
	if err != nil {
		return 0
	}
	return fi.Size()
}

func FileMTime(name string) int64 {
	fi, err := os.Stat(name)
	if err != nil {
		return 0
	}
	mtime := fi.ModTime().UnixNano()
	return mtime
}

func IsFileWritten(path string) (bool, int64) {
	mtime1 := FileMTime(path)
	time.Sleep(1 * time.Second)
	mtime2 := FileMTime(path)
	return mtime1 != mtime2, mtime2
}

func UntilWritten(path string) {
	time.Sleep(30 * time.Millisecond)
	size := FileSize(path)
	if size > 1024*20 {
		for {
			if isWritten, _ := IsFileWritten(path); !isWritten {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func RelativePath(syncDir string, pathName string) string {
	return path.Join("./", strings.ReplaceAll(pathName, syncDir, ""))
}
