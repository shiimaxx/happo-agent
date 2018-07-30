package awsmock

import (
	"strings"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

// MockAutoScalingClient mock of autoscaling client
type MockAutoScalingClient struct {
	autoscalingiface.AutoScalingAPI
}

// DescribeAutoScalingGroups mock of autoscaling.DescriveAutoScalingGroup
func (m *MockAutoScalingClient) DescribeAutoScalingGroups(input *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	output := &autoscaling.DescribeAutoScalingGroupsOutput{AutoScalingGroups: []*autoscaling.Group{{}}}
	switch *input.AutoScalingGroupNames[0] {
	case "dummy-prod-ag":
		output.AutoScalingGroups[0].Instances = []*autoscaling.Instance{
			{InstanceId: aws.String("i-aaaaaa"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-bbbbbb"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-cccccc"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-dddddd"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-eeeeee"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-ffffff"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-gggggg"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-hhhhhh"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-iiiiii"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-jjjjjj"), LifecycleState: aws.String("InService")},
		}
	case "fail-dummy-prod-ag":
		output.AutoScalingGroups[0].Instances = []*autoscaling.Instance{
			{InstanceId: aws.String("i-aaaaaa"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-bbbbbb"), LifecycleState: aws.String("Terminating")},
			{InstanceId: aws.String("i-cccccc"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-dddddd"), LifecycleState: aws.String("Terminated")},
			{InstanceId: aws.String("i-eeeeee"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-ffffff"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-gggggg"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-hhhhhh"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-iiiiii"), LifecycleState: aws.String("Pending")},
			{InstanceId: aws.String("i-jjjjjj"), LifecycleState: aws.String("InService")},
		}
	case "dummy-stg-ag":
		output.AutoScalingGroups[0].Instances = []*autoscaling.Instance{
			{InstanceId: aws.String("i-kkkkkk"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-llllll"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-mmmmmm"), LifecycleState: aws.String("InService")},
			{InstanceId: aws.String("i-nnnnnn"), LifecycleState: aws.String("InService")},
		}
	case "allfali-dummy-stg-ag":
		output.AutoScalingGroups[0].Instances = []*autoscaling.Instance{
			{InstanceId: aws.String("i-kkkkkk"), LifecycleState: aws.String("Terminating")},
			{InstanceId: aws.String("i-llllll"), LifecycleState: aws.String("Terminating")},
			{InstanceId: aws.String("i-mmmmmm"), LifecycleState: aws.String("Terminating")},
			{InstanceId: aws.String("i-nnnnnn"), LifecycleState: aws.String("Terminating")},
		}
	case "nil-dummy-stg-ag":
		output.AutoScalingGroups[0].Instances = []*autoscaling.Instance(nil)
	}
	return output, nil
}

// MockEC2Client mock of ec2 client
type MockEC2Client struct {
	ec2iface.EC2API
}

// DescribeInstances mock of ec2.DescribeInstances
func (m *MockEC2Client) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	output := &ec2.DescribeInstancesOutput{Reservations: []*ec2.Reservation{}}
	reservations := []*ec2.Reservation{
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-aaaaaa"), PrivateIpAddress: aws.String("192.0.2.11")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-bbbbbb"), PrivateIpAddress: aws.String("192.0.2.12")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-cccccc"), PrivateIpAddress: aws.String("192.0.2.13")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-dddddd"), PrivateIpAddress: aws.String("192.0.2.14")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-eeeeee"), PrivateIpAddress: aws.String("192.0.2.15")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-ffffff"), PrivateIpAddress: aws.String("192.0.2.16")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-gggggg"), PrivateIpAddress: aws.String("192.0.2.17")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-hhhhhh"), PrivateIpAddress: aws.String("192.0.2.18")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-iiiiii"), PrivateIpAddress: aws.String("192.0.2.19")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-jjjjjj"), PrivateIpAddress: aws.String("192.0.2.20")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-kkkkkk"), PrivateIpAddress: aws.String("192.0.2.21")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-llllll"), PrivateIpAddress: aws.String("192.0.2.22")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-mmmmmm"), PrivateIpAddress: aws.String("192.0.2.23")}}},
		{Instances: []*ec2.Instance{{InstanceId: aws.String("i-nnnnnn"), PrivateIpAddress: aws.String("192.0.2.24")}}},
	}

	for _, instanceID := range input.InstanceIds {
		for _, r := range reservations {
			if *instanceID == *r.Instances[0].InstanceId {
				output.Reservations = append(output.Reservations, r)
			}
		}
	}

	return output, nil
}

// MockSsmClient mock of ssm client
type MockSsmClient struct {
	ssmiface.SSMAPI
}

// GetParametersByPath mock of ssm.GetParametersByPath
func (m *MockSsmClient) GetParametersByPath(input *ssm.GetParametersByPathInput) (*ssm.GetParametersByPathOutput, error) {
	output := &ssm.GetParametersByPathOutput{Parameters: []*ssm.Parameter{}}
	parameters := []*ssm.Parameter{
		{
			Name:  aws.String("/happo-agent-env-1/HAPPO_AGENT_DAEMON_AUTOSCALING_BASTION_ENDPOINT"),
			Value: aws.String("http://192.0.2.100:6777"),
		},
		{
			Name:  aws.String("/happo-agent-env-1/HAPPO_AGENT_DAEMON_AUTOSCALING_JOIN_WAIT_SECONDS"),
			Value: aws.String("10"),
		},
		{
			Name:  aws.String("/happo-agent-env-2/HAPPO_AGENT_DAEMON_AUTOSCALING_BASTION_ENDPOINT"),
			Value: aws.String("http://192.0.2.200:6777"),
		},
		{
			Name:  aws.String("/happo-agent-env-3/HAPPO_AGENT_DAEMON_AUTOSCALING_JOIN_WAIT_SECONDS"),
			Value: aws.String("20"),
		},
		{
			Name:  aws.String("/happo-agent-env-4/HAPPO_AGENT_DAEMON_AUTOSCALING_BASTION_ENDPOINT"),
			Value: aws.String(""),
		},
		{
			Name:  aws.String("/happo-agent-env-4/HAPPO_AGENT_DAEMON_AUTOSCALING_JOIN_WAIT_SECONDS"),
			Value: aws.String(""),
		},
	}

	for _, p := range parameters {
		if strings.HasPrefix(*p.Name, fmt.Sprintf("%s/", *input.Path)) {
			output.Parameters = append(output.Parameters, p)
		}
	}

	return output, nil
}
