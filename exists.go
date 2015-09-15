package main

import "gozfs/nv"

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

	bytes, err := nv.Encode(l)
	if err != nil {
		panic(err)
	}

	return ioctl(zfs, name, bytes, nil)
}
