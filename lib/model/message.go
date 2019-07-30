package model

type Message struct {
	Hash                string `json:"hash"`
	ExpirationTimestamp uint64 `json:"expiration"`
	Data                string `json:"data"`
}

type RetrieveRequest struct {
	PubKey   string `json:"pubKey"`
	LastHash string `json:"lastHash"`
}
