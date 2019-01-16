package pow

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"strconv"
)

//ErrBadPoW means the proof of work is not suffiecent
var ErrBadPoW = errors.New("bad PoW")

// CheckPOW checks if a proof of work is valid for a given nonce, timestamp, ttl, recipiant and data
// returns the hash of the message and nil on success
// returns nil and error on fail
func CheckPOW(nonce, timestamp, ttl, recipiant string, body io.Reader) ([]byte, error) {

	nonce_bytes, err := base64.URLEncoding.DecodeString(nonce)
	if err != nil {
		return nil, err
	}

	_, err = strconv.ParseUint(timestamp, 10, 64)
	if err != nil {
		return nil, err
	}

	ttl_int, err := strconv.ParseUint(ttl, 10, 64)
	if err != nil {
		return nil, err
	}

	payload := []byte{}

	payload_str := timestamp + ttl + recipiant
	payload = append(payload, []byte(payload_str)...)

	totalLen := BYTE_LEN + uint64(len(payload))

	h := sha512.New()
	h.Write(payload)
	n, err := io.Copy(h, body)
	if err != nil {
		return nil, err
	}
	totalLen += uint64(n)
	hashresult := h.Sum(nil)

	h.Reset()

	ttlMult := ttl_int * totalLen
	innerFract := ttlMult / uint64(65536)
	lenPlusInnerFract := totalLen + innerFract
	denom := NONCE_TRIALS * lenPlusInnerFract

	target := ^uint64(0) / denom

	inner := append(nonce_bytes, hashresult[:]...)

	hash := h.Sum(inner)

	if binary.BigEndian.Uint64(hash[:]) < target {
		return hashresult, nil
	}
	return nil, ErrBadPoW
}
