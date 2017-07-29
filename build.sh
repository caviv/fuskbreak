#!/bin/bash
env GOOS=linux GOARCH=386 go build
mv fuskbreak fuskbreak32
env GOOS=windows GOARCH=386 go build
mv fuskbreak.exe fuskbreak32.exe
env GOOS=netbsd GOARCH=amd64 go build
mv fuskbreak fuskbreakmac
env GOOS=windows GOARCH=amd64 go build
env GOOS=linux GOARCH=amd64 go build

