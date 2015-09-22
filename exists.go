package main

import (
	"fmt"
	"gozfs/nv"
	"os"
)

func exists(name string) error {
	l := nv.List{
		Pairs: []nv.Pair{
			{Value: "zfs_exists"},
			{Value: uint64(0)},
		},
	}
	l.Pairs[0].EncodedSize = 0x28
	l.Pairs[0].DecodedSize = 0x28
	l.Pairs[0].Name = "cmd"
	l.Pairs[0].NElements = 1
	l.Pairs[0].Type = nv.STRING

	l.Pairs[1].EncodedSize = 0x24
	l.Pairs[1].DecodedSize = 0x20
	l.Pairs[1].Name = "version"
	l.Pairs[0].NElements = 1
	l.Pairs[1].Type = nv.UINT64
	fmt.Println(l)

	lbytes, err := nv.Encode(l)
	if err != nil {
		panic(err)
	}
	f, err := os.Create("exists.list.in")
	if err != nil {
		panic(err)
	}
	fmt.Fprint(f, lbytes)
	f.Close()

	var exists = struct {
		Command string `nv:"cmd"`
		Version uint64 `nv:"version"`
	}{
		Command: "zfs_exists",
		Version: 0,
	}

	sbytes, err := nv.Encode(exists)
	if err != nil {
		panic(err)
	}
	f, err = os.Create("exists.struct.in")
	if err != nil {
		panic(err)
	}
	fmt.Fprint(f, sbytes)
	f.Close()

	fmt.Println("name:", name)
	return ioctl(zfs, name, sbytes, nil)
}
