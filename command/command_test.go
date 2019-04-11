package command

import (
	"flag"

	"github.com/codegangsta/cli"
)

func buildBasicContext(command, endpoint, groupname, ip string) *cli.Context {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("endpoint", endpoint, "")
	set.String("group_name", groupname, "")
	set.String("ip", ip, "")
	mockCLI := cli.NewContext(app, set, nil)
	mockCLI.Command.Name = command
	return mockCLI
}
