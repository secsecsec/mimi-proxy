all:
	go build -o mimi-proxy

run:
	sudo ./mimi-proxy -path="./config.json"

compile_run: all
	./tmp.sh
	sudo ./mimi-proxy -path="./config.json"
