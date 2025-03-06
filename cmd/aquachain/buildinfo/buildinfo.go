package buildinfo

type BuildInfo struct {
	GitTag    string
	GitCommit string
	BuildDate string
	BuildTags string
}

var binfo BuildInfo

func SetBuildInfo(info BuildInfo) {
	binfo = info
}

func GetBuildInfo() BuildInfo {
	if binfo == (BuildInfo{}) {
		return BuildInfo{
			GitTag:    "v0.0.0-unknown",
			GitCommit: "unknown",
			BuildDate: "unknown",
			BuildTags: "unknown",
		}
	}
	return binfo
}
