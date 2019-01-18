package swarmserv

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"swarmserv/version"
	"time"
)

// Main is the main entry point for swarmserv daemon
func Main() {
	if len(os.Args) != 3 {
		fmt.Printf("usage: %s ip port\n", os.Args[0])
		return
	}
	fmt.Println(version.Version)
	serv := NewServer()
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
	for {
		addr := net.JoinHostPort(os.Args[1], os.Args[2])
		err = http.ListenAndServe(addr, serv)
		if err != nil {
			fmt.Printf("ListenAndServe: %s\n", err.Error())
			time.Sleep(time.Second)
		}
	}

}
