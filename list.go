package main

import (
	"fmt"
	"gozfs/nv"
	"io"
	"os"
	"strings"
	"time"
)

func decodeAndPrintList(buf []byte) {
	l, err := nv.Decode(buf)
	if err != nil {
		panic(err)
	}
	printList("", l)
}

func printList(indent string, l nv.List) {
	for _, p := range l.Pairs {
		if p.Type == nv.NVLIST {
			fmt.Fprintf(os.Stderr, "%sName: %s, Type: %s Size: %d\n", indent, p.Name, p.Type, p.EncodedSize)
			printList(strings.Repeat(" ", len(indent)+2), p.Value.(nv.List))
		} else {
			fmt.Fprintf(os.Stderr, "%sName: %s, Type: %s, Size: %d Value: %+v\n", indent, p.Name, p.Type, p.EncodedSize, p.Value)
		}
	}
}

func list(name string) error {
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	go io.Copy(os.Stdout, reader)
	defer reader.Close()
	defer writer.Close()

	var sargs = struct {
		Command string `nv:"cmd"`
		InNVL   struct {
		} `nv:"innvl"`
		Opts struct {
			Fd int32 `nv:"fd"`
		} `nv:"opts"`
		Version uint64 `nv:"version"`
	}{
		Command: "zfs_list",
		Opts: struct {
			Fd int32 `nv:"fd"`
		}{
			Fd: int32(writer.Fd()),
		},
	}
	fmt.Fprintln(os.Stderr, sargs)
	sbytes, err := nv.Encode(sargs)
	if err != nil {
		panic(err)
	}

	/*
		w := bufio.NewWriter(os.Stderr)
		w.Write(bytes)
		w.Flush()
		return nil
	*/

	/*
		if !reflect.DeepEqual(lbytes, sbytes) {
			want, got := diff(lbytes, sbytes)
			fmt.Fprintf(os.Stderr, "want:|%s|\n\n got:|%s|\n", want, got)
			fmt.Fprintln(os.Stderr, "want:")
			decodeAndPrintList(lbytes)
			fmt.Fprintln(os.Stderr, " got:")
			decodeAndPrintList(sbytes)
		}
	*/

	fmt.Fprintln(os.Stderr, "enter ioctl")
	err = ioctl(zfs, name, sbytes, nil)
	fmt.Fprintln(os.Stderr, "exit  ioctl")
	if err != nil {
		return err
	}

	/*
		fmt.Fprintln(os.Stderr, "start reading")
		buf, err := ioutil.ReadAll(reader)
		fmt.Fprintln(os.Stderr, "stop reading")
		if err != nil {
			return err
		}
		list, err := nv.Decode(buf)
		if err != nil {
			return err
		}
		printList("", list)
	*/
	time.Sleep(1 * time.Second)
	return nil
}
