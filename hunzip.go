package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/gobs/httpclient"
	//"github.com/raff/hunzip/flate"
)

// func OpenHttpFile(url string, headers map[string]string) (*HttpFile, error)

func fatal(args ...interface{}) {
	fmt.Println(args...)
	os.Exit(1)
}

func main() {
	extract := flag.Bool("x", false, "extract files")
	flag.BoolVar(extract, "extract", *extract, "extract files")
	match := flag.String("match", "", "extract only matching files")
	skipdir := flag.Bool("no-dir", false, "don't create directories")
	debug := flag.Bool("debug", false, "enable debug logging")

	flag.Parse()

	var match_re *regexp.Regexp

	if *match != "" {
		match_re = regexp.MustCompile(*match)
	}

	for _, zipfile := range flag.Args() {
		fmt.Println("====", zipfile, "=================================")

		var reader io.ReaderAt
		var length int64

		if strings.Contains(zipfile, "://") { // URL
			f, err := httpclient.OpenHttpFile(zipfile, nil)
			if err != nil {
				fmt.Println(err)
				continue
			}

			f.Buffer = make([]byte, 128*1024)
			f.Debug = *debug

			defer f.Close()

			reader = f
			length = f.Size()
		} else {
			f, err := os.Open(zipfile)
			if err != nil {
				fmt.Println(err)
				continue
			}

			defer f.Close()

			fi, err := f.Stat()
			if err != nil {
				fmt.Println(zipfile, err)
				continue
			}

			reader = f
			length = fi.Size()
		}

		r, err := zip.NewReader(reader, length)
		if err != nil {
			fatal(err)
		}

		//r.RegisterDecompressor(zip.Deflate, func(r io.Reader) io.ReadCloser {
		//    return flate.NewReader(r)
		//})

		for _, f := range r.File {
			process := match_re == nil || match_re.MatchString(f.Name)

			if strings.HasSuffix(f.Name, "/") { // skip directories
				continue
			} else if !process {
				fmt.Println("skip", f.Name)
				continue
			} else {
				fmt.Println("extract", f.Name, f.UncompressedSize64)
			}

			if *extract {
				fname := f.Name
				if *skipdir {
					fname = path.Base(fname)
				} else {
					dir := path.Dir(fname)
					_ = os.Mkdir(dir, os.ModeDir|0755)
				}

				fin, err := f.Open()
				if err != nil {
					fatal(err)
				}

				fout, err := os.Create(fname)
				if err != nil {
					fatal(err)
				}

				n, err := io.Copy(fout, fin)
				if err != nil {
					fatal(fname, err)
				}

				if uint64(n) != f.UncompressedSize64 {
					fatal("wrong size", n)
				}

				fin.Close()
				fout.Close()
			}
		}
	}
}
