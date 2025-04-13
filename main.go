package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
)

const tmpDir = "./gradeall-tmp"
const nScripts = 100

func main() {
	// Validate args
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: gradeall")
		os.Exit(1)
	}

	// Initialize the tmpDir
	if err := os.RemoveAll(tmpDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to clear %s\n", tmpDir)
		os.Exit(1)
	}
	if err := os.Mkdir(tmpDir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create %s\n", tmpDir)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	var nCorrect atomic.Uint32

	for i := 1; i <= nScripts; i++ {
		wg.Add(1)
		go runGradescript(i, &wg, &nCorrect)
	}

	wg.Wait()

	fmt.Printf("%d correct out of %d", nCorrect.Load(), nScripts)
}

func runGradescript(scriptNo int, wg *sync.WaitGroup, nCorrect *atomic.Uint32) {
	// Create temp dir to run gradescript inside of
	chrootDir := path.Join(tmpDir, fmt.Sprintf("gradescript-%02d", scriptNo))
	os.RemoveAll(chrootDir)
	os.Mkdir(chrootDir, 0o700)

	// Link `Gradescript-Examples`, `bin`, and all executables in the CWD to our temp dir
	linkfiles := []string{"Gradescript-Examples", "bin"}
	dirents, err := os.ReadDir(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read CWD")
		wg.Done()
		return
	}
	for _, dirent := range dirents {
		mode := dirent.Type()
		// Only consider executable files
		if !mode.IsRegular() || (mode&0o100) == 0 {
			continue
		}
		linkfiles = append(linkfiles, dirent.Name())
	}
	for _, file := range linkfiles {
		if err := os.Symlink(file, path.Join(chrootDir, file)); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to link file %s\n", file)
			wg.Done()
			return
		}
	}

	// Run the gradescript
	cmd := exec.Command("bash", "gradescript", strconv.Itoa(scriptNo))
	cmd.Dir = chrootDir
	err = cmd.Run()
	if err == nil {
		nCorrect.Add(1)
		fmt.Printf("Problem %d is correct\n", scriptNo)
	} else {
		fmt.Printf("Problem %d is incorrect.\n", scriptNo)
	}

	wg.Done()
}
