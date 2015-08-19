cflags=-I/usr/include/libzfs -I/usr/include/libspl
libs=-lzfs -lzfs_core -lnvpair

all: nv

nv: nv.c

%:%.c
	gcc -Og -ggdb3 -std=gnu11 -W -Wall -Wextra -o $@ $^ ${cflags} ${libs}

nv.txt: nv

.PHONY: known_good_data_test.go
known_good_data_test.go: ../known_good_data_test.go

../known_good_data_test.go: nv
	./$^ > $@

clean:
	rm -rf nv known_good_data_test.go
