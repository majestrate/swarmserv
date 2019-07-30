package model

type Message struct {
	Hash                string `json:"hash"`
	ExpirationTimestamp uint64 `json:"expiration"`
	Data                string `json:"data"`
}
