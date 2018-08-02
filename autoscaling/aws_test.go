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
