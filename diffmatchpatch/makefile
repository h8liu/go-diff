all: check fmt
	go install -v ./...
	gotags -R . > tags

test:
	go test .

fmt:
	gofmt -s -w -l .

check:
	e8chk -path="github.com/sergi/go-diff/diffmatchpatch" -width=1000 -height=3000

hello:
	echo "hello"
