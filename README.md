Naemon config for host and service
```
service_perfdata_file_template=timestamp::$TIMET$!**!*!**!host::$HOSTNAME$!**!*!**!service::$SERVICEDESC$!**!*!**!state::$SERVICESTATEID$!**!*!**!perfdata::$SERVICEPERFDATA$!**!*!**!ciid::$_HOSTCIID$!**!*!**!ciname::$_HOSTCINAME$!**!*!**!monitoringprofile::$_HOSTMONITORINGPROFILE$!**!*!**!customer::$_HOSTCUST$!**!*!**!output::$SERVICEOUTPUT$

host_perfdata_file_template=timestamp::$TIMET$!**!*!**!host::$HOSTNAME$!**!*!**!service::CI-Alive!**!*!**!state::$HOSTSTATEID$!**!*!**!perfdata::$HOSTPERFDATA$!**!*!**!ciid::$_HOSTCIID$!**!*!**!ciname::$_HOSTCINAME$!**!*!**!monitoringprofile::$_HOSTMONITORINGPROFILE$!**!*!**!customer::$_HOSTCUST$!**!*!**!output::$HOSTOUTPUT$
```