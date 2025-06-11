package main

import (
	"os/user"
	"path/filepath"
)

func getCacheDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return filepath.Join(usr.HomeDir, ".cache"), nil
}
