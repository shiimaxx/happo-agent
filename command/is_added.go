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

// CmdIsAdded implements subcommand `is_added`. Check host database
func CmdIsAdded(c *cli.Context) error {
	if c.String("endpoint") == halib.DefaultAPIEndpoint {
		return cli.NewExitError("ERROR: endpoint must set with args or environment variable", 1)
	}

	manageRequest, err := util.BindManageParameter(c)
	data, err := json.Marshal(manageRequest)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	resp, err := util.RequestToManageAPI(c.String("endpoint"), "/manage/is_added", data)
	if err != nil && resp == nil {
		return cli.NewExitError(err.Error(), 1)
	}
	if resp.StatusCode == http.StatusNotFound {
		return cli.NewExitError("Not found.", 1)
	} else if resp.StatusCode == http.StatusFound {
		fmt.Println("Found.")
		return nil
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Failed! [%d] (response body cannot be read)", resp.StatusCode), 2)
		}
		return cli.NewExitError(fmt.Sprintf("Failed! [%d] %s", resp.StatusCode, body), 2)
	}
}
