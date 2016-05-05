all: check fmt
	go install -v ./...
	gotags -R . > tags

test:
	go test .

fmt:
	gofmt -s -w -l .

check:
	e8chk -path="github.com/sergi/go-diff/diffmatchpatch" -width=100 -height=1600

hello:
	echo "hello"
