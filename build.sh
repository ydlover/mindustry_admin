export TAG=1.4 

gox -osarch="windows/amd64"
gox -osarch="linux/386"
gox -osarch="linux/amd64"
gox -osarch="linux/arm"
gox -osarch="linux/arm64"
gox -osarch="windows/386"
zip -r release_${TAG}.zip . -x "./config/*" -x "./server-release.jar" -x "*.go" -x "./logs/*" -x ".git/*" -x ".gitignore"
