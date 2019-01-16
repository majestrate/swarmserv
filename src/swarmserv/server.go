package swarmserv

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"swarmserv/pow"
	"swarmserv/storage"
	"time"
)

type Server struct {
	store storage.Store
}

func NewServer() *Server {
	return &Server{
		store: storage.NewSkiplistStore("."),
	}
}

func (s *Server) Init() error {
	return s.store.Init()
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
	fmt.Println(msg)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(msg)))
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	io.WriteString(w, msg)
}

func (s *Server) handleStore(w http.ResponseWriter, r *http.Request) {
	var buf [5]byte
	rand.Read(buf[:])
	tmpfilename := fmt.Sprintf("tmp-%d-%s", time.Now().UnixNano(), base32.StdEncoding.EncodeToString(buf[:]))
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

}
