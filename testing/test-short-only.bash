set -e
set -eo pipefail
tmp1=$(go list ./... 2>/dev/null| egrep -v 'p2p|fetchers|downloader|peer|simulation')
tmp1=$(go list ./... 2>/dev/null)
CGO_ENABLED=${CGO_ENABLED-0} go test $@ $tmp1 # accepts any flag
exitcode=$?
if [ $exitcode == 0 ]; then
	echo OK
	exit 0
fi
echo FAIL
exit $exitcode


