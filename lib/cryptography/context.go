package cryptography

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"

	"golang.org/x/crypto/curve25519"
)

// CryptoContext is a context for all encryption and signing
type CryptoContext struct {
	privkey [32]byte
}

// ErrInvalidSeed indicates a error when the seed value for the CryptoContext is invalid
var ErrInvalidSeed = errors.New("invalid seed")

// EnsurePubKeyEqualTo returns true if the public key for our seed is equal to pk
func (cc *CryptoContext) EnsurePubKeyEqualTo(pk []byte) bool {
	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &cc.privkey)
	return subtle.ConstantTimeCompare(pk, pub[:]) == 1
}

// LoadPrivateKey loads the encryption seed from disk by filename
func (cc *CryptoContext) LoadPrivateKey(filename string) error {
	privkey, err := ioutil.ReadFile(filename)
	if err == nil && len(privkey) >= 35 {
		copy(cc.privkey[:], privkey[3:35])
	} else if err == nil {
		err = ErrInvalidSeed
	}
	return err
}

// prevents short writes
func writefull(w io.Writer, buf []byte) (err error) {
	n := 0
	written := 0
	for err == nil && written < len(buf) {
		n, err = w.Write(buf[written:])
		if err == nil {
			written += n
		}
	}
	return
}

// ErrInvalidPubKeyHex is the error when we get an invalid public key as hex
var ErrInvalidPubKeyHex = errors.New("invalid public key hex")

func (cc *CryptoContext) deriveSharedSecret(pk string) ([]byte, error) {
	b, err := hex.DecodeString(pk)
	if err != nil {
		return nil, err
	}
	if len(b) != 32 {
		return nil, ErrInvalidPubKeyHex
	}
	var pub [32]byte
	copy(pub[:], b[0:32])
	var shared [32]byte
	curve25519.ScalarMult(&shared, &cc.privkey, &pub)
	return shared[:], nil
}

func (cc *CryptoContext) processCipherBlocks(shared, iv []byte, r io.Reader, w io.Writer) error {
	// encrypt
	block, err := aes.NewCipher(shared)
	if err != nil {
		return err
	}
	// TODO: multiple cipher blocks
	enc := cipher.NewCBCEncrypter(block, iv)
	outbuf := make([]byte, enc.BlockSize())
	inbuf := make([]byte, enc.BlockSize())
	readbuf := inbuf[:]
	for err == nil {
		n, err := r.Read(readbuf)
		if err == io.EOF {
			// zero out remaining buffer
			for idx := range readbuf[n:] {
				readbuf[idx] = 0
			}
			n = enc.BlockSize()
			err = nil
		}
		if err == nil {
			if n < enc.BlockSize() {
				readbuf = readbuf[n:]
				continue
			} else {
				readbuf = inbuf[:]
			}
			enc.CryptBlocks(outbuf, inbuf)
			err = writefull(w, outbuf)
		}
	}
	return err
}

// DecryptFrom decrypts a message from a recipiant with public key
func (cc *CryptoContext) DecryptFrom(pk string, body io.Reader, dest io.Writer) error {
	shared, err := cc.deriveSharedSecret(pk)
	if err != nil {
		return err
	}
	iv := make([]byte, 16)
	_, err = io.ReadFull(body, iv)
	if err != nil {
		return err
	}
	return cc.processCipherBlocks(shared, iv, body, dest)
}

// EncryptTo encrypts an io.Reader to and io.Writer using a shared secret generated for PK using AES CBC and generates a random IV
func (cc *CryptoContext) EncryptTo(pk string, body io.Reader, dest io.Writer) error {
	shared, err := cc.deriveSharedSecret(pk)
	if err != nil {
		return err
	}

	iv := make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return err
	}
	// full write
	err = writefull(dest, iv)
	if err != nil {
		return err
	}
	return cc.processCipherBlocks(shared, iv, body, dest)
}
