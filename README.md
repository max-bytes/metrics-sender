# metrics-sender

## Intro
The purpose of metrics-sender is to take local files containing metrics data (produced by a configured Naemon instance) in a configured directory, process them, and then send their contents - formatted as Influx Line Protocol - to a suitable influx REST API. In practice this is most likely a metrics-receiver. After processing, the files are deleted.

## Configuration
see config/config.sample.yml

## Input files
The input files that can be processed need to follow a syntax. A file is processed line-by-line, and each line represents a check result.  A typical line looks like this:
```
timestamp::1623407324!**!*!**!host::bkpvieadm01_H12712005!**!*!**!service::CI-Alive!**!*!**!state::0!**!*!**!perfdata::rta=1.948000ms;3000.000000;5000.000000;0.000000 pl=0%;80;100;0!**!*!**!ciid::H12712005!**!*!**!ciname::bkpvieadm01!**!*!**!monitoringprofile::profiledev-default-mid-datadomain!**!*!**!customer::INTERN_SHARED!**!*!**!output::PING OK - Packet loss = 0%, RTA = 1.95 ms
```
A line consists of a series of key/value pairs, separated by the special delimiter sequence (!\*\*!\*!\*\*!).

A line like the above results in two metrics, one from the state and one from the perfdata.

For a more detailed explanation of the file format, have a look at the sourcecode, starting at pkg/parser/parser.go.

Sample naemon config for the service_perfdata_file_template option to produce valid files:
### Sample Naemon config for host and service
```
service_perfdata_file_template=timestamp::$TIMET$!**!*!**!host::$HOSTNAME$!**!*!**!service::$SERVICEDESC$!**!*!**!state::$SERVICESTATEID$!**!*!**!perfdata::$SERVICEPERFDATA$!**!*!**!ciid::$_HOSTCIID$!**!*!**!ciname::$_HOSTCINAME$!**!*!**!monitoringprofile::$_HOSTMONITORINGPROFILE$!**!*!**!customer::$_HOSTCUST$!**!*!**!output::$SERVICEOUTPUT$
```
```
host_perfdata_file_template=timestamp::$TIMET$!**!*!**!host::$HOSTNAME$!**!*!**!service::CI-Alive!**!*!**!state::$HOSTSTATEID$!**!*!**!perfdata::$HOSTPERFDATA$!**!*!**!ciid::$_HOSTCIID$!**!*!**!ciname::$_HOSTCINAME$!**!*!**!monitoringprofile::$_HOSTMONITORINGPROFILE$!**!*!**!customer::$_HOSTCUST$!**!*!**!output::$HOSTOUTPUT$
```

## License

This project is licensed under the **Apache 2.0 license**.

See [LICENSE](LICENSE) for more information.