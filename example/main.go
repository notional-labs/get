package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/notional-labs/get"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("cannot get pwd")
	}
	joiner := []string{dir, "genesis.json"}
	path := strings.Join(joiner, "")
	get.Get(path, "Qmc54DreioPpPDUdJW6bBTYUKepmcPsscfqsfFcFmTaVig")
}
