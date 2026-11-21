package main

import (
	"fmt"

	gomod "github.com/IUCLabs/workflows/testdata/go-mod"
)

var version = "dev"

func main() {
	fmt.Println(gomod.Greet(""), version)
}
