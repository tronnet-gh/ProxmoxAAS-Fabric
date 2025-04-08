.PHONY: build test clean

build: clean
	CGO_ENABLED=0 go build -ldflags="-s -w" -v -o dist/ .

test: clean
	go run .

clean:
	go clean
	rm -f dist/*