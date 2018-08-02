package autoscaling

import (
	"testing"

	"github.com/heartbeatsjp/happo-agent/autoscaling/awsmock"
	"github.com/heartbeatsjp/happo-agent/halib"
	"github.com/stretchr/testify/assert"
)

func TestGetAutoScalingNodeConfigParameters(t *testing.T) {
	var cases = []struct {
		name         string
		input        string
		expected     halib.AutoScalingNodeConfigParameters
		isNormalTest bool
	}{
		{
			name:  "/happo-agent-env-1",
			input: "/happo-agent-env-1",
			expected: halib.AutoScalingNodeConfigParameters{
				BastionEndpoint: "http://192.0.2.100:6777",
				JoinWaitSeconds: 10,
			},
			isNormalTest: true,
		},
		{
			name:  "/happo-agent-env-2",
			input: "/happo-agent-env-2",
			expected: halib.AutoScalingNodeConfigParameters{
				BastionEndpoint: "http://192.0.2.200:6777",
				JoinWaitSeconds: 0,
			},
			isNormalTest: true,
		},
		{
			name:  "/happo-agent-env-3",
			input: "/happo-agent-env-3",
			expected: halib.AutoScalingNodeConfigParameters{
				BastionEndpoint: "",
				JoinWaitSeconds: 20,
			},
			isNormalTest: true,
		},
		{
			name:         "/happo-agent-env-4",
			input:        "/happo-agent-env-4",
			expected:     halib.AutoScalingNodeConfigParameters{},
			isNormalTest: false,
		},
		{
			name:         "/happo-agent-env-5",
			input:        "/happo-agent-env-5",
			expected:     halib.AutoScalingNodeConfigParameters{},
			isNormalTest: false,
		},
	}

	client := &NodeAWSClient{
		SvcSSM: &awsmock.MockSsmClient{},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := client.GetAutoScalingNodeConfigParameters(c.input)
			if c.isNormalTest {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestGetInstanceMetadata(t *testing.T) {
	var cases = []struct {
		name        string
		isAvailable bool
		hasError    bool
		expected    struct {
			instanceID string
			ip         string
		}
	}{
		{
			name:        "default",
			isAvailable: true,
			hasError:    false,
			expected: struct {
				instanceID string
				ip         string
			}{instanceID: "i-aaaaaa", ip: "192.0.2.11"},
		},
		{
			name:        "ec2metadata is not available",
			isAvailable: false,
			hasError:    false,
			expected: struct {
				instanceID string
				ip         string
			}{instanceID: "", ip: ""},
		},
		{
			name:        "ec2metadata has error",
			isAvailable: false,
			hasError:    true,
			expected: struct {
				instanceID string
				ip         string
			}{instanceID: "", ip: ""},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := &NodeAWSClient{
				SvcEC2Metadata: &awsmock.MockEC2MetadataClient{
					IsAvailable: c.isAvailable,
					HasError:    c.hasError,
				},
			}
			actualInstanceID, actualIP, err := client.GetInstanceMetadata()

			if c.isAvailable && !c.hasError {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
			assert.Equal(t, c.expected.instanceID, actualInstanceID)
			assert.Equal(t, c.expected.ip, actualIP)
		})
	}
}

func TestGetAutoScalingGroupName(t *testing.T) {
	var cases = []struct {
		name         string
		input        string
		expected     string
		isNormalTest bool
	}{
		{
			name:         "i-aaaaaa dummy-prod-ag",
			input:        "i-aaaaaa",
			expected:     "dummy-prod-ag",
			isNormalTest: true,
		},
		{
			name:         "i-kkkkkk dummy-stg-ag",
			input:        "i-kkkkkk",
			expected:     "dummy-stg-ag",
			isNormalTest: true,
		},
		{
			name:         "empty",
			input:        "",
			expected:     "",
			isNormalTest: false,
		},
	}

	client := &NodeAWSClient{
		SvcAutoScaling: &awsmock.MockAutoScalingClient{},
	}

	for _, c := range cases {
		result, err := client.GetAutoScalingGroupName(c.input)
		if c.isNormalTest {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
		assert.Equal(t, c.expected, result)
	}
}
