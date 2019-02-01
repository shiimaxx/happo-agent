package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/heartbeatsjp/happo-agent/halib"
	"github.com/heartbeatsjp/happo-agent/util"
	"github.com/pkg/errors"
)

// CmdListAliases list aliases
func CmdListAliases(c *cli.Context) error {
	bastionEndpoint := c.String("bastion-endpoint")
	agName := c.String("autoscaling_group_name")
	listAll := c.Bool("all")

	out, err := listAliases(bastionEndpoint, agName, listAll)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	fmt.Println(out)

	return nil
}

func listAliases(bastionEndpoint, agName string, listAll bool) (string, error) {
	res, err := util.RequestToAutoScalingAPI(bastionEndpoint)
	if err != nil {
		return "", err
	}

	var autoScalingResponse halib.AutoScalingResponse
	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(data, &autoScalingResponse); err != nil {
		return "", err
	}

	if len(autoScalingResponse.AutoScaling) < 1 {
		return "", errors.New("Missing auto scaling group")
	}

	var outs []string
	for _, a := range autoScalingResponse.AutoScaling {
		if agName == "" || agName == a.AutoScalingGroupName {
			for _, i := range a.Instances {
				if listAll || i.InstanceData.IP != "" {
					outs = append(outs, strings.Join([]string{i.Alias, i.InstanceData.IP, i.InstanceData.InstanceID}, ","))
				}
			}
		}
	}

	return strings.Join(outs, "\n"), nil
}
