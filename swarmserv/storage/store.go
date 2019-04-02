package storage

import (
	"github.com/majestrate/swarmserv/swarmserv/model"
	"path/filepath"
	"time"
)

// MessageVisitor visits a message that was loaded
type MessageVisitor func(model.Message) error

type Store interface {
	// Init intializes the storage backend
	Init() error
	// IterAllFor iterates over all messages for owner
	IterAllFor(owner string, visit MessageVisitor) error
	// IterSinceHashFor iterates over all messages received after the message with hash
	// hash may be nil
	IterSinceHashFor(owner string, hash []byte, Visit MessageVisitor) error
	// PutMessageFor puts a message for owner
	PutMessageFor(owner string, msg *model.Message, bodyFilePath string) (bool, error)
	// Expire expires all old messages
	Expire() error
	// Mktemp generates a new temp file name
	Mktemp() string
}

func NewSkiplistStore(rootdir string) Store {
	return &fsSkiplistStore{
		root:           filepath.Join(rootdir, "storage"),
		expireDuration: time.Minute * 60,
	}
}
