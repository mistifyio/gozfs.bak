package main

import (
	//"bufio"
	"fmt"
	"gozfs/nv"
	"io/ioutil"
	"os"
	"strings"
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

func list(name string) error {
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	defer reader.Close()
	defer writer.Close()

	opts := nv.List{
		Pairs: []nv.Pair{
			{Value: int(writer.Fd())},
		},
	}
	opts.Pairs[0].EncodedSize = 0x1c
	opts.Pairs[0].DecodedSize = 0x20
	opts.Pairs[0].Name = "fd"
	opts.Pairs[0].NElements = 1
	opts.Pairs[0].Type = nv.INT32

	innvl := nv.List{}

	args := nv.List{
		Pairs: []nv.Pair{
			{Value: "zfs_list"},
			{Value: innvl},
			{Value: opts},
			{Value: uint64(0)},
		},
	}
	args.Pairs[0].EncodedSize = 0x24
	args.Pairs[0].DecodedSize = 0x28
	args.Pairs[0].Name = "cmd"
	args.Pairs[0].NElements = 1
	args.Pairs[0].Type = nv.STRING

	args.Pairs[1].EncodedSize = 0x2c
	args.Pairs[1].DecodedSize = 0x30
	args.Pairs[1].Name = "innvl"
	args.Pairs[1].NElements = 1
	args.Pairs[1].Type = nv.NVLIST

	args.Pairs[2].EncodedSize = 0x44
	args.Pairs[2].DecodedSize = 0x30
	args.Pairs[2].Name = "opts"
	args.Pairs[2].NElements = 1
	args.Pairs[2].Type = nv.NVLIST

	args.Pairs[3].EncodedSize = 0x24
	args.Pairs[3].DecodedSize = 0x20
	args.Pairs[3].Name = "version"
	args.Pairs[3].NElements = 1
	args.Pairs[3].Type = nv.UINT64

	bytes, err := nv.Encode(args)
	if err != nil {
		panic(err)
	}

	/*
		w := bufio.NewWriter(os.Stderr)
		w.Write(bytes)
		w.Flush()
		return nil
	*/

	fmt.Println("enter ioctl")
	err = ioctl(zfs, name, bytes, nil)
	fmt.Println("exit  ioctl")
	if err != nil {
		return err
	}

	fmt.Println("start reading")
	buf, err := ioutil.ReadAll(reader)
	fmt.Println("stop reading")
	if err != nil {
		return err
	}
	list, err := nv.Decode(buf)
	if err != nil {
		return err
	}
	printList("", list)
	return nil
}
