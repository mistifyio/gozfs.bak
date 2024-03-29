package nv

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

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
	if len(g) > min {
		diff = true
		g[min] = red + g[min]
	}
	if len(w) > min {
		diff = true
		w[min] = green + w[min]
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

func TestEncodeGood(t *testing.T) {
	for _, s := range good {
		l, err := Decode(s.payload)
		if err != nil {
			t.Fatal(s.name, err)
		}
		b, err := Encode(l)
		if err != nil {
			t.Fatal(s.name, err)
		}
		if !reflect.DeepEqual(b, s.payload) {
			want, got := diff(s.payload, b)
			t.Fatalf("%s failed: encoded does not match payload\nwant:|%s|\n got:|%s|\n",
				s.name, want, got)
		}
	}
}

func TestEncodeBad(t *testing.T) {
	for _, s := range encode_bad {
		fmt.Println("=========")
		fmt.Println("testing:", s.err)
		p, err := Encode(s.payload)
		if err == nil {
			t.Fatalf("expected an error, wanted:|%s|, for payload: |%v|, buf:|%v|\n", s.err, s.payload, p)
		}
		if s.err != err.Error() {
			t.Fatalf("error mismatch, want:|%s|, got:|%s|, payload:|%v|\n",
				s.err, err.Error(), s.payload)
		}
	}
}
