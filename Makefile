# build
build:
	@go build -ldflags="-s -w" -o winget-prune.exe

# run
run: build
	@./winget-prune.exe
