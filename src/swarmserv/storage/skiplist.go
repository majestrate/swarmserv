package storage

import (
	"crypto/sha256"
	"encoding/base32"
	"os"
	"path/filepath"
)

var enc = base32.StdEncoding.WithPadding(base32.NoPadding)

type fsSkiplistStore struct {
	root string
}

func (s *fsSkiplistStore) Init() error {
	err := s.ensureDir("")
	if err != nil {
		return err
	}
	for _, r := range "QWERTYUIOPASDFGHJKLZXCVBNM234567" {
		str := string(r)
		err := s.ensureDir(str)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *fsSkiplistStore) ensureDir(dir string) error {
	if dir == "" {
		dir = s.root
	} else {
		dir = filepath.Join(s.root, dir)
	}
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0700)
	}
	return err
}

func (s *fsSkiplistStore) ensureBucketDir(bucket, dir string) error {

	f := filepath.Join(s.root, bucket, dir)

	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		return os.MkdirAll(f, 0700)
	}
	return err
}

func (s *fsSkiplistStore) IterAllFor(owner string, visit MessageVisitor) error {
	bucket, dir := s.getSkiplistFor(owner)
	p := filepath.Join(s.root, bucket, dir)
	_, err := os.Stat(p)
	if err == nil {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		names, err := f.Readdirnames(0)
		if err != nil {
			return err
		}
		for _, name := range names {
			f, err := os.Open(filepath.Join(p, name))
			if err == nil {
				err = visit(f)
				f.Close()
				if err != nil {
					return err
				}
			}
		}
	} else if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *fsSkiplistStore) VisitByHashFor(owner string, hash []byte, visit MessageVisitor) error {
	bucket, dir := s.getSkiplistFor(owner)
	fname := filepath.Join(s.root, bucket, dir, enc.EncodeToString(hash))
	f, err := os.Open(fname)
	if err == nil {
		err = visit(f)
		f.Close()
	}
	return err
}

func (s *fsSkiplistStore) getFilenameFor(bucket, dir string, hash []byte) string {
	str := enc.EncodeToString(hash)
	return filepath.Join(s.root, bucket, dir, str)
}

func (s *fsSkiplistStore) getSkiplistFor(owner string) (string, string) {
	h := sha256.Sum256([]byte(owner))
	str := enc.EncodeToString(h[:])
	return str[:1], str[1:]
}

func (s *fsSkiplistStore) PutMessageFor(owner string, hash []byte, infname string) (bool, error) {
	bucket, dir := s.getSkiplistFor(owner)
	err := s.ensureDir(bucket)
	if err != nil {
		return false, err
	}
	err = s.ensureBucketDir(bucket, dir)
	if err != nil {
		return false, err
	}
	outfname := s.getFilenameFor(bucket, dir, hash)
	_, e := os.Stat(outfname)
	if os.IsNotExist(e) {
		err := os.Rename(infname, outfname)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, e
}
