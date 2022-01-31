package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/slack/bootstrap"
)

const (
	DefaultPHPServerConfig = "defaults.yaml"
	ConfigDir              = "config"

	y   = "y"
	yes = "yes"
	n   = "n"
	no  = "no"
)

func main() {
	b := bootstrap.Client{}
	err := b.Run(ConfigDir, DefaultPHPServerConfig)
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Do you want to continue? [y/n]")

	input, err := reader.ReadString('\n')
	if err != nil {
		panic(errors.Wrap(err, "stdin error"))
	}

	input = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(input), "\n", ""))

	if input == n || input == no {
		os.Exit(0)
	}

	if input != n && input != no &&
		input != y && input != yes {
		fmt.Println("Abort")
		os.Exit(0)
	}

	err = b.Apply()
	if err != nil {
		fmt.Printf("apply error %v\n", err)
	}
}
