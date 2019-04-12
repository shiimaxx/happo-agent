package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/codegangsta/cli"
	"github.com/heartbeatsjp/happo-agent/halib"
	"github.com/heartbeatsjp/happo-agent/util"
)

// CmdAdd implements subcommand `add`
func CmdAdd(c *cli.Context) error {
	if c.String("endpoint") == halib.DefaultAPIEndpoint {
		return cli.NewExitError("ERROR: endpoint must set with args or environment variable", 1)
	}

	manageRequest, err := util.BindManageParameter(c)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	data, err := json.Marshal(manageRequest)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	resp, err := util.RequestToManageAPI(c.String("endpoint"), "/manage/add", data)
	if err != nil && resp == nil {
		return cli.NewExitError(err.Error(), 1)
	}
	if resp.StatusCode == http.StatusFound {
		return cli.NewExitError("Conflict!", 1)
	} else if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			cli.NewExitError(fmt.Sprintf("Failed! [%d] (response body cannot be read)", resp.StatusCode), 1)
		}
		return cli.NewExitError(fmt.Sprintf("Failed! [%d] %s", resp.StatusCode, body), 1)
	}
	fmt.Println("Success.")
	return nil
}
