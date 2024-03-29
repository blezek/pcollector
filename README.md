# pcollector

PRTG Collector based on bosun/scollector

## Usage

[PRTG](https://www.paessler.com/prtg) supports
[custom sensors](https://prtg.paessler.com/api.htm?username=demo&password=demodemo&tabid=7),
allowing arbitrary channels of data to be sent to the PRTG server.  [Bosun](http://bosun.org) contains a sub-project ([scollector](http://bosun.org/scollector/)) to collect data for OpenTSB.  `pcollector` brings these two together so data can be pushed into PRTG using the same framework af `scollector`.

Usage:

```
Usage of pcollector:
  -aws="": AWS keys and region, format: "access_key:secret_key@region".
  -b=0: OpenTSDB batch size. Used for debugging bad data.
  -c="": External collectors directory.
  -conf="": Location of configuration file. Defaults to scollector.conf in directory of the scollector executable.
  -d=false: Enables debug output.
  -e="": Excludes collectors matching this term, multiple terms separated by comma. Works with all other arguments.
  -f="": Filters collectors matching this term, multiple terms separated by comma. Works with all other arguments.
  -fake=0: Generates X fake data points on the test.fake metric per second.
  -freq="15": Set the default frequency in seconds for most collectors.
  -h="": Bosun or OpenTSDB host. Ex: "http://bosun.example.com:8070".
  -hostname="": If set, use as value of host tag instead of system hostname.
  -i="": ICMP host to ping of the format: "host[,host...]".
  -l=false: List available collectors.
  -m=false: Disable sending of metadata.
  -n=false: Disable sending of scollector self metrics.
  -p=false: Print to screen instead of sending to a host
  -s="": SNMP host to poll of the format: "community@host[,community@host...]".
  -t="": Tags to add to every datapoint in the format dc=ny,rack=3. If a collector specifies the same tag key, this one will be overwritten. The host tag is not supported.
  -u=false: Enables full hostnames: doesn't truncate to first ".".
  -v="": vSphere host to poll of the format: "user:password@host[,user:password@host...]".
  -version=false: Prints the version and exits.
```

The major difference from `scollector` is the addition of the `-e` flag, to exclude collectors.



