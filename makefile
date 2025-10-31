run:
	go run *.go -config ./ddns-srv.conf -debug -provider-debug-level 1

# github.com/libdns/
build-plugin:
	go run ../ddns-plugin-builder/main.go -save-path=$$PWD/data/plugins github.com/libdns/transip@v1.0.1 github.com/libdns/mijnhost@v1.1.1