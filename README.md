# happo-agent - yet another Nagios nrpe

[![wercker status](https://app.wercker.com/status/1d02bef8da5959d5b6456e25835ae026/s/ "wercker status")](https://app.wercker.com/project/byKey/1d02bef8da5959d5b6456e25835ae026)

## Description

`happo-agent` is yet another Nagios nrpe plugin. And improvement nrpe functions.

- More secure communication. Supports TLS 1.2.
- Less fork cost at bastion(proxy) mode. Proxy request handled by thread (not fork()).
- Metric collection. Compatible to Sensu plugin format.
- inventory collection.


## Usage

### Requires

- Red Hat Enterprise Linux (RHEL) 6.x, 7.x
- CentOS 6.x, 7.x
- Ubuntu 12.04 or later

### Daemon mode (for monitoring)

#### How to execute

```
/path/to/happo-agent daemon -A [Accept from IP/Subnet] -B [Public key file] -R [Private key file] -M [Metric config file (Accept empty file)]
```

**Many configuration can be with environment variables.**

See `/etc/default/happo-agent.env`
(example is in [contrib/etc/default/happo-agent.env](contrib/etc/default/happo-agent.env))

#### Monitoring

Call plugin from [`check_happo`](https://github.com/heartbeatsjp/check_happo), `happo-agent` calls local nagios plugin program. Then, return code and value to `check_happo`.

For more information, please see `check_happo` README.

#### Metric collection

Every one minute, execute sensu metrics plugin defined by `metrics.yaml`, and buffering results.

If you collect buffering results, you can use API `/metric` method.

#### Inventory collection

Get command based inventory data via API `/inventory` method.

### API client mode

You create `happo-agent` client management server if you want.

Use api client commands, `happo-agent` calls endpoint url which is client management server.

#### Host add request

```
/path/to/happo-agent add -e [ENDPOINT_URL] -g [GROUP_NAME[!SUB_GROUP_NAME]] -i [OWN_IP] -H [HOSTNAME] [-p BASTON_IP]
```

#### Is host available ?

```
/path/to/happo-agent is_added -e [ENDPOINT_URL] -g [GROUP_NAME[!SUB_GROUP_NAME]] -i [OWN_IP]
```

#### Host remove request

```
/path/to/happo-agent remove -e [ENDPOINT_URL] -g [GROUP_NAME[!SUB_GROUP_NAME]] -i [OWN_IP]
```

## Install

### Source based install (Use upstart)

```bash
$ sudo yum install epel-release
$ sudo yum install nagios-plugins-all
$ go get -dv github.com/heartbeatsjp/happo-agent
$ sudo install $GOHOME/src/bin/happo-agent /usr/local/bin/happo-agent
$ sudo install -d -m 755 /etc/happo
$ cd /etc/happo
$ sudo openssl genrsa -out happo-agent.key 2048
$ sudo openssl req -new -key happo-agent.key -sha256 -out happo-agent.csr
$ sudo openssl x509 -in happo-agent.csr -days 3650 -req -signkey happo-agent.key -sha256 -out happo-agent.pub
$ sudo touch metrics.yaml
$ sudo chmod go-rwx happo-agent.key
$ sudo install contrib/etc/default/happo-agent.env /etc/default/happo-agent.env
$ sudo install contrib/etc/init/happo-agent.conf   /etc/init/happo-agent.conf
$ sudo initctl reload-configuration
$ sudo initctl start happo-agent
```

You want to use sensu metrics plugins, should install `/usr/local/bin`.

Pre build binary maybe useful.
[Releases · heartbeatsjp/happo\-agent](https://github.com/heartbeatsjp/happo-agent/releases)

### Metric collection configuration

metrics.yaml

```
metrics:
  - hostname: [HOSTNAME]
    plugins:
    - plugin_name: [Sensu plugin name (Path not needed)]
      plugin_option: [Sensu plugin name options]
    - ...
  - ...
```

## With AWS EC2 Auto Scaling

Since the 2.0.0 release, AWS EC2 Auto Scaling is supported.

Bastion's agent stores Auto Scaling instance info like an instance id, private ip address in dbms.
Also each instances are assigned alias, that too stored in dbms too.

e.g)

※ Be careful this is different of structure of actual stored in dbms.

| Alias | Instance id | Private ip address |
|-------|-------------|--------------------|
| hb-autoscaling-web-01 | i-aaaaaa | 192.0.2.1 |
| hb-autoscaling-web-02 | i-bbbbbb | 192.0.2.2 |
| hb-autoscaling-web-03 | i-cccccc | 192.0.2.3 |
| : | : | : |

Instance info assigned to alias is automatic change according to the actual change of Auto Scaling instances.

Bastion's agent transfers request to private ip address resolved from alias when received request of proxy to alias. This concrete behave is difference depend on request type(parameter of [/proxy](#proxy)).

Monitoring

- When `request_type: monitor`, proxy to private ip address resolved from alias

Metric collection

- When `request_type: metric`, proxy to private ip address resolved from alias
- When `request_type: metric/config/update`(`proxy_hostport` need to be Auto Scaling Group Name instead of alias), proxy to active instances contained in Auto Scaling Group

Inventory collection

- When `request_type: inventory`(`proxy_hostport` need to be Auto Scaling Group Name instead of alias), proxy to one of active instance contained in Auto Scaling Group

### Subcommands

#### AutoScaling add request

Subcommand for [API client mode](#api-client-mode)

```
/path/to/happo-agent add_ag -e [ENDPOINT_URL] -g [GROUP_NAME[!SUB_GROUP_NAME]] -n [AUTOSCALING_GROUP_NAME] -H [HOST_PREFIX] -c [AUTOSCALING_COUNT] [-p BASTON_IP]
```

#### Resolve from alias to private ip

```
/path/to/happo-agent resolve_alias -b [BASTION_ENDPOINT_URL] <alias>
```

#### Deregister node from instances information stored bastion

```
/path/to/happo-agent leave -n [NODE_ENDPOINT_URL]
```

#### List aliases

```
/path/to/happo-agent list_aliases -b [BASTION_ENDPOINT_URL] -n [AUTOSCALING_GROUP_NAME] -all
```


### Setting for bastion

```bash
$ cd /etc/happo
$ sudo touch autoscaling.yaml
```

#### AutoScaling configuration

autoscaling.yaml

```
autoscalings:
- autoscaling_group_name: [AutoScaling Group Name]
  autoscaling_count: [Number of AutoScaling Group Instances]
  host_prefix: [HOSTNAME Prefix]
- ...
```

IMPORTANT NOTICE:

AutoScaling instances data is stored with DBMS, DB key prefix is composed of autoscaling group name and hostprefix (see also [DBMS](#dbms)).
You should take care about DB key confrict when update autoscaling configuration.

In this case, DB key confrict. because it will generate DB key in same prefix composed of `autoscaling_group_name` and `host_prefix`.

```
autoscalings:
- autoscaling_group_name: sysx-web-a
  autoscaling_count: 4
  host_prefix: ap
- autoscaling_group_name: sysx-web
  autoscaling_count: 4
  host_prefix: a-ap
```

### Setting for node(instance to be launched by AWS EC2 Auto Scaling)

Should be specify some parameters in `/etc/default/happo-agent.env`.

```
HAPPO_AGENT_DAEMON_AUTOSCALING_NODE="true"
HAPPO_AGENT_DAEMON_AUTOSCALING_BASTION_ENDPOINT="https://192.0.2.100:6777"
HAPPO_AGENT_DAEMON_AUTOSCALING_JOIN_WAIT_SECONDS="60"
```

You can also specify `HAPPO_AGENT_DAEMON_AUTOSCALING_BASTION_ENDPOINT` and `HAPPO_AGENT_DAEMON_AUTOSCALING_JOIN_WAIT_SECONDS` in AWS SSM Parameter Store.
See also [contrib/etc/default/happo-agent.env](https://github.com/heartbeatsjp/happo-agent/blob/master/contrib/etc/default/happo-agent.env).

Enable Upstart job of leave at node shutdown

```bash
$ sudo install contrib/etc/init/happo-agent-autoscaling-leave.conf   /etc/init/happo-agent-autoscaling-leave.conf
$ sed -ie "s/^#start on/start on/" /etc/init/happo-agent-autoscaling-leave.conf
$ sed -ie "s/^#pre-stop/pre-stop/" /etc/init/happo-agent.conf
$ sudo initctl reload-configuration
$ sudo initctl restart happo-agent
```

### Setting AWS credentials

happo-agent uses AWS API when using feature of AWS EC2 Auto Scaling, you should specify a AWS credentials setting one of the following way.
　
- EC2 IAM role
- Credentials file (`~/.aws/credentials`)
- Configuration file (`~/.aws/config`)
    - If the `AWS_SDK_LOAD_CONFIG` is set
- Environment variables

## API

- Listen port: 6777 (Default)
- HTTPS, TLS 1.2, CipherSuite: `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`

### /

Check available.

- Input format
    - None
- Return format
    - String "OK"

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/
OK
```

### /proxy

Use agent bastion(proxy) mode.

- Input format
    - JSON
- Input variables
    - proxy\_hostport:
        - (Array) bastion_ip:port. It can multiple define.
    - request\_type: request type (e.g. `monitor`)
    - request\_json: Base64 encoded JSON string to be sent to destination host.
- Return format
    - JSON
- Return variables
    - By `request_type` type.

In case `--proxy-timeout-seconds` reached, return `504 Gateway Timeout` .

If destination host is AutoScaling instance, it will behave as follows.

- `request_type: monitor` 
    - In case it can be resolved alias, proxy request to Auto Scaling instance.
    - In case it can't be resolved alias, return dummy response from bastion.
        - dummy response: `{"return_value":0,"message":"<alias> has not been assigned Instance\n"}`

```
$ echo -n '{"apikey":"","plugin_name":"check_procs","plugin_option":"-w 100 -c 200"}' | base64
eyJhcGlrZXkiOiIiLCJwbHVnaW5fbmFtZSI6ImNoZWNrX3Byb2NzIiwicGx1Z2luX29wdGlvbiI6Ii13IDEwMCAtYyAyMDAifQ==

$ wget -q --no-check-certificate -O - https://192.0.2.1:6777/proxy --post-data='{"proxy_hostport": ["198.51.100.1:6777"], "request_type": "monitor", "request_json": "eyJhcGlrZXkiOiIiLCJwbHVnaW5fbmFtZSI6ImNoZWNrX3Byb2NzIiwicGx1Z2luX29wdGlvbiI6Ii13IDEwMCAtYyAyMDAifQ=="}'
{"return_value":1,"message":"PROCS WARNING: 168 processes\n"}
```

Example calls `wget host -> https://192.0.2.1:6777/proxy -> https://198.51.100.1:6777/monitor`.

### /inventory

Get inventory information from command.

- Input format
    - JSON
- Input variables
    - apikey: ""
    - command: execute command
    - command\_option: command option
- Return format
    - JSON
- Return variables
    - return\_code: commands return code
    - return\_value: commands return value (stdout, stderr)

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/inventory --post-data='{"apikey": "", "command": "uname", "command_option": "-a"}'
{"return_code":0,"return_value":"Linux saito-hb-vm101 2.6.32-573.3.1.el6.x86_64 #1 SMP Thu Aug 13 22:55:16 UTC 2015 x86_64 x86_64 x86_64 GNU/Linux\n"}
```

### /monitor

Call monitor plugin. It likes nrpe.

- Input format
    - JSON
- Input variables
    - apikey: ""
    - command: execute nagios plugin command
    - command\_option: command option
- Return format
    - JSON
- Return variables
    - return\_code: commands return code
    - return\_value: commands return value (stdout, stderr)

In case `--command-timeout` reached, return `500 Internal Server Error` .

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/monitor --post-data='{"apikey": "", "plugin_name": "check_procs", "plugin_option": "-w 100 -c 200"}'
{"return_value":1,"message":"PROCS WARNING: 168 processes\n"}
```

### /metric

Get collected metric values.

- Input format
    - JSON
- Input variables
    - apikey: ""
- Return format
    - JSON
- Return variables
    - MetricData:
        - (Array)
            - hostname: Hostname
            - timestamp: Unix time
            - metrics: metric name - metric value (key-value)
    - Message: message from agent (if error occurred)

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/metric --post-data='{"apikey": ""}'
{"metric_data":[{"hostname":"saito-hb-vm101","timestamp":1444028730,"metrics":{"linux.context_switches.context_switches":32662,"linux.disk.elapsed.iotime_sda":52,"linux.disk.elapsed.iotime_weighted_sda":82,"linux.disk.rwtime.tsreading_sda":0,"linux.disk.rwtime.tswriting_sda":82,"linux.forks.forks":88,"linux.interrupts.interrupts":19642,"linux.ss.CLOSE-WAIT":0,"linux.ss.CLOSING":0,"linux.ss.ESTAB":9,"linux.ss.FIN-WAIT-1":0,"linux.ss.FIN-WAIT-2":0,"linux.ss.LAST-ACK":0,"linux.ss.LISTEN":31,"linux.ss.SYN-RECV":0,"linux.ss.SYN-SENT":0,"linux.ss.TIME-WAIT":7,"linux.ss.UNCONN":0,"linux.ss.UNKNOWN":0,"linux.swap.pswpin":0,"linux.swap.pswpout":0,"linux.users.users":1}},…(snip)…],"message":""}
```

### /metric/append

Append metric values. (passive metrics collection)

- Input format
    - JSON
- Input variables
    - apikey: ""
    - MetricData:
        - (Array)
            - hostname: Hostname
            - timestamp: Unix time
            - metrics: metric name - metric value (key-value)
- Return format
    - JSON
- Return variables
    - Message: message from agent (if error occurred)

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/metric/append --post-data='{"apikey": "", "metric_data":[{"hostname":"saito-hb-vm101","timestamp":1444028730,"metrics":{"linux.context_switches.context_switches":32662,"linux.disk.elapsed.iotime_sda":52,"linux.disk.elapsed.iotime_weighted_sda":82,"linux.disk.rwtime.tsreading_sda":0,"linux.disk.rwtime.tswriting_sda":82,"linux.forks.forks":88,"linux.interrupts.interrupts":19642,"linux.ss.CLOSE-WAIT":0,"linux.ss.CLOSING":0,"linux.ss.ESTAB":9,"linux.ss.FIN-WAIT-1":0,"linux.ss.FIN-WAIT-2":0,"linux.ss.LAST-ACK":0,"linux.ss.LISTEN":31,"linux.ss.SYN-RECV":0,"linux.ss.SYN-SENT":0,"linux.ss.TIME-WAIT":7,"linux.ss.UNCONN":0,"linux.ss.UNKNOWN":0,"linux.swap.pswpin":0,"linux.swap.pswpout":0,"linux.users.users":1}},...(snip)...]}'
{"status": "ok", "message": ""}
```

### /metric/config/update

*TODO*

### /metric/status

replaced to /status

### /autoscaling

List registered autoscaling instances

- Input format
    - None
- Input variables
    - None
- Return format
    - JSON
- Return variables
    - autoscaling: autoscaling list
        - (Array)
            - autoscaling_group_name: autoscaling group name
            - instances: autoscaling group instances
                - (Array)
                    - alias: alias of instance
                    - instance_data:
                        - ip: private ip address by Amazon EC2
                        - instance_id: instance id by Amazon EC2
                        - metric_plugins:
                            - (Array)
                                - plugin_name: metric plugin name
                                - plugin_option: metric plugin option

```
$ wget -q --no-check-certificate -O -  https://127.0.0.1:6777/autoscaling
{"autoscaling":[{"autoscaling_group_name":"hb-autoscaling","instances":[{"alias":"hb-autoscaling-app-1","instance_data":{"ip":"192.0.2.11","instance_id":"i-aaaaaaaaaaaaaaaaa","metric_plugins":[{"plugin_name":"","plugin_option":""}]}},{"alias":"hb-autoscaling-app-2","instance_data":{"ip":"192.0.2.12","instance_id":"i-bbbbbbbbbbbbbbbbb","metric_plugins":[{"plugin_name":"","plugin_option":""}]}},{"alias":"hb-autoscaling-app-3","instance_data":{"ip":"192.0.2.13","instance_id":"i-ccccccccccccccccc","metric_plugins":[{"plugin_name":"","plugin_option":""}]}},{"alias":"hb-autoscaling-app-4","instance_data":{"ip":"192.0.2.14","instance_id":"i-ddddddddddddddddd","metric_plugins":[{"plugin_name":"","plugin_option":""}]}}]}]}
```

### /autoscaling/resolve/:alias

Resolve ip from alias

- Input format
    - None
- Input variables
    - None
- Return format
    - JSON
- Return variables
    - status: result status
    - ip: private ip address by Amazon EC2

```
# wget -q --no-check-certificate -O -  https://127.0.0.1:6777/autoscaling/resolve/hb-autoscaling-app-1
{"Status":"OK","ip":"192.0.2.11"}
```

### /autoscaling/config/update

Update autoscaling config

- Input format
    - JSON
- Input variables
    - apikey: ""
    - config: configuration of autoscaling groups
        - autoscalings:
            - autoscaling_group_name: autoscaling group name
            - autoscaling_count: num of autoscaling instances
            - host_prefix: hostname(alias) prefix
- Return format
    - JSON
- Return variables
    - status: result status
    - message: message from agent (if error occurred)

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/autoscaling/config/update --post-data="{\"apikey\":\"\",\"config\":{\"autoscalings\":[{\"autoscaling_group_name\":\"hb-autoscaling\",\"autoscaling_count\":"4",\"host_prefix\":\"app\"}]}}"
{"status":"OK","message":""}
```

### /autoscaling/refresh

Refresh autoscaling instances

- Input format
    - JSON
- Input variables
    - autoscaling_group_name: autoscaling group name
- Return format
    - JSON
- Return variables
    - status: result status
    - message: message from agent (if error occurred)

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/autoscaling/refresh --post-data="{\"autoscaling_group_name\": \"hb-autoscaling\"}"
{"status":"OK","message":""}
```

### /autoscaling/delete

Delete autoscaling instances data

- Input format
    - JSON
- Input variables
    - autoscaling_group_name: autoscaling group name
- Return format
    - JSON
- Return variables
    - status: result status
    - message: message from agent (if error occurred)

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/autoscaling/delete --post-data="{\"autoscaling_group_name\": \"hb-autoscaling\"}"
{"status":"OK","message":""}
```

### /autoscaling/instance/register

Register autoscaling instance

- Input format
    - JSON
- Input variables
    - apikey: ""
    - autoscaling_group_name: autoscaling group name
    - ip: private ip address by Amazon EC2
    - instance_id: instance id by Amazon EC2
- Return format
    - JSON
- Return variables
    - status: result status
    - message: message from agent (if error occurred)
    - alias: assigned alias to instance
    - instance_data:
        - ip: private ip address by Amazon EC2
        - instance_id: instance id by Amazon EC2
        - metric_plugins:
            - (Array)
                - plugin_name: metric plugin name
                - plugin_option: metric plugin option

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/autoscaling/instance/register --post-data="{\"apikey\":\"\",\"autoscaling_group_name\": \"hb-autoscaling\",\"ip\":\"192.0.2.1\",\"instance_id\":\"i-aaaaaa\"}"
{"status":"OK","message":"","alias":"hb-autoscaling-web-01","instance_data":{"ip":"192.0.2.1","instance_id":"i-aaaaaa","metric_config":{"Metrics":null}}}
```

### /autoscaling/instance/deregister

Deregister autocaling instances

- Input format
    - JSON
- Input variables
    - apikey: ""
    - instance_id: instance id by Amazon EC2
- Return format
    - JSON
- Return variables
    - status: result status
    - message: message from agent (if error occurred)

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/autoscaling/instance/deregister --post-data="{\"apikey\":\"\",\"instance_id\":\"i-aaaaaa\"}"
{"status":"OK","message":""}
```

### /autoscaling/leave

Deregister node from autoscaling bastion. This handler is available only in agent running with autoscaling node

- Input format
    - JSON
- Input variables
    - apikey: ""
- Return format
    - JSON
- Return variables
    - status: result status
    - message: message from agent (if error occurred)

```
wget -q --no-check-certificate -O -  https://127.0.0.1:6777/autoscaling/leave --post-data="{\"apikey\":\"\"}"
{"status":"OK","message":""}
```

### /autoscaling/health/:alias

- Input format
    - None
- Input variables
    - None
- Return format
    - JSON
- Return variables
    - status: result status
    - message: message from agent (if error occurred)
    - ip: private ip address by Amazon EC2

```
# wget -q --no-check-certificate -O -  https://127.0.0.1:6777/autoscaling/health/hb-autoscaling-app-1
{"Status":"OK","message":"",ip:"192.0.2.1"}
```


### /status

Get happo-agent status

- Input format
    - None
- Input variables
    - None
- Return format
    - JSON
- Return variables
    - app_version: happo-agent version ( equivalent to `happo-agent -v` )
    - uptime_seconds: seconds from happo-agent started
    - num_goroutine: number of goroutine
    - metric_buffer_status
        - oldest_timestamp: oldest Timestamp(int64) in metric_data_buffer
        - newest_timestamp: newest Timestamp(int64) in metric_data_buffer
    - callers: `filepath:linenum` of each goroutines

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/status
{"app_version":"1.0.0","uptime_seconds":13,"num_goroutine":15,"metric_buffer_status":{"newest_timestamp":1505180794,"oldest_timestamp":1504852118},"callers":["/goroot/src/runtime/extern.go:219","/gopath/src/github.com/heartbeatsjp/happo-agent/model/status.go:28",...(snip)...]}
```

### /status/memory

Get happo-agent memory usage status

- Input format
    - None
- Input variables
    - None
- Return format
    - JSON
- Return variables
    - runtime.MemStatus

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/status/memory
{"Alloc":7155296,"TotalAlloc":12148632,"Sys":14395640,"Lookups":34,"Mallocs":23456,"Frees":6565,...(snip)...}%
```

### /status/request

Get request status/count.

- Input format
    - None
- Input variables
    - None
- Return format
    - JSON
- Return variables
    - last1: Last 1 Minutes results
        - url: url
        - counts:
            - `<status_code>`
            - count
    - last5: Last 5 Minutes results
        - same as last1

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/status/request
{"keys":["s-1498112479","s-1498112819"]}
{"last1":[{"url":"/","counts":{"200":3,"403":1}},{"url":"/proxy","counts":{"200":1,"403":1}}],"last5":[{"url":"/","counts":{"200":3,"403":1}},{"url":"/proxy","counts":{"200":1,"403":1}}]}
```

### /status/autoscaling

Get autoscaling status

- Input format
    - None
- Input variables
    - None
- Return format
    - JSON
- Return variables
    - (Array)
        - autoscaling_group_name: autoscaling group name
        - status: result status
        - message: message from agent (if error occurred)

```
$ wget -q --no-check-certificate -O -  https://127.0.0.1:6777/status/autoscaling
[{"autoscaling_group_name":"hb-autoscaling","status":"ok","message":""}]
```

### /machine-state

Get machine state key list.

- Input format
    - None
- Input variables
    - None
- Return format
    - JSON
- Return variables
    - keys: machine-state key list

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/machine-state
{"keys":["s-1498112479","s-1498112819"]}
```

### /machine-state/:key

Get machine state.

- Input format
    - None
- Input variables
    - key (can find from `/machine-state/` )
- Return format
    - JSON
- Return variables
    - machineState: command results

```
$ wget -q --no-check-certificate -O - https://127.0.0.1:6777/machine-state/s-1498112479
{"machineState":"********** w (2017-06-22T15:21:19+09:00) cron 15:21:19 up 13 days, ..."}
```

## DBMS

- key `m-<timestamp>` are metrics(timestamp is unixtime).
    - value: `happo_agent.MetricsData`
- key `s-<timestamp>` are saved machine state(timestamp is unixtime).
    - value: `string`
- key `ag-<autoscaling group name>-<host prefix>-<serial number>` are saved autoscaling instance data.
    - value: `happo_agent.InstanceData`

[syndtr/goleveldb: LevelDB key/value database in Go\.](https://github.com/syndtr/goleveldb)

## Experimental: Windows Support

Supported API

```
/
/proxy
/inventory
/monitor
/metric
/metric/append
/metric/config/update
/metric/status
/status
/status/memory
```

### Note1: Windows' Service Management

Use NSSM to handle happo-agent as Windows' Service

NSSM - the Non-Sucking Service Manager https://nssm.cc/

In below example, use NSSM.

1. Download NSSM binary from https://nssm.cc/download
2. Extract ZIP and put `nssm.exe` to `C:\happo-agent`
    - `C:\happo-agent\nssm.exe`

### Note2: Monitoring / Metrics Plugins

Use with sensu-plugins-windows is very convenient.
https://github.com/sensu-plugins/sensu-plugins-windows

In below example, use sensu-plugins-windows(powershell)

1. Clone or Download https://github.com/sensu-plugins/sensu-plugins-windows
    - In below example, clone to `C:\happo-agent\sensu-plugins-windows`
2. Do `Set-ExecutionPolicy RemoteSigned` in Administrative PowerShell
    1. Find `Windows PowerShell` in Start Menu
    2. Right Click => Run as Administrator
    3. `Set-ExecutionPolicy RemoteSigned` and Enter

### Example: happo-agent on Windows

1. Download happo-agent release binary for Windows from GitHub
    - https://github.com/heartbeatsjp/happo-agent/releases
    - Save as `C:\happo-agent\happo-agent.exe`
2. Install NSSM and sensu-plugins-windows
    - see above
3. Put required files(Can generate on another host)
    - Public Key : In this example, use `C:\happo-agent/etc/happo-agent.pub`
    - Private Key : In this example, use `C:\happo-agent/etc/happo-agent.key`
    - Metric Config  : In this example, use `C:\happo-agent/etc/metrics.yaml`
3. Install happo-agent as Windows' Service on administrative `cmd.exe`
    1. Find `cmd.exe` in Start Menu
    2. Right Click => Run as Administrator
    3. Install

        ```
        cd c:\happo-agent
        nssm.exe install happo-agent C:\happo-agent\happo-agent.exe daemon
        nssm.exe set happo-agent AppEnvironmentExtra PATH="%PATH%;C:\happo-agent;C:\happo-agent\sensu-plugins-windows\bin\powershell" HAPPO_AGENT_ALLOWED_HOSTS=0.0.0.0/0 HAPPO_AGENT_PUBLIC_KEY=C:\happo-agent\etc\happo-agent.pub HAPPO_AGENT_PRIVATE_KEY=C:\happo-agent\etc\happo-agent.key HAPPO_AGENT_METRIC_CONFIG=C:\happo-agent\etc\metrics.yaml HAPPO_AGENT_NAGIOS_PLUGIN_PATHS=C:\happo-agent\sensu-plugins-windows\bin\powershell HAPPO_AGENT_SENSU_PLUGIN_PATHS=C:\happo-agent\sensu-plugins-windows\bin\powershell
        nssm.exe set happo-agent AppStdout C:\happo-agent\happo-agent.out
        nssm.exe set happo-agent AppStderr C:\happo-agent\happo-agent.out
        nssm.exe start happo-agent
        ```

#### Note : When you use `happo-agent.exe` in production

When you use `happo-agent.exe` in production, below environments are useful.

```
MARTINI_ENV="production"
HAPPO_AGENT_LOG_LEVEL="warn"
```

`nssm.exe set happo-agent AppEnvironmentExtra ...` overwrites every time.
so if you want to ADD `MARTINI_ENV="production"` and `HAPPO_AGENT_LOG_LEVEL="warn"` to above installation,
do below.

```
nssm.exe set happo-agent AppEnvironmentExtra PATH="%PATH%;C:\happo-agent;C:\happo-agent\sensu-plugins-windows\bin\powershell" HAPPO_AGENT_ALLOWED_HOSTS=0.0.0.0/0 HAPPO_AGENT_PUBLIC_KEY=C:\happo-agent\etc\happo-agent.pub HAPPO_AGENT_PRIVATE_KEY=C:\happo-agent\etc\happo-agent.key HAPPO_AGENT_METRIC_CONFIG=C:\happo-agent\etc\metrics.yaml HAPPO_AGENT_NAGIOS_PLUGIN_PATHS=C:\happo-agent\sensu-plugins-windows\bin\powershell HAPPO_AGENT_SENSU_PLUGIN_PATHS=C:\happo-agent\sensu-plugins-windows\bin\powershell MARTINI_ENV="production" HAPPO_AGENT_LOG_LEVEL="warn"
nssm.exe restart happo-agent
```

## Contribution

1. Fork ([http://github.com/heartbeatsjp/happo-agent/fork](http://github.com/heartbeatsjp/happo-agent/fork))
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Run test suite with the `go test ./...` command and confirm that it passes
1. Run `gofmt -s`
1. Create a new Pull Request

### Run test suite with docker

You also can run test suite with docker on local PC.

See [helper scripts usage](contrib/development/README.md)

## About testing

Overview : Basically use `go test`

- Unit test : `go test` in CI
- Endpoint behavior test : `go test` in CI
- Regression test : automatically do in CI

`daemontest` pipeline is the daemon running test.
if you run `daemontest` on local with wercker-cli,
run below.

```bash
wercker build --pipeline daemontest
```

### about daemontest

To confirm binary will suite to the criteria.

- Case:
    - Test Duration: 2700sec(45min)
    - Monitor Requests: 10kreq/45min => 3333/3min
    - Metric Count: 200 => metrics data stored 200 metrics per minute
- Criteria:
    - CPU Usage: up to 4%
        - Monitoring agent's cpu usage shoud be small.
    - Mem Usage: up to 500MB
        - Monitoring agent's memory usage shoud be small. And more, we have to avoid memory leaking.
    - Disk Usage: up to 250KB
        - Disk Usage is almost related to the amount of storing metrics. We have to keep disk usage properly.

We know that long-long running test is good for daemon,
but max is 59min, because of Wercker's restriction.

... note about implementation:

- daemontest configurations are in wercker.yml `daemontest > steps > script.name=="test daemon behavior" > code`
    - yq filter is `.daemontest.steps[] | select(.script.name=="test daemon behavior") | .script.code`
- Case:
    - Test Duration: `TEST_DURATION_SEC`
        - to complete requests while test duration, maybe we have to change `MONITOR_REQUESTS` and `MONITOR_REQUESTS_INTERVAL`
    - Monitor Requests: `MONITOR_REQUESTS / TEST_DURATION_SEC`
    - Metric Count: `METRICS_COUNT`
- Criteria:
    - CPU Usage: `CPU_THRESHOLD_PERCENT`
    - Mem Usage: `MEM_THRESHOLD_KB`
    - Disk Usage: `DB_DISK_THRESHOLD_KB`

## Author

- [Yuichiro Saito](https://github.com/koemu)
- [Toshiaki Baba](https://github.com/netmarkjp)

## License

Copyright 2016 HEARTBEATS Corporation.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
