package octocache

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/AeroNotix/fileutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FullFileInfo struct {
	Path string
	Info os.FileInfo
}

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

func CollectGitDirectories(basedir string) ([]FullFileInfo, error) {
	git_directories := []FullFileInfo{}
	return git_directories, filepath.Walk(basedir, func(path string, info os.FileInfo, err error) error {
		if IsGitDir(path) {
			git_directories = append(git_directories, FullFileInfo{path, info})
		}
		return nil
	})
}

func BackupDirectory(fi FullFileInfo, cache_dir string) error {
	return fileutil.CopyDirectory(
		filepath.Join(cache_dir, fi.Info.Name(), ".git"),
		filepath.Join(fi.Path, ".git"),
	)
}

func GenerateURLRewrite(finfo FullFileInfo, cache_dir string) (string, error) {
	oldwd, _ := os.Getwd()
	os.Chdir(finfo.Path)
	git_branches, err := exec.Command("git", "remote", "-v").Output()
	os.Chdir(oldwd)
	if err != nil {
		return "", err
	}
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
	if len(branches) == 0 {
		return "", nil
	}
	output := bytes.NewBuffer([]byte{})
	cache_dir = filepath.Join(fileutil.MakeAbs(cache_dir), finfo.Info.Name())
	output.WriteString(
		fmt.Sprintf("[url \"%s/\"]\n\t", cache_dir),
	)
	for key, _ := range branches {
		output.WriteString(
			fmt.Sprintf("insteadOf = %s\n\n", key),
		)
	}
	return output.String(), nil
}

func CacheDirectories(git_directories []FullFileInfo, cache_dir string) []string {
	config := make([]string, len(git_directories))
	for _, git_dir := range git_directories {
		err := BackupDirectory(git_dir, cache_dir)
		if err != nil {
			log.Println(err)
		}

		rewrite_rule, err := GenerateURLRewrite(git_dir, cache_dir)
		if err != nil {
			log.Println(err)
			continue
		}
		if rewrite_rule != "" {
			config = append(config, rewrite_rule)
		}
	}
	return config
}
