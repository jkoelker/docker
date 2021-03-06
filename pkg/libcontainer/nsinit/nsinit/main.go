package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dotcloud/docker/pkg/libcontainer"
	"github.com/dotcloud/docker/pkg/libcontainer/nsinit"
)

var (
	dataPath  = os.Getenv("data_path")
	console   = os.Getenv("console")
	rawPipeFd = os.Getenv("pipe")
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("invalid number of arguments %d", len(os.Args))
	}

	container, err := loadContainer()
	if err != nil {
		log.Fatalf("unable to load container: %s", err)
	}

	switch os.Args[1] {
	case "exec": // this is executed outside of the namespace in the cwd
		var nspid, exitCode int
		if nspid, err = readPid(); err != nil && !os.IsNotExist(err) {
			log.Fatalf("unable to read pid: %s", err)
		}

		if nspid > 0 {
			exitCode, err = nsinit.ExecIn(container, nspid, os.Args[2:])
		} else {
			term := nsinit.NewTerminal(os.Stdin, os.Stdout, os.Stderr, container.Tty)
			exitCode, err = nsinit.Exec(container, term, "", dataPath, os.Args[2:], nsinit.DefaultCreateCommand, nil)
		}

		if err != nil {
			log.Fatalf("failed to exec: %s", err)
		}
		os.Exit(exitCode)
	case "init": // this is executed inside of the namespace to setup the container
		// by default our current dir is always our rootfs
		rootfs, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		pipeFd, err := strconv.Atoi(rawPipeFd)
		if err != nil {
			log.Fatal(err)
		}
		syncPipe, err := nsinit.NewSyncPipeFromFd(0, uintptr(pipeFd))
		if err != nil {
			log.Fatalf("unable to create sync pipe: %s", err)
		}

		if err := nsinit.Init(container, rootfs, console, syncPipe, os.Args[2:]); err != nil {
			log.Fatalf("unable to initialize for container: %s", err)
		}
	default:
		log.Fatalf("command not supported for nsinit %s", os.Args[0])
	}
}

func loadContainer() (*libcontainer.Container, error) {
	f, err := os.Open(filepath.Join(dataPath, "container.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var container *libcontainer.Container
	if err := json.NewDecoder(f).Decode(&container); err != nil {
		return nil, err
	}
	return container, nil
}

func readPid() (int, error) {
	data, err := ioutil.ReadFile(filepath.Join(dataPath, "pid"))
	if err != nil {
		return -1, err
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return -1, err
	}
	return pid, nil
}
