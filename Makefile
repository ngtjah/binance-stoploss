build-all: build-linux build-windows

build-linux:
	echo "Compiling for Linux"
	GOOS=linux GOARCH=386 go build -o bin/crypto-ejector .

build-windows:
	echo "Compiling for Windows"
	GOOS=windows GOARCH=386 go build -o bin/crypto-ejector.exe .
