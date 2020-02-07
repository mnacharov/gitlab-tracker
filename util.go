package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"text/template"

	"github.com/sirupsen/logrus"
)

func ConfigureLogging(logLevel string) error {
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
	return nil
}

func ProcessCommand(rule *Rule, args []string) (*exec.Cmd, error) {
	var argsExec []string
	for _, templ := range args {
		arg, err := gotmpl(templ, rule)
		if err != nil {
			return nil, err
		}
		argsExec = append(argsExec, os.ExpandEnv(arg))
	}
	if argsExec == nil {
		return nil, errors.New("empty command")
	}
	if len(argsExec) > 1 {
		c := exec.Command(argsExec[0], argsExec[1:]...)
		c.Env = os.Environ()
		return c, nil
	}
	c := exec.Command(argsExec[0])
	c.Env = os.Environ()
	return c, nil
}

func gotmpl(templ string, data interface{}) (string, error) {
	var templateEng *template.Template
	buf := bytes.NewBufferString("")
	templateEng = template.New("hook")
	if messageTempl, err := templateEng.Parse(templ); err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	} else if err := messageTempl.Execute(buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return buf.String(), nil
}
