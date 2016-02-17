@echo off
go test -coverprofile=coverage.out %*|grep -v -e "^\.\.*$"|grep -v "^$"|grep -v "thus far"
