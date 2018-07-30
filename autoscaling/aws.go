package autoscaling

import (
	"strconv"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/heartbeatsjp/happo-agent/halib"
)

// AWSClient allows you to get the list of IP addresses of instanes of an Auto Scaling group
type AWSClient struct {
	SvcEC2         ec2iface.EC2API
	SvcAutoscaling autoscalingiface.AutoScalingAPI
}

// NewAWSClient return AWSClient
func NewAWSClient() *AWSClient {
	sess := session.Must(session.NewSession())
	return &AWSClient{
		SvcAutoscaling: autoscaling.New(sess, aws.NewConfig().WithRegion("ap-northeast-1")),
		SvcEC2:         ec2.New(sess, aws.NewConfig().WithRegion("ap-northeast-1")),
	}
}

// AWSSsmClient provides interface to SSM Parameter Store
type AWSSsmClient struct {
	SvcSSM ssmiface.SSMAPI
}

// NewAWSSsmClient return AWSSsmClient
func NewAWSSsmClient() *AWSSsmClient {
	sess := session.Must(session.NewSession())
	return &AWSSsmClient{
		SvcSSM: ssm.New(sess, aws.NewConfig().WithRegion("ap-northeast-1")),
	}
}

func (client *AWSClient) describeAutoScalingInstances(autoScalingGroupName string) ([]*ec2.Instance, error) {
	var autoScalingInstances []*ec2.Instance

	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(autoScalingGroupName),
		},
	}

	result, err := client.SvcAutoscaling.DescribeAutoScalingGroups(input)
	if err != nil {
		return nil, err
	}
	if len(result.AutoScalingGroups) < 1 || result.AutoScalingGroups[0].Instances == nil {
		return autoScalingInstances, nil
	}

	var instanceIds []*string
	for _, instance := range result.AutoScalingGroups[0].Instances {
		if *instance.LifecycleState == "InService" {
			instanceIds = append(instanceIds, aws.String(*instance.InstanceId))
		}
	}
	if len(instanceIds) < 1 {
		return autoScalingInstances, nil
	}

	input2 := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	}

	result2, err := client.SvcEC2.DescribeInstances(input2)
	if err != nil {
		return nil, err
	}
	if len(result2.Reservations) < 1 {
		return autoScalingInstances, nil
	}

	for _, r := range result2.Reservations {
		autoScalingInstances = append(autoScalingInstances, r.Instances[0])
	}

	return autoScalingInstances, nil
}

// GetAutoScalingNodeConfigParameters returns parameters of autoscaling node config from AWS SSM Parameter Store
func (client *AWSSsmClient) GetAutoScalingNodeConfigParameters(path string) (halib.AutoScalingNodeConfigParameters, error) {
	input := &ssm.GetParametersByPathInput{
		Path: aws.String(path),
	}

	result, err := client.SvcSSM.GetParametersByPath(input)
	if err != nil {
		return halib.AutoScalingNodeConfigParameters{}, err
	}

	if len(result.Parameters) < 1 {
		return halib.AutoScalingNodeConfigParameters{}, fmt.Errorf("parameter store not found: %s", path)
	}

	var nodeConfigParameters halib.AutoScalingNodeConfigParameters
	for _, p := range result.Parameters {
		if *p.Name == fmt.Sprintf("%s/HAPPO_AGENT_DAEMON_AUTOSCALING_BASTION_ENDPOINT", path) {
			nodeConfigParameters.BastionEndpoint = *p.Value
		}
		if *p.Name == fmt.Sprintf("%s/HAPPO_AGENT_DAEMON_AUTOSCALING_JOIN_WAIT_SECONDS", path) {
			joinWaitSeconds, err := strconv.Atoi(*p.Value)
			if err != nil {
				return halib.AutoScalingNodeConfigParameters{}, err
			}
			nodeConfigParameters.JoinWaitSeconds = joinWaitSeconds
		}
	}

	return nodeConfigParameters, nil
}
