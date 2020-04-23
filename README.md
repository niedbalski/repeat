### Repeat

*repeat* allows you to define a set of linux commands that needs to be run with a given periodicity and gather the output 
of those commands into a compressed tarball report for further analysis.

### Installation

The exporter is available on the https://snapcraft.io/repeat

For installing the latest master build (edge channel):
```shell script
snap install --channel edge repeat
```

For installing the latest stable version (stable channel):
```shell script
snap install --channel stable repeat
```

Also docker images are available:

* Amd64 docker container [![Docker Repository on Quay](https://quay.io/repository/niedbalski/repeat-linux-amd64/status "Docker Repository on Quay")](https://quay.io/repository/niedbalski/repeat-linux-amd64)
* Arm64 docker container [![Docker Repository on Quay](https://quay.io/repository/niedbalski/repeat-linux-arm64/status "Docker Repository on Quay")](https://quay.io/repository/niedbalski/repeat-linux-arm64)

```shell script
docker run -v "$HOME/metrics.yaml":/config.yaml -it quay.io/niedbalski/repeat-linux-amd64 --config config.yaml (params)
```

#### Command line

```shell script
usage: repeat --config=CONFIG [<flags>]

Flags:
  -h, --help             Show context-sensitive help (also try --help-long and --help-man).
  -l, --loglevel="info"  Log level: [debug, info, warn, error, fatal]
  -t, --timeout=0s       Timeout: overall timeout for all collectors
  -c, --config=CONFIG    Path to collectors configuration file
  -b, --basedir="/tmp"   Temporary base directory to create the resulting collection tarball
  -r, --results-dir="."  Directory to store the resulting collection tarball

```

##### Example configuration

```yaml
collections:
  lsof:
    command: lsof -i # command to run
    run-every: 10s  # periodicity 
    exit-codes: 0 # allowed exit codes (space separed list of accepted exit codes) 

  sockstat:
    command: cat /proc/sys/net/ipv4/tcp*mem /proc/net/sockstat
    run-every: 2s
    exit-codes: any

  sar:
    run-once: true   #it can be run a single time
    exit-codes: 0 127 126
    script: |    # a script can be given instead of a command.
      #!/bin/bash

      sar -n EDEV && true

  uname:
    run-once: true
    script: |
      netstat -atn
```

#### Running with configuration

An example of running the collection for 5s (could be expressed in s,m,hours)

```shell script
repeat --config metrics.yaml --timeout=5s --results-dir=.
```

This command will generate the following output:

```shell script
INFO[2020-04-23 17:57:13] Loading collectors from configuration file: example_metrics.yaml 
INFO[2020-04-23 17:57:13] Scheduler timeout set to: 5.000000 seconds   
INFO[2020-04-23 17:57:13] Scheduling run of sleep collector every 2.000000 secs 
INFO[2020-04-23 17:57:13] Scheduling run of sockstat collector every 2.000000 secs 
INFO[2020-04-23 17:57:13] Scheduling run of sar collector every 0.000000 secs 
INFO[2020-04-23 17:57:13] Scheduling run of uname collector every 0.000000 secs 
INFO[2020-04-23 17:57:13] Scheduling run of lsof collector every 10.000000 secs 
INFO[2020-04-23 17:57:14] Running command for collector sleep          
INFO[2020-04-23 17:57:14] Running command for collector uname          
INFO[2020-04-23 17:57:14] Running command for collector lsof           
INFO[2020-04-23 17:57:14] Running command for collector sar            
INFO[2020-04-23 17:57:14] Running command for collector sockstat       
INFO[2020-04-23 17:57:14] Command for collector sleep, successfully ran, stored results into file: /tmp/repeat-044825910/sleep-2020-04-23-17:57:14 
INFO[2020-04-23 17:57:14] Command for collector uname, successfully ran, stored results into file: /tmp/repeat-044825910/uname-2020-04-23-17:57:14 
ERRO[2020-04-23 17:57:14] Command for collector lsof exited with exit code: exit status 255 - (not allowed by exit-codes config) 
INFO[2020-04-23 17:57:14] Command for collector sar, successfully ran, stored results into file: /tmp/repeat-044825910/sar-2020-04-23-17:57:14 
INFO[2020-04-23 17:57:14] Command for collector sockstat, successfully ran, stored results into file: /tmp/repeat-044825910/sockstat-2020-04-23-17:57:14 
INFO[2020-04-23 17:57:15] Running command for collector uname          
INFO[2020-04-23 17:57:15] Command for collector uname, successfully ran, stored results into file: /tmp/repeat-044825910/uname-2020-04-23-17:57:15 
INFO[2020-04-23 17:57:16] Running command for collector sockstat       
INFO[2020-04-23 17:57:16] Running command for collector uname          
INFO[2020-04-23 17:57:16] Running command for collector sleep          
INFO[2020-04-23 17:57:16] Command for collector sockstat, successfully ran, stored results into file: /tmp/repeat-044825910/sockstat-2020-04-23-17:57:16 
INFO[2020-04-23 17:57:16] Command for collector sleep, successfully ran, stored results into file: /tmp/repeat-044825910/sleep-2020-04-23-17:57:16 
INFO[2020-04-23 17:57:16] Command for collector uname, successfully ran, stored results into file: /tmp/repeat-044825910/uname-2020-04-23-17:57:16 
INFO[2020-04-23 17:57:17] Running command for collector uname          
INFO[2020-04-23 17:57:17] Command for collector uname, successfully ran, stored results into file: /tmp/repeat-044825910/uname-2020-04-23-17:57:17 
INFO[2020-04-23 17:57:18] Scheduler timeout (5.000000) reached, cleaning up and killing process 
INFO[2020-04-23 17:57:18] Cleaning up resources                        
INFO[2020-04-23 17:57:18] Creating report tarball at: repeat-report-2020-04-23-17-57.tar.gz  
Killed
```

### Contributing

Feel free to send PR(s) or reach niedbalski on #freenode or Telegram.
