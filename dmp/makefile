all: check fmt
	go install -v ./...
	gotags -R . > tags

test:
	go test .

fmt:
	gofmt -s -w -l .

check:
	e8chk -path="github.com/sergi/go-diff/dmp" -width=100 -height=1600

wc:
	wc -l `find . -name "*.go"`
