env GOOS=windows GOARCH=amd64 go build -o ./dist/cloudgun-windows.exe .
env GOOS=darwin GOARCH=amd64 go build -o ./dist/cloudgun-mac-x86 .
env GOOS=darwin GOARCH=arm64 go build -o ./dist/cloudgun-mac-arm .
aws s3 cp ./dist s3://cloudgunfiles --recursive