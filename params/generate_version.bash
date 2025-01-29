#!/bin/bash
# a script to parse "version" from version.go file for release packaging
# prints to stdout something like '1.7.17' with no newline
set -e
inputfile=version.go
outputfile=../VERSION
# path exception for go generate -v ./... 
if [ ! -f $inputfile ] || [ ! -f $outputfile ]; then
    inputfile=params/$inputfile
    outputfile=VERSION
fi
if [ ! -f $inputfile ] || [ ! -f $outputfile ]; then
    echo "File not found: $inputfile or $outputfile" 1>&2
    echo "This script should be run from the 'params' directory" 1>&2
    exit 1
fi
# here, using '$(( ))' to convert output of 'awk' to bash number
VERSION_MAJOR=$(($(grep 'VersionMajor = ' $inputfile | awk '{print $3}')))
VERSION_MINOR=$(($(grep 'VersionMinor = ' $inputfile | awk '{print $3}')))
VERSION_PATCH=$(($(grep 'VersionPatch = ' $inputfile | awk '{print $3}')))
if [ -z "$VERSION_MAJOR" ] || [ -z "$VERSION_MINOR" ] || [ -z "$VERSION_PATCH" ]; then
    echo "Failed to get version from $inputfile" 1>&2
    exit 1
fi
if [ "$VERSION_MAJOR" != 1 ] || [ "$VERSION_MINOR" -lt 5 ] || [ "$VERSION_PATCH" -lt 0 ]; then
    echo "Invalid version $VERSION_MAJOR.$VERSION_MINOR.$VERSION_PATCH" 1>&2
    exit 1
fi
echo -n "$VERSION_MAJOR.$VERSION_MINOR.$VERSION_PATCH" >$outputfile