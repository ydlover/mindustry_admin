#!/bin/bash

export TAG=1.0 
if [ -n "$1" ]; then
        TAG=$1
        echo ver:${TAG}
else
        echo "Please input build version!"
        echo "eg:build.sh 1.0"
        exit
fi
if [ "$2" != 'nb' ]; then
    gox -ldflags "-X main._VERSION_=${TAG}" -osarch="windows/amd64"
    gox -ldflags "-X main._VERSION_=${TAG}" -osarch="linux/386"
    gox -ldflags "-X main._VERSION_=${TAG}" -osarch="linux/amd64"
    gox -ldflags "-X main._VERSION_=${TAG}" -osarch="linux/arm"
    gox -ldflags "-X main._VERSION_=${TAG}" -osarch="linux/arm64"
    gox -ldflags "-X main._VERSION_=${TAG}" -osarch="windows/386"
fi
zip -r release_${TAG}.zip . -x "mindustry_admin" -x "*.zip" -x "./config/*" -x "./server-release.jar" -x "*.go" -x "./logs/*" -x ".git/*" -x ".gitignore" -x "./web/*"
