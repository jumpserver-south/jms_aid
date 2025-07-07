#!/bin/bash

version="0.3.0"

rm -rf jms_aid_*_amd64
rm -rf jms_aid_*_arm64

GOOS=linux GOARCH=amd64 go build -o jms_aid_${version}_amd64 jms_aid.go
GOOS=linux GOARCH=arm64 go build -o jms_aid_${version}_arm64 jms_aid.go