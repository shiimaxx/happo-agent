package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/codegangsta/cli"
	"github.com/heartbeatsjp/happo-agent/halib"
	"github.com/heartbeatsjp/happo-agent/util"
)

// CmdListAliases list aliases
func CmdListAliases(c *cli.Context) error {
	bastionEndpoint := c.String("bastion-endpoint")

	res, err := util.RequestToAutoScalingAPI(bastionEndpoint)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	var autoScalingResponse halib.AutoScalingResponse
	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	if err := json.Unmarshal(data, &autoScalingResponse); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if len(autoScalingResponse.AutoScaling) < 1 {
		cli.NewExitError("Missing auto scaling group", 1)
	}

	agName := c.String("autoscaling_group_name")
	listAll := c.Bool("all")

	for _, a := range autoScalingResponse.AutoScaling {
		if agName == "" || agName == a.AutoScalingGroupName {
			for _, i := range a.Instances {
				if listAll || i.InstanceData.IP != "" {
					fmt.Println(i.Alias + "," + i.InstanceData.IP + "," + i.InstanceData.InstanceID)
				}
			}
		}
	}

	return nil
}
