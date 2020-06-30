package main

import (
	"github.com/abiosoft/ishell"
)

func Xp(shell *ishell.Shell) (f func(c *ishell.Context)) {
	return func(c *ishell.Context) {
		errString, ok := syntaxCheck(c, 1)
		if !ok {
			shell.Println(errString)
			return
		}
		if err := copyFromEntry(shell, c.Args[0], "password"); err != nil {
			shell.Printf("could not copy password: %s", err)
			return
		}
	}
}
