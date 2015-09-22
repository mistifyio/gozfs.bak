package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var zfs *os.File

func diff(want, got []byte) (string, string) {
	w := strings.Split(fmt.Sprintf("%03v", want), " ")
	w[0] = strings.TrimLeft(w[0], "[")
	w[len(w)-1] = strings.TrimRight(w[len(w)-1], "]")

	g := strings.Split(fmt.Sprintf("%03v", got), " ")
	g[0] = strings.TrimLeft(g[0], "[")
	g[len(g)-1] = strings.TrimRight(g[len(g)-1], "]")

	min := len(w)
	if len(g) < min {
		min = len(g)
	}
	red := "\x1b[31;1m"
	green := "\x1b[32;1m"
	reset := "\x1b[0m"
	diff := false
	for i := 0; i < min; i++ {
		if g[i] != w[i] {
			if diff == false {
				diff = true
				g[i] = red + g[i]
				w[i] = green + w[i]
			}
		} else if diff {
			diff = false
			g[i] = reset + g[i]
			w[i] = reset + w[i]
		}
	}
	if diff {
		g[len(g)-1] += reset
		w[len(w)-1] += reset
	}
	w[0] = "[" + w[0]
	w[len(w)-1] += "]"
	g[0] = "[" + g[0]
	g[len(g)-1] += "]"
	return strings.Join(w, " "), strings.Join(g, " ")
}
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
