package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/AeroNotix/fileutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var BaseDirectory = flag.String("basedir", "", "The base directory which contains the git directories you want to cache.")
var CacheDirectory = flag.String("cache", "", "The directory which to cache git directories to.")

// IsGitDir will return a boolean value indicating whether or not the
// directory path is a git directory or not.
//
// It does not walk upwards in directories, just if the directory is
// passed is a git directory.
func IsGitDir(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	fis, err := f.Readdir(-1)
	if err != nil {
		return false
	}
	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}
		if strings.Contains(fi.Name(), ".git") {
			return true
		}
	}
	return false
}

func CheckFilePathAgainstRules(directory string, message string) bool {
	if directory == "" {
		log.Printf("Argument Error: %s", message)
		return false
	}
	return true
}

func BackupDirectory(fi FullFileInfo) error {
	return fileutil.CopyDirectory(
		filepath.Join(*CacheDirectory, fi.Info.Name(), ".git"),
		filepath.Join(fi.Path, ".git"),
	)
}

func GenerateURLRewrite(finfo FullFileInfo, cache_dir string) (string, error) {
	oldwd, _ := os.Getwd()
	os.Chdir(finfo.Path)
	git_branches, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		return "", err
	}
	os.Chdir(oldwd)
	git_branch_scanner := bufio.NewScanner(bytes.NewReader(git_branches))

	// we use two because a single cloned repo will have two
	// repositories in the git remote -v line. Usually this will
	// be enough.
	//
	// We use a map since that will easily filter out duplicates.
	branches := make(map[string]struct{}, 2)

	for git_branch_scanner.Scan() {
		split_branch_line := strings.Split(git_branch_scanner.Text(), " ")
		if len(split_branch_line) != 2 {
			fmt.Println(split_branch_line, len(split_branch_line))
			return "", fmt.Errorf(
				"Malformed output when scanning git branch output:\n%s",
				git_branch_scanner.Text(),
			)
		}
		branches[strings.Split(split_branch_line[0], "\t")[1]] = struct{}{}
	}

	// A configuration entry for a `git' rewrite should look like:
	//
	// [url "/path/to/repo/"]
	//     insteadOf = git@host:project.git
	output := bytes.NewBuffer([]byte{})
	cache_dir = filepath.Join(fileutil.MakeAbs(cache_dir), finfo.Info.Name())
	output.WriteString(
		fmt.Sprintf("[url \"%s/\"]\n\t", cache_dir),
	)
	for key, _ := range branches {
		output.WriteString(
			fmt.Sprintf("insteadOf = %s\n", key),
		)
	}
	return output.String(), nil
}

func init() {
	flag.Parse()
}

type FullFileInfo struct {
	Path string
	Info os.FileInfo
}

func main() {
	if !CheckFilePathAgainstRules(*BaseDirectory, "You must supply a search directory.") ||
		!CheckFilePathAgainstRules(*CacheDirectory, "You must supply a cache directory.") {
		return
	}
	git_directories := []FullFileInfo{}
	err := filepath.Walk(*BaseDirectory, func(path string, info os.FileInfo, err error) error {
		if IsGitDir(path) {
			git_directories = append(git_directories, FullFileInfo{path, info})
		}
		return nil
	})
	if err != nil {
		log.Println(err)
		return
	}
	config := make([]string, len(git_directories))
	for _, git_dir := range git_directories {
		err = BackupDirectory(git_dir)
		if err != nil {
			log.Println(err)
		}
		rewrite_rule, err := GenerateURLRewrite(git_dir, *CacheDirectory)
		if err != nil {
			log.Println(err)
		}
		config = append(config, rewrite_rule)
	}
	fmt.Println(strings.Join(config, "\n"))
}
