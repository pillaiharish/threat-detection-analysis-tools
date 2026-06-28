.PHONY: run docker docker-intranet docker-internet test vet fmt lint clean

run:
	cd browser-fingerprinting && go run ./server

docker:
	docker build -t browser-fingerprinting:latest browser-fingerprinting

docker-intranet:
	docker run --rm -p 8080:8080 -e VANTAGE=intranet browser-fingerprinting:latest

docker-internet:
	docker run --rm -p 8080:8080 -e VANTAGE=internet browser-fingerprinting:latest

test:
	cd browser-fingerprinting && go test ./...

vet:
	cd browser-fingerprinting && go vet ./...

fmt:
	cd browser-fingerprinting && gofmt -w server/

lint: vet
	@command -v golangci-lint >/dev/null && (cd browser-fingerprinting && golangci-lint run ./...) || echo "golangci-lint not installed; skipping"

clean:
	rm -f browser-fingerprinting/client_info.txt browser-fingerprinting/index.csv