package main

import (
	"flag"
	"fmt"
	"github.com/AeroNotix/octocache"
	"log"
	"strings"
)

var BaseDirectory = flag.String("basedir", "", "The base directory which contains the git directories you want to cache.")
var CacheDirectory = flag.String("cache", "", "The directory which to cache git directories to.")

func init() {
	flag.Parse()
}

func CheckFilePathAgainstRules(directory string, message string) bool {
	if directory == "" {
		log.Printf("Argument Error: %s", message)
		return false
	}
	return true
}

func main() {
	if !CheckFilePathAgainstRules(*BaseDirectory, "You must supply a search directory.") ||
		!CheckFilePathAgainstRules(*CacheDirectory, "You must supply a cache directory.") {
		return
	}
	git_directories, err := octocache.CollectGitDirectories(*BaseDirectory)

	if err != nil {
		log.Println(err)
		return
	}
	config := octocache.CacheDirectories(git_directories, *CacheDirectory)
	fmt.Println(strings.Join(config, ""))
}
