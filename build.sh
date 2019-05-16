export TAG=1.0 
go build -ldflags "-X main._VERSION_='$TAG'" 
gox