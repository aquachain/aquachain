set -e
set -eo pipefail
tmp1=$(go list ./... 2>/dev/null| egrep -v 'p2p|fetchers|filters|downloader|peer|simulation|whisper')
CGO_ENABLED=${CGO_ENABLED-0} go test $tmp1


