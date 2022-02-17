all: main.go
	go build -o faucet main.go

clean: faucet
	rm -rf faucet