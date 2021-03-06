package halib

// --- Struct

// MetricConfig is struct of metric collection config yaml file
type MetricConfig struct {
	Metrics []struct {
		Hostname string `yaml:"hostname" json:"Hostname"`
		Plugins  []struct {
			PluginName   string `yaml:"plugin_name" json:"Plugin_Name"`
			PluginOption string `yaml:"plugin_option" json:"Plugin_Option"`
		} `yaml:"plugins" json:"Plugins"`
	} `yaml:"metrics" json:"Metrics"`
}

// CrawlConfigAgent is struct of actual crawl operation
type CrawlConfigAgent struct {
	GroupName   string                `yaml:"group_name" json:"group_name"`
	IP          string                `yaml:"ip" json:"ip"`
	Hostname    string                `yaml:"hostname" json:"hostname"`
	Port        int                   `yaml:"port" json:"port"`
	AutoScaling AutoScalingConfigData `yaml:"autoscaling,omitempty" json:"autoscaling,omitempty"`
	Proxies     []string              `yaml:"proxies" json:"proxies"`
	Disabled    bool                  `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

// AutoScalingConfig is struct of autoscaling config yaml file
type AutoScalingConfig struct {
	AutoScalings []AutoScalingConfigData `yaml:"autoscalings" json:"autoscalings"`
}
