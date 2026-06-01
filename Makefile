all:
	go build -o bin/damonctl .
	go build -o bin/mtune ./mtune
	gcc -O2 -Wall -Wextra -o bin/hotmem scripts/hotmem.c
	go build -o bin/v2paddr scripts/v2paddr.go
	
clean:
	rm -f bin/damonctl bin/mtune bin/hotmem bin/v2paddr