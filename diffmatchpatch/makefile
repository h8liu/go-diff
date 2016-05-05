all: check fmt
	go build .

test:
	go test .

fmt:
	gofmt -s -w -l .

check:
	e8chk -path="github.com/sergi/go-diff/diffmatchpatch" -width=1000 -height=3000

hello:
	echo "hello"
