package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"nv"
)

func printList(indent string, l nv.List) {
	for _, p := range l.Pairs {
		if p.Type == nv.NVLIST {
			fmt.Printf("%sName: %s, Type: %s\n", indent, p.Name, p.Type)
			printList(strings.Repeat(" ", len(indent)+2), p.Value.(nv.List))
		} else {
			fmt.Printf("%sName: %s, Type: %s, Value:%+v\n", indent, p.Name, p.Type, p.Value)
		}
	}
}

func main() {
	skip := flag.Int("skip", 0, "number of leading bytes to skip")
	flag.Parse()

	if *skip > 0 {
		buf := make([]byte, *skip)
		i, err := io.ReadFull(os.Stdin, buf)
		if i != *skip {
			fmt.Println("failed to skip leading bytes")
			return
		}
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	buf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
		return
	}
	list, err := nv.Decode(buf)
	if err != nil {
		fmt.Println(err)
		return
	}
	printList("", list)
}
