package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/majestrate/swarmserv/lib/model"
	"github.com/majestrate/swarmserv/lib/pow"
	"github.com/majestrate/swarmserv/lib/storage"
	"io"
	"net/http"
	"os"
	"time"
)

type Server struct {
	store storage.Store
}

func NewServer(storedir string) *Server {
	return &Server{
		store: storage.NewSkiplistStore(storedir),
	}
}

func (s *Server) Init() error {
	return s.store.Init()
}

func (s *Server) Tick() {
	err := s.store.Expire()
	if err != nil {
		fmt.Printf("!!! [%s] error during expiration: %s\n", time.Now().String(), err.Error())
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.URL.Path {
	case "/store":
		s.handleStore(w, r)
	case "/retrieve":
		s.handleRetrieve(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (s *Server) plain(w http.ResponseWriter, code int, msg string) {
	fmt.Printf("%d %s\n", code, msg)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(msg)))
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	io.WriteString(w, msg)
}

func (s *Server) handleStore(w http.ResponseWriter, r *http.Request) {
	tmpfilename := s.store.Mktemp()
	defer r.Body.Close()
	nonce := r.Header.Get("X-Loki-pow-nonce")
	ttl := r.Header.Get("X-Loki-ttl")
	ts := r.Header.Get("X-Loki-timestamp")
	recip := r.Header.Get("X-Loki-recipient")
	f, err := os.OpenFile(tmpfilename, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		s.plain(w, http.StatusInternalServerError, err.Error())
		return
	}
	pr, pw := io.Pipe()
	mw := io.MultiWriter(pw, f)
	go func() {
		var buf [65536]byte
		io.CopyBuffer(mw, r.Body, buf[:])
		pw.Close()
		f.Close()
	}()
	h, err := pow.CheckPOW(nonce, ts, ttl, recip, pr)
	if err != nil {
		os.Remove(tmpfilename)
		s.plain(w, http.StatusForbidden, err.Error())
		return
	}
	ok, err := s.store.PutMessageFor(recip, h, tmpfilename)
	if ok {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
		fmt.Printf("[%s] stored message\n", time.Now().String())
	} else {
		os.Remove(tmpfilename)
		if err == nil {
			s.plain(w, http.StatusConflict, "duplicate hash")
		} else {
			s.plain(w, http.StatusInternalServerError, err.Error())
		}
	}
}

func (s *Server) handleRetrieve(w http.ResponseWriter, r *http.Request) {

	var msgs []model.Message

	lastHash, _ := hex.DecodeString(r.Header.Get("X-Loki-last-hash"))
	owner := r.Header.Get("X-Loki-recipient")
	lastExpire := uint64(0)

	err := s.store.IterSinceHashFor(owner, lastHash, func(m model.Message) error {
		msgs = append(msgs, m)
		if lastExpire < m.ExpirationTimestamp {
			lastExpire = m.ExpirationTimestamp
			lastHash, _ = hex.DecodeString(m.Hash)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("[%s] error retrieving messages: %s\n", time.Now().String(), err.Error())
		s.plain(w, http.StatusInternalServerError, err.Error())
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": msgs,
		"lastHash": hex.EncodeToString(lastHash),
	})
}
