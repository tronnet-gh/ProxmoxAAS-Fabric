.PHONY: build test clean

build: clean
	@echo "======================== Building Binary ======================="
	CGO_ENABLED=0 go build -ldflags="-s -w" -v -o dist/ .

test: clean
	go run .

clean:
	@echo "======================== Cleaning Project ======================"
	go clean
	rm -rf dist/*