package autoscaling

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/heartbeatsjp/happo-agent/halib"
)

// ErrNotRunningEC2 represents error for not running within Amazon EC2 when daemon mode
var ErrNotRunningEC2 = errors.New("not running within Amazon EC2")

// AWSClient allows you to get the list of IP addresses of instanes of an Auto Scaling group
type AWSClient struct {
	SvcEC2         ec2iface.EC2API
	SvcAutoscaling autoscalingiface.AutoScalingAPI
}

// NewAWSClient returns AWSClient when running within Amazon EC2.
// If running in not Amazon EC2, returns ErrNotRunningEC2 as an error.
func NewAWSClient() (*AWSClient, error) {
	sess := session.Must(session.NewSession())
	ec2Meta := ec2metadata.New(session.Must(session.NewSession()))
	if !ec2Meta.Available() {
		return nil, ErrNotRunningEC2
	}

	region, err := ec2Meta.Region()
	if err != nil {
		return nil, err
	}
	return &AWSClient{
		SvcAutoscaling: autoscaling.New(sess, aws.NewConfig().WithRegion(region)),
		SvcEC2:         ec2.New(sess, aws.NewConfig().WithRegion(region)),
	}, nil
}

// EC2MetadataAPI interface of ec2metadata.EC2Metadata
type EC2MetadataAPI interface {
	Available() bool
	GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
}

// NodeAWSClient provides interface to SSM Parameter Store
type NodeAWSClient struct {
	SvcSSM         ssmiface.SSMAPI
	SvcAutoScaling autoscalingiface.AutoScalingAPI
	SvcEC2Metadata EC2MetadataAPI
}

// NewNodeAWSClient returns NodeAWSClient when running within Amazon EC2.
// If running in not Amazon EC2, returns ErrNotRunningEC2 as an error.
func NewNodeAWSClient() (*NodeAWSClient, error) {
	sess := session.Must(session.NewSession())
	ec2Meta := ec2metadata.New(session.Must(session.NewSession()))
	if !ec2Meta.Available() {
		return nil, ErrNotRunningEC2
	}

	region, err := ec2Meta.Region()
	if err != nil {
		return nil, err
	}
	return &NodeAWSClient{
		SvcSSM:         ssm.New(sess, aws.NewConfig().WithRegion(region)),
		SvcAutoScaling: autoscaling.New(sess, aws.NewConfig().WithRegion(region)),
		SvcEC2Metadata: ec2Meta,
	}, nil
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

	for _, r := range result2.Reservations {
		for _, i := range r.Instances {
			autoScalingInstances = append(autoScalingInstances, i)
		}
	}

	return autoScalingInstances, nil
}

// GetAutoScalingNodeConfigParameters returns parameters of autoscaling node config from AWS SSM Parameter Store
func (client *NodeAWSClient) GetAutoScalingNodeConfigParameters(path string) (halib.AutoScalingNodeConfigParameters, error) {
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

// GetInstanceMetadata return instance meta data
func (client *NodeAWSClient) GetInstanceMetadata() (string, string, error) {
	if client.SvcEC2Metadata.Available() {
		i, err := client.SvcEC2Metadata.GetInstanceIdentityDocument()
		if err != nil {
			return "", "", err
		}
		return i.InstanceID, i.PrivateIP, nil
	}
	return "", "", errors.New("agent is not running with EC2 Instance or metadata service is not available")
}

// GetAutoScalingGroupName return autoscaling group name
func (client *NodeAWSClient) GetAutoScalingGroupName(instanceID string) (string, error) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{}
	var groups []*autoscaling.Group

	for {
		result, err := client.SvcAutoScaling.DescribeAutoScalingGroups(input)
		if err != nil {
			return "", err
		}
		groups = append(groups, result.AutoScalingGroups...)
		if result.NextToken == nil {
			break
		}
		input.SetNextToken(*result.NextToken)
	}

	for _, a := range groups {
		for _, i := range a.Instances {
			if *i.InstanceId == instanceID {
				return *a.AutoScalingGroupName, nil
			}
		}
	}

	return "", fmt.Errorf("%s is not autoscaling node", instanceID)
}
