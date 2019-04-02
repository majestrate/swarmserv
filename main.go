package main

import (
	"fmt"
	"github.com/majestrate/swarmserv/lib/server"
	"github.com/majestrate/swarmserv/lib/version"
	"net"
	"net/http"
	"os"
	"time"
)

// Main is the main entry point for swarmserv daemon
func main() {
	if len(os.Args) != 3 {
		fmt.Printf("usage: %s ip port\n", os.Args[0])
		return
	}
	fmt.Println(version.Version)
	serv := server.NewServer("storage")
	err := serv.Init()
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

	s := &http.Server{
		Handler: serv,
		Addr:    net.JoinHostPort(os.Args[1], os.Args[2]),
	}
	for {
		err = s.ListenAndServe()
		if err != nil {
			fmt.Printf("ListenAndServe: %s\n", err.Error())
			time.Sleep(time.Second)
		}
	}
}
