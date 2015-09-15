package main

import (
	"flag"
	"fmt"
	"os"
)

var zfs *os.File

func init() {
	z, err := os.OpenFile("/dev/zfs", os.O_RDWR, 0)
	if err != nil {
		panic(err)
	}
	zfs = z
}

var funcs = map[string]func(string) error{
	"exists": exists,
	"list":   list,
}

func main() {
	name := flag.String("name", "", "dataset name")
	cmd := flag.String("cmd", "", "command to invoke")
	flag.Parse()

	if *cmd == "" {
		fmt.Println("error: cmd is required")
		return
	}
	fn, ok := funcs[*cmd]
	if !ok {
		fmt.Println("error: uknown command", *cmd)
		return
	}

	if err := fn(*name); err != nil {
		fmt.Println("error:", err)
	}
}
