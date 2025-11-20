.PHONY: gen install-gen gen-dragonfly test

# Install the bevi code generator
install-gen:
	go install ./cmd/gen

# Run code generation for the entire module
gen:
	go run ./cmd/gen -root .

# Run dragonfly event handler generation
gen-dragonfly:
	cd dragonfly && go run ./cmd/gen .

# Run tests
test:
	go test ./...
