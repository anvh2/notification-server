package main

import (
	"github.com/anvh2/notification-server/cmd"
)

const revision = "1.0.0"

func main() {
	cmd.SetRevision(revision)
	cmd.Execute()
}
