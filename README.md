[![Build 
Status](https://github.com/szaffarano/gotas/workflows/Go%20CI/badge.svg)](https://github.com/szaffarano/gotas/actions?workflow=Go%20CI)
[![Go Report Card](https://goreportcard.com/badge/github.com/szaffarano/gotas)](https://goreportcard.com/report/github.com/szaffarano/gotas)
[![codecov](https://codecov.io/gh/szaffarano/gotas/branch/master/graph/badge.svg?token=8UPQNA4E34)](https://codecov.io/gh/szaffarano/gotas)
![GitHub](https://img.shields.io/github/license/szaffarano/gotas)

# Gotas

Gotas is a [task server](https://github.com/GothenburgBitFactory/taskserver/) Go implementation.

If we already have a mature and fully functional (and official) implementation, why reinvent the wheel then? I've got 
two main purposes, the first one is to continue learning Go, and a good way to archive it is by doing real-world 
projects.  I'm a [Task Warrior](https://github.com/GothenburgBitFactory/taskwarrior/) user and fan, and hence I'll doing 
something useful at least for my personal use.  The second reason is that I think it could be interesting to have a 
multiplatform task server that doesn't have any 3rd party libraries dependency.

## Status

Merge algorithm is fully implemented, tested against different task clients, and 
[comparing](https://github.com/szaffarano/gotas/tree/master/pkg/task/testdata/payloads) 
both taskd and gotas results. Furthermore, either the configuration files, and 
the filesystem layout is the same, so technically, switching between taskd and 
gotas is transparent.

| Feature      | Taskd | Gotas |
|--------------|-------|-------|
| sync         | ✅    | ✅    |
| init         | ✅    | ✅    |
| add user     | ✅    | ✅    |
| remove user  | ✅    | ❌    |
| suspend user | ✅    | ❌    |
| resume user  | ✅    | ❌    |
| add org      | ✅    | ✅    |
| remove org   | ✅    | ❌    |
| suspend org  | ✅    | ❌    |
| resume org   | ✅    | ❌    |
| client api   | ✅    | ❌    |


## Getting started

Disclaimer: This project is under development. Please **backup** your current 
task server data directory to avoid any possible data loss.

### Already configured taskd instance

After **backing up** your task server data directory, stop taskd and start 
gotas using the same syntax:

```sh
$ /path/to/gotas server --data /path/to/taskd-data/dir
```

or using `TASKDDATA` environment variable

```sh
$ export TASKDDATA="/path/to/taskd-data/dir"
$ /path/to/gotas server
```

Gotas will read `TASKDDATA/config` file and work as expected.

### Starting from scratch

1. Initialize `gotas` repository:

        $ gotas init --data /path/to/taskd-data/dir
2. Create an initial PKI setup.  Gotas includes an embedded command to deal with it:
    1. Create a new CA
 
            $ gotas pki -p /tmp/pki init
            INFO    /tmp/pki/ca.pem: created successfully
            INFO    /tmp/pki/ca.key: created successfully
        In case you already have an existent CA, just omit this step, and from now on, use the `-p` flag pointing it to 
        the directory where the certificate and private key are located. They have to be named `ca.pem` and `ca.key`.
        
    3. Create a new server certificate:
 
            gotas pki -p /tmp/pki add server -c $(hostname) # or just use any fqdn, or even localhost
            INFO    /tmp/pki/my-hostname.pem: created successfully
            INFO    /tmp/pki/my-hostname.key: created successfully
        You can now configure gotas in the same way taskd, i.e.:

            cat $TASKDDATA/config
            ca.cert=/tmp/pki/ca.pem
            server.cert=/tmp/pki/my-hostname.pem
            server.key=/tmp/pki/my-hostname.key
    4. Create one or more client certificates to distribute in your clients:

            $ gotas pki -p /tmp/pki add client -c john
            INFO    /tmp/pki/john.pem: created successfully
            INFO    /tmp/pki/john.key: created successfully
3. Start gotas

            $ export TASKDDATA="/path/to/taskd-data/dir"
            $ /path/to/gotas server

### Limitations

- Be aware that the `--daemon` flag is not implemented yet, so gotas will run 
  in the foreground.  
- Because gotas only runs foreground, it only logs to stdout and stderr
- CRL (Certificate Revocation List) validation is not implemented yet, so this 
  configuration will be silently ignored.
- Gotas does a full client validation (`trust=strict`), which means that this 
  configuration will be ignored as well. Future versions will implement it.
