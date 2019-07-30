package swarmserv

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/majestrate/swarmserv/lib/network"
	"github.com/majestrate/swarmserv/lib/version"
)

const storageport = "8080"

// Main is the main entry point for swarmserv daemon
func Main() {
	// TODO: override me on runtime somehow
	dnshost := "127.3.2.1"
	dnsport := "53"

	netctx := network.CreateNetContext(dnshost, dnsport)

	lokiaddr, err := netctx.LookupOurAddress()

	if err != nil {
		fmt.Printf("failed to lookup loki address: %s\n", err.Error())
	}

	fmt.Println(version.Version)
	serv := NewServer("storage")
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
		Addr:    net.JoinHostPort(lokiaddr, storageport),
	}
	for {
		err = server.ListenAndServe()
		if err != nil {
			fmt.Printf("ListenAndServe: %s\n", err.Error())
			time.Sleep(time.Second)
		}
	}
}
