package main

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

type localExecutor struct {
	wd string
}

func (l *localExecutor) exec(args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, errors.New("Empty args")
	}
	name := args[0]
	arg := []string{}
	if len(args) > 1 {
		arg = args[1:]
	}
	cmd := exec.Command(name, arg...)
	cmd.Dir = l.wd
	return cmd.CombinedOutput()
}

func (l *localExecutor) commit() (string, error) {
	commitBytes, err := l.exec([]string{"git", "log", "-1", "--format=format:%H"})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(commitBytes)), nil
}

func (l *localExecutor) addAndCommit() error {
	commands := [][]string{
		[]string{"git", "add", "."},
		[]string{"git", "commit", "-am", "Commit"},
	}
	for _, command := range commands {
		_, err := l.exec(command)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *localExecutor) initWorkspace() error {
	body := `image: foobar:1.0.0`
	if err := ioutil.WriteFile(path.Join(l.wd, "test_file"), []byte(body), os.ModePerm); err != nil {
		return err
	}
	if _, err := l.exec([]string{"git", "init"}); err != nil {
		return err
	}
	return l.addAndCommit()
}
