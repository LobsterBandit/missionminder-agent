module github.com/lobsterbandit/missionminder-agent

go 1.18

require (
	github.com/radovskyb/watcher v1.0.7
	golang.org/x/exp v0.0.0-20220414153411-bcd21879b8fd
)

replace github.com/radovskyb/watcher => github.com/circleci/watcher v1.99.1
