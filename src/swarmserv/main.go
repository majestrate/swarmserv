package swarmserv

import (
	"fmt"
	"swarmserv/version"
)

// Main is the main entry point for swarmserv daemon
func Main() {
	fmt.Println(version.Version)
}
