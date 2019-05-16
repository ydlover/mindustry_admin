export TAG=1.0 
go build -ldflags "-X main._VERSION_='$TAG'" 

gox -osarch="windows/amd64"
gox -osarch="linux/386"
gox -osarch="linux/amd64"
gox -osarch="linux/arm"
gox -osarch="linux/arm64"
gox -osarch="windows/386"
