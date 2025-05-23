package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
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

	// Check required files
	if info, err := os.Stat("gradescript"); err != nil || !info.Mode().IsRegular() {
		fmt.Fprintf(os.Stderr, "missing gradescript file\n")
		os.Exit(1)
	}
	if info, err := os.Stat("Gradescript-Examples"); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "missing Gradescript-Examples dir\n")
		os.Exit(1)
	}

	var wg sync.WaitGroup
	var nCorrect atomic.Uint32

	for i := 1; i <= nScripts; i++ {
		wg.Add(1)
		go runGradescript(i, &wg, &nCorrect)
	}

	wg.Wait()

	fmt.Printf("%d correct out of %d\n", nCorrect.Load(), nScripts)
}

func runGradescript(scriptNo int, wg *sync.WaitGroup, nCorrect *atomic.Uint32) {
	// Create temp dir to run gradescript inside of
	chrootDir := path.Join(tmpDir, fmt.Sprintf("gradescript-%02d", scriptNo))
	os.RemoveAll(chrootDir)
	os.Mkdir(chrootDir, 0o700)

	// Copy `Gradescript-Examples`, `bin`, and all executables in the CWD to our temp dir
	copyFiles := []string{"Gradescript-Examples", "bin", "gradescript"}
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
		copyFiles = append(copyFiles, dirent.Name())
	}
	for _, file := range copyFiles {
		info, _ := os.Stat(file)
		if info.IsDir() {
			//err = os.CopyFS(path.Join(chrootDir, file), os.DirFS(file))
			cmd := exec.Command("cp", "-r", file, path.Join(chrootDir, file))
			err = cmd.Run()
		} else {
			err = os.Link(file, path.Join(chrootDir, file))
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to copy file %s\n", file)
		}
	}

	// Run the gradescript
	cmd := exec.Command("bash", "gradescript", strconv.Itoa(scriptNo))
	cmd.Dir = chrootDir
	stdout, _ := cmd.StdoutPipe()
	err = cmd.Start()
	scanner := bufio.NewScanner(stdout)
	var correct bool
	if scanner.Scan() {
		head := strings.Fields(scanner.Text())
		correct = head[len(head)-1] == "correct."
	}
	for scanner.Scan() {
		// Flush the rest of stdout
	}
	cmd.Wait()

	if err == nil && correct {
		nCorrect.Add(1)
		fmt.Printf("Problem %d is correct.\n", scriptNo)
	} else {
		fmt.Printf("Problem %d is incorrect.\n", scriptNo)
	}

	wg.Done()
}
