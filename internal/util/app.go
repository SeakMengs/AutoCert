package util

import (
	"runtime"
)

func GetAppName() string {
	return "AutoCert"
}

func GetAppLogoURL(frontURL string) string {
	return frontURL + "/logo.png"
}

func DetermineWorkers(jobCount int) int {
	if jobCount <= 0 {
		return max(runtime.GOMAXPROCS(0), 1)
	}

	return min(max(runtime.GOMAXPROCS(0)*2, 1), jobCount)
}
