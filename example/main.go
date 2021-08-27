package main

import (
	"fmt"
	"os"

	"github.com/notional-labs/get"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("cannot get pwd")
	}
	fmt.Println(dir)
	get.Get(dir+"/genesis.json", "Qmc54DreioPpPDUdJW6bBTYUKepmcPsscfqsfFcFmTaVig")
}
