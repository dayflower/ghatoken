TARGET=ghatoken

target:	$(TARGET)

$(TARGET):	cmd/ghatoken/main.go cmd/ghatoken/version.go lib.go
	go build -ldflags "-X main.Version=$(shell git describe --tags) -X main.Revision=$(shell git rev-parse --short HEAD)" -buildvcs=true github.com/dayflower/ghatoken/v0/cmd/ghatoken

clean:
	rm -f $(TARGET)

