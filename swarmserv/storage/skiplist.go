package storage

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"github.com/majestrate/swarmserv/swarmserv/model"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var enc = base32.StdEncoding.WithPadding(base32.NoPadding)

type fsSkiplistStore struct {
	root           string
	expireDuration time.Duration
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

func (s *fsSkiplistStore) visitMessage(owner string, f *os.File, since time.Time, visit MessageVisitor) error {
	var msg model.Message
	st, err := f.Stat()
	if err != nil {
		return err
	}
	mod := st.ModTime()
	if mod.Before(since) {
		return nil
	}
	hash, err := enc.DecodeString(st.Name())
	if err != nil {
		return err
	}
	msg.Hash = hex.EncodeToString(hash)
	msg.ExpirationTimestamp = uint64(mod.Add(s.expireDuration).Unix())
	buff := make([]byte, st.Size())
	_, err = io.ReadFull(f, buff)
	if err != nil {
		return err
	}
	msg.Data = string(buff)
	return visit(msg)
}

func (s *fsSkiplistStore) Mktemp() string {
	var buf [5]byte
	rand.Read(buf[:])
	return filepath.Join(s.root, fmt.Sprintf("tmp-%d-%s", time.Now().UnixNano(), base32.StdEncoding.EncodeToString(buf[:])))
}

func (s *fsSkiplistStore) IterAllFor(owner string, visit MessageVisitor) error {
	return s.iterAllForSince(owner, visit, time.Unix(0, 0))
}

func (s *fsSkiplistStore) iterAllForSince(owner string, visit MessageVisitor, since time.Time) error {
	bucket, dir := s.getSkiplistFor(owner)
	err := s.ensureBucketDir(bucket, dir)
	if err != nil {
		return err
	}
	p := filepath.Join(s.root, bucket, dir)
	_, err = os.Stat(p)
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
				defer f.Close()
				err = s.visitMessage(owner, f, since, visit)
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

func (s *fsSkiplistStore) IterSinceHashFor(owner string, hash []byte, visit MessageVisitor) error {
	if hash == nil {
		return s.IterAllFor(owner, visit)
	}
	bucket, dir := s.getSkiplistFor(owner)
	err := s.ensureBucketDir(bucket, dir)
	if err != nil {
		return err
	}
	fname := filepath.Join(s.root, bucket, dir, enc.EncodeToString(hash))
	stat, err := os.Stat(fname)
	if err == nil {
		since := stat.ModTime()
		err = s.iterAllForSince(owner, visit, since)
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

// acquire fs lock, do a function and remove fs lock
func (s *fsSkiplistStore) withIndexLock(fn func() error) error {
	fpath := filepath.Join(s.root, "index.lock")
	for {
		_, e := os.Stat(fpath)
		if e == nil {
			// spinlock
			time.Sleep(time.Millisecond * 10)
			continue
		}
		if os.IsNotExist(e) {
			f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			fmt.Fprintf(f, "%d", os.Getpid())
			err = f.Sync()
			if err != nil {
				return err
			}
			err = fn()
			f.Close()
			os.Remove(fpath)
			return err
		} else {
			fmt.Printf("error: %s\n", e.Error())
		}
	}
}

func (s *fsSkiplistStore) waitUntilIndexLockFree() {
	lock := filepath.Join(s.root, "index.lock")
	for {
		_, err := os.Stat(lock)
		if os.IsNotExist(err) {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
}

func (s *fsSkiplistStore) appendIndexExpireEntry(fullpath string, expiresAt uint64) error {
	// check for index lock
	s.waitUntilIndexLockFree()
	index := filepath.Join(s.root, "index")
	f, err := os.OpenFile(index, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0700)
	if err != nil {
		fmt.Printf("failed to open index: %s\n", err.Error())
		return err
	}
	_, err = fmt.Fprintf(f, "%s %d\n", fullpath, expiresAt)
	f.Sync()
	f.Close()
	if err != nil {
		fmt.Printf("failed to append index entry: %s\n", err.Error())
		return err
	}
	return err
}

func (s *fsSkiplistStore) Expire() error {
	_, err := os.Stat(filepath.Join(s.root, "index"))
	if err == nil {
		return s.withIndexLock(func() error {
			newf, err := os.OpenFile(filepath.Join(s.root, "index.new"), os.O_CREATE|os.O_WRONLY, 0700)
			if err != nil {
				return err
			}
			f, err := os.Open(filepath.Join(s.root, "index"))
			if err != nil {
				newf.Close()
				os.Remove(filepath.Join(s.root, "index.new"))
				return err
			}
			now := uint64(time.Now().Unix())
			scan := bufio.NewScanner(f)
			for scan.Scan() {
				parts := strings.Split(scan.Text(), " ")
				if len(parts) == 2 {
					// sanity check
					if strings.HasPrefix(parts[0], s.root) && strings.Index(parts[0], "..") == -1 {
						t, err := strconv.ParseUint(parts[1], 10, 64)
						if err == nil && now >= t {
							// expire old file and discard index entry
							fmt.Printf("expire %s\n", parts[0])
							e := os.Remove(parts[0])
							if e != nil {
								fmt.Printf("error: %s\n", e.Error())
							}
						} else {
							// write new index contents
							fmt.Fprintf(newf, "%s %s\n", parts[0], parts[1])
						}
					}
				}
			}
			newf.Sync()
			newf.Close()
			f.Close()
			os.Remove(filepath.Join(s.root, "index"))
			return os.Rename(filepath.Join(s.root, "index.new"), filepath.Join(s.root, "index"))
		})
	} else if os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}

func (s *fsSkiplistStore) PutMessageFor(owner string, msg *model.Message, infname string) (bool, error) {
	bucket, dir := s.getSkiplistFor(owner)
	err := s.ensureDir(bucket)
	if err != nil {
		return false, err
	}
	err = s.ensureBucketDir(bucket, dir)
	if err != nil {
		return false, err
	}
	hash, _ := hex.DecodeString(msg.Hash)
	outfname := s.getFilenameFor(bucket, dir, hash)
	_, e := os.Stat(outfname)
	if os.IsNotExist(e) {
		err = s.appendIndexExpireEntry(outfname, msg.ExpirationTimestamp)
		if err != nil {
			return false, err
		}
		err := os.Rename(infname, outfname)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, e
}
