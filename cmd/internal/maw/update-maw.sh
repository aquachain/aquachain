# Updates maw.go, embedding MAW.zip in current directory.
go-bindata -nocompress -tags='!nomaw' -pkg maw -o maw.go MAW.zip && \
gofmt -w -l -s maw.go
