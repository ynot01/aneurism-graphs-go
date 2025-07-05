# aneurism-graphs-go

Backend for [ynot01/aneurism-graphs](https://github.com/ynot01/aneurism-graphs)

Fetches data every 5 minutes and constructs data.ts, then publishes to surge.sh

Expects aneurism-graphs to be as a subfolder `./aneurism-graphs/`

`go build .`

`go run .`

`setsid ./aneurism-graphs-go > ./tracker.log 2>&1 < /dev/null &`

`git submodule foreach --recursive git reset --hard`

`git submodule update --recursive --remote`