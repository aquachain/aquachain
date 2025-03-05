(
	set -o pipefail
	which go >/dev/null || {
		echo "go not found in PATH"
		exit 3
	}
	packagelist=${TESTPACKAGELIST-$(go list ./... 2>/dev/null | egrep -v 'p2p|fetchers|downloader|peer|simulation')}
	tmpfile=$(mktemp tmpaqua-short-tests.XXXXXX.tmp)
	echo testshorttmpfile=$tmpfile
	echo "running short tests for packages: $packagelist" 1>&2
	# trap 'rm -f $tmpfile' EXIT # TODO: uncomment this line to cleanup
	CGO_ENABLED=${CGO_ENABLED-0} go test $@ ${packagelist} -short | tee -a $tmpfile 1>&2
	exitcode=$?
	echo all done with exit code $exitcode
	if [ $exitcode == 0 ]; then
		echo status=OK
		exit 0
	fi
	fails=$(cat $tmpfile | egrep -- '.*FAIL.*\..*')
	echo "num_fails=$(echo "$fails" | wc -l)"
	echo "$fails" 1>&2
	echo 1>&2
	echo "----------------" 1>&2
	echo "status=FAIL"
	exit $exitcode
)
exit $?
