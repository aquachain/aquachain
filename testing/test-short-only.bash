set -e
set -eo pipefail
#tmp1=$(go list ./... 2>/dev/null)
tmp1=$(go list ./... 2>/dev/null| egrep -v 'p2p|fetchers|downloader|peer|simulation')
set -x
CGO_ENABLED=${CGO_ENABLED-0} go test $@ $tmp1 # accepts any flag


