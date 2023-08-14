package dev

const (
	DatastoreName  = ".dssnap"
	PrivateKeyName = ".pkey"
)

var (
	IgnoreNames = []string{".git", DatastoreName, PrivateKeyName}
)
