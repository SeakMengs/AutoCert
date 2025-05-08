package main

import "github.com/SeakMengs/AutoCert/internal/util"

func main() {
	dir := "cmd/example/zip/zip_test"
	zipFile := "cmd/example/zip/zip_test/output.zip"

	// Zip dir walk through the dir, if output zip is inside that dir, it would cause recursion
	err := util.ZipDir(dir, zipFile)
	if err != nil {
		panic(err)
	}

	println("Directory zipped successfully!")
}
