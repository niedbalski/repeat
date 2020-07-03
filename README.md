# Repeat [![CircleCI](https://circleci.com/gh/niedbalski/repeat.svg?style=svg)](https://circleci.com/gh/niedbalski/repeat)

*repeat* is a data collection tool for linux, with the following features:

* Collections (command runs) can be run on a given time periodicity or a single shot.
* Command output can be processed and stored on files or on a local database for further analysis. 
* Collections can be shared through imports (local or via http[s]+)
* A compressed tarball report will be generated.

### Installation

Release artifacts can be found in the releases section [Github Releases](https://github.com/niedbalski/repeat/releases)
```shell script
wget -c https://github.com/niedbalski/repeat/releases/download/v0.0.1/repeat-0.0.1.linux-amd64.tar.gz -O - | tar -xz -C . --strip=1
./repeat --help
```
For installing the latest master build snap (edge channel):
```shell script
snap install --channel edge repeat --classic
```

For installing the latest stable version snap (stable channel):
```shell script
snap install --channel stable repeat --classic
```

Also docker images are available:

* Amd64 docker container [![Docker Repository on Quay](https://quay.io/repository/niedbalski/repeat-linux-amd64/status "Docker Repository on Quay")](https://quay.io/repository/niedbalski/repeat-linux-amd64)
* Arm64 docker container [![Docker Repository on Quay](https://quay.io/repository/niedbalski/repeat-linux-arm64/status "Docker Repository on Quay")](https://quay.io/repository/niedbalski/repeat-linux-arm64)

```shell script
docker run -v "$(pwd):/config" -it quay.io/niedbalski/repeat-linux-amd64:master --config /config/example_metrics.yaml
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
      --db-dir="."       Path to store the local results database 
```

#### Running with configuration

An example of running the collection for 5s (could be expressed in s,m,hours)

```shell script
repeat --config metrics.yaml --timeout=5s --results-dir=.
```

#### Example configuration

* *Note* : Imports are allowed as http[s]/files, local collection names have precedence over imported ones.
* *Note2* : database storage and fields configuration are totally user-defined.

```yaml
import:
  - https://raw.githubusercontent.com/niedbalski/repeat/master/example_metrics.yaml#md5sum=6c5b5d8fafd343d5cf452a7660ad9dd1

collections:
  tcp_mem:
    command: cat /proc/sys/net/ipv4/tcp*mem
    run-every: 2s
    exit-codes: 0

  # scripts can be defined inline
  sar:
    run-once: true
    exit-codes: 0 127 126
    script: |
      #!/bin/bash

      echo "testing"

  process_list:
    command: ps aux --no-headers
    run-every: 1s
    exit-codes: any
    # store type database, will create a table in the collections database
    # and use the map-values definition to populate each column for th given
    # command output
    store: database
    database:
      map-values:
        field-separator: " "
        fields:
          - name: rss
            type: int
            field-index: 5
          - name: vsz
            type: int
            field-index: 4
          - name: pid
            type: string
            field-index: 1

  sockstat_tcp:
    command: grep -i tcp /proc/net/sockstat
    run-every: 1s
    exit-codes: any
    store: database
    database:
      map-values:
        field-separator: " "
        fields:
          - name: inuse
            type: int
            field-index: 2
          - name: alloc
            type: int
            field-index: 8
```

This command will generate the following report structure:

```shell script
$ tar -xvf repeat-report-2020-07-04-00-05.tar.gz 
repeat-077356600/collections.db
repeat-077356600/collections.db-journal
repeat-077356600/run-script-557986359
repeat-077356600/sar-2020-07-04-00:05:04
repeat-077356600/tcp_mem-2020-07-04-00:05:04
repeat-077356600/tcp_mem-2020-07-04-00:05:06
repeat-077356600/tcp_mem-2020-07-04-00:05:08
repeat-077356600/tcp_mem-2020-07-04-00:05:10
repeat-077356600/tcp_mem-2020-07-04-00:05:12
repeat-077356600/tcp_mem-2020-07-04-00:05:14
```
* Each file represents a single command run on the given periodicity
* Collections with storage type of database, will store its results in the corresponding tables,
as an example:

```shell script
$ sqlite3 repeat-077356600/collections.db 
SQLite version 3.31.1 2020-01-27 19:55:54
Enter ".help" for usage hints.
sqlite> select avg(rss) from process_list;
9837.64936336925
sqlite> select min(inuse), max(inuse), avg(inuse) from sockstat_tcp;
23|23|23.0
```


### Contributing

Feel free to send PR(s) or reach niedbalski on #freenode or Telegram.