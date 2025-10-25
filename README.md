# aneurism-graphs-go

Backend for [ynot01/aneurism-graphs](https://github.com/ynot01/aneurism-graphs)

Fetches data every 5 minutes and constructs data.ts, then publishes to surge.sh

Expects aneurism-graphs to be as a subfolder `./aneurism-graphs/`

## Compilation & Usage

Build to ./aneurism-graphs-go(.exe): `go build .`

Build and run immediately, for testing: `go run .`

Update aneurism-graphs submodule: `git submodule foreach --recursive git reset --hard` `git submodule update --recursive --remote`

Run in the background (still runs when tty exits): `setsid ./aneurism-graphs-go > ./tracker.log 2>&1 < /dev/null &`

## Requirements

[node and npm](https://nodejs.org/en/download) in your PATH, visit link for install instructions

[surge CLI](https://surge.sh/) in your PATH, which should be accomplished by running `npm install --global surge`

For a fool-proof build demonstration, visit .github/workflows/build.yml; This workflow triggers on commit and outputs builds which can be accessed in the [Actions tab](https://github.com/ynot01/aneurism-graphs-go/actions/workflows/build.yml) or more simply [here](https://nightly.link/ynot01/aneurism-graphs-go/workflows/build/main)

Obviously, these builds point to a4tracker.surge.sh, so you will be unable to use them
