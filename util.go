package main

import "github.com/sirupsen/logrus"

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
