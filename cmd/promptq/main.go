package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ast-lw/promptq/internal/promptq"
)

func main() {
	if err := promptq.Run(os.Args[1:]); err != nil {
		if !errors.Is(err, promptq.ErrUsage) {
			fmt.Fprintln(os.Stderr, promptq.ErrorLine(err))
		}
		os.Exit(1)
	}
}
