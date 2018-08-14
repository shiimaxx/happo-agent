package command

import (
	"encoding/json"
	"net/http"

	"io/ioutil"

	"fmt"

	"github.com/codegangsta/cli"
	"github.com/heartbeatsjp/happo-agent/halib"
	"github.com/heartbeatsjp/happo-agent/util"
)

// CmdLeave implements subcommand `leave`
func CmdLeave(c *cli.Context) error {
	req := &halib.AutoScalingLeaveRequest{
		APIKey: "",
	}
	postdata, err := json.Marshal(req)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	nodeEndpoint := c.String("node-endpoint")

	res, err := util.RequestToAutoScalingLeaveAPI(nodeEndpoint, postdata)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if res.StatusCode == http.StatusNotFound {
		return cli.NewExitError("node-endpoint was not available", 1)
	}

	var autoScalingLeaveResponce halib.AutoScalingLeaveResponse
	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if err := json.Unmarshal(data, &autoScalingLeaveResponce); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if res.StatusCode != http.StatusOK {
		return cli.NewExitError(autoScalingLeaveResponce.Message, 1)
	}

	fmt.Println("Success.")

	return nil
}
