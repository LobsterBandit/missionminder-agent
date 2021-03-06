module github.com/lobsterbandit/missionminder-agent

go 1.18

require (
	github.com/fatih/color v1.13.0
	github.com/radovskyb/watcher v1.0.7
	golang.org/x/exp v0.0.0-20220414153411-bcd21879b8fd
)

require (
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	golang.org/x/sys v0.0.0-20211019181941-9d821ace8654 // indirect
)

replace github.com/radovskyb/watcher => github.com/lobsterbandit/watcher v1.99.2-0.20220503231916-aab8e3698fb1
