package swarmserv

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/majestrate/swarmserv/lib/cryptography"
	"github.com/majestrate/swarmserv/lib/encode"
	"github.com/majestrate/swarmserv/lib/network"
	"github.com/majestrate/swarmserv/lib/version"
)

const storageport = "8080"

// Main is the main entry point for swarmserv daemon
func Main() {
	// TODO: override me on runtime somehow
	dnshost := "127.3.2.1"
	dnsport := "53"
	seedfile := "identity.private"
	dbroot := "storage"
	idx := 0
	// parse args
	for idx < len(os.Args) {
		arg := os.Args[idx]
		if arg == "--lokinet-identity" {
			idx++
			if idx < len(os.Args) {
				seedfile = os.Args[idx]
			}
		} else if arg == "--db-location" {
			idx++
			if idx < len(os.Args) {
				dbroot = os.Args[idx]
			}
		}
		idx++
	}

	cryptoctx := new(cryptography.CryptoContext)

	err := cryptoctx.LoadPrivateKey(seedfile)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	netctx := network.CreateNetContext(dnshost, dnsport)

	snodeaddr, err := netctx.LookupOurAddress()

	if err != nil {
		fmt.Printf("failed to lookup snode address: %s\n", err.Error())
		return
	}
	pk, err := encode.ZBase32Encoding.DecodeString(strings.TrimSuffix(snodeaddr, ".snode"))
	if err != nil {
		fmt.Printf("invalid snode address detected: %s\n", snodeaddr)
		return
	}
	if !cryptoctx.EnsurePubKeyEqualTo(pk) {
		fmt.Println("snode pubkey missmatch")
		return
	}

	fmt.Println(version.Version)
	serv := NewServer(dbroot)
	err = serv.Init()
	if err != nil {
		fmt.Printf("error during server init: %s", err.Error())
		return
	}
	go func() {
		for {
			serv.Tick()
			time.Sleep(time.Second * 10)
		}
	}()
	server := &http.Server{
		Handler: serv,
		Addr:    net.JoinHostPort(snodeaddr, storageport),
	}
	fmt.Printf("swarmserv going up on %s\n", server.Addr)
	for {
		err = server.ListenAndServe()
		if err != nil {
			fmt.Printf("ListenAndServe: %s\n", err.Error())
			time.Sleep(time.Second)
		}
	}
}
