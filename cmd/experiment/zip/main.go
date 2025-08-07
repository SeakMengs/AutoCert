package main

import "github.com/SeakMengs/AutoCert/pkg/autocert"

func main() {
	dir := "cmd/example/zip/zip_test"
	zipFile := "cmd/example/zip/zip_test/output.zip"

	// Zip dir walk through the dir, if output zip is inside that dir, it would cause recursion
	err := autocert.ZipDir(dir, zipFile)
	if err != nil {
		panic(err)
	}

	println("Directory zipped successfully!")
}
