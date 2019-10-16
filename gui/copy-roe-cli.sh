set -e

ls package.json
mkdir -p extraResources
rm -f extraResources/*
cd ../pkg
go build -o ../gui/extraResources/roe-cli cmd/roecli/*.go
# GOOS=windows GOARCH=386 go build -o ../gui/extraResources/roe-cli cmd/roecli/*.go
