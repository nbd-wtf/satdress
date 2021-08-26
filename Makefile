satdress: $(shell find . -name "*.go") index.html go.mod
	CC=$$(which musl-gcc) go build -ldflags='-s -w -linkmode external -extldflags "-static"' -o ./satdress

deploy: satdress
	ssh root@hulsmann 'systemctl stop bitmia tinytip payaddress paymentlink'
	scp satdress hulsmann:satdress/satdress
	ssh root@hulsmann 'systemctl start bitmia tinytip payaddress paymentlink'
