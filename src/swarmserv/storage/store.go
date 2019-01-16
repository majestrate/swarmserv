package storage

import (
	"io"
	"path/filepath"
)

type MessageVisitor func(io.Reader) error

type Store interface {
	// Init intializes the storage backend
	Init() error
	// IterAllFor iterates over all messages for owner
	IterAllFor(owner string, visit MessageVisitor) error
	// VisitByHashFor visits a message of owner by hash of message
	VisitByHashFor(owner string, hash []byte, visit MessageVisitor) error
	// PutMessageFor puts a message for owner
	PutMessageFor(owner string, hash []byte, infname string) (bool, error)
}

func NewSkiplistStore(rootdir string) Store {
	return &fsSkiplistStore{
		root: filepath.Join(rootdir, "storage"),
	}
}
