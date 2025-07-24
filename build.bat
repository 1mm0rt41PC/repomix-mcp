set GOOS=windows
set GOARCH=386
go build -o bin/repomix-mcp_win_x86.exe -ldflags "-w -s -extldflags=-static" -trimpath -gcflags "-l=4" ./cmd/repomix-mcp
set GOARCH=amd64
go build -o bin/repomix-mcp_win_x64.exe -ldflags "-w -s -extldflags=-static" -trimpath -gcflags "-l=4" ./cmd/repomix-mcp

set GOOS=linux
set GOARCH=386
go build -o bin/repomix-mcp_linux_x86.bin -ldflags "-w -s -extldflags=-static" -trimpath -gcflags "-l=4" ./cmd/repomix-mcp
set GOARCH=amd64
go build -o bin/repomix-mcp_linux_x64.bin -ldflags "-w -s -extldflags=-static" -trimpath -gcflags "-l=4" ./cmd/repomix-mcp