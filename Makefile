satdress: $(shell find . -name "*.go") $(shell find . -name "*.html") $(shell find . -name "*.css") go.mod
	CC=$$(which musl-gcc) go build -ldflags='-s -w -linkmode external -extldflags "-static"' -o ./satdress

deploy: satdress
	ssh root@turgot 'systemctl stop bitmia tinytip payaddress paymentlink'
	scp satdress turgot:satdress/satdress
	ssh root@turgot 'systemctl start bitmia tinytip payaddress paymentlink'
