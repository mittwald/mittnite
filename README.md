[![Build Status](https://travis-ci.com/mittwald/mittnite.svg?branch=master)](https://travis-ci.com/mittwald/mittnite)

# Mittnite - Smart init system for containers

Mittnite is a small, but smart init system designed for usage as `ENTRYPOINT` in container images.

It offers the following features:

- Render configuration files from templates using Go's [`text/template`](https://golang.org/pkg/text/template/) engine.
- Start processes and manage their lifecycle
- Watch configuration files and send configurable signals to processes on change
- Wait until required services are up before starting processes (currently supporting filesystem mounts, HTTP services, MySQL, Redis, AMQP and MongoDB)

## Table of contents

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Getting started](#getting-started)
  - [CLI usage](#cli-usage)
    - [Basic](#basic)
    - [Render templates and execute custom command](#render-templates-and-execute-custom-command)
  - [Docker](#docker)
    - [Build your (go) application on top of the `mittnite` docker-image](#build-your-go-application-on-top-of-the-mittnite-docker-image)
    - [Download `mittnite` in your own custom `Dockerfile`](#download-mittnite-in-your-own-custom-dockerfile)
- [Configuration](#configuration)
  - [Directives](#directives)
    - [Job](#job)
    - [Boot Jobs](#boot-jobs)
    - [File](#file)
    - [Probe](#probe)
  - [HCL examples](#hcl-examples)
    - [Start a process](#start-a-process)
    - [Start a process lazily on first request](#start-a-process-lazily-on-first-request)
    - [Render a file on startup](#render-a-file-on-startup)
    - [Wait until a Redis connection is possible](#wait-until-a-redis-connection-is-possible)
  - [More examples](#more-examples)
- [mittnitectl](#mittnitectl)
<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Getting started

### CLI usage

#### Basic
```bash
$ mittnite --help

Mittnite is a small, but smart init system designed for usage as `ENTRYPOINT` in container images

Usage:
  mittnite [flags]
  mittnite [command]

Available Commands:
  help        Help about any command
  renderfiles
  up
  version     Show extended information about the current version of mittnite

Flags:
  -c, --config-dir string   set directory to where your .hcl-configs are located (default "/etc/mittnite.d")
  -h, --help                help for mittnite

Use "mittnite [command] --help" for more information about a command.
```

#### Render templates and execute custom command
This will render all template files and execute the `sleep 10` afterwards.
```bash
$ mittnite renderfiles sleep 10
```

### Docker
#### Build your (go) application on top of the `mittnite` docker-image
In order to run your own static application - e.g. a `golang`-binary with `mittnite`, we recommend to inherit the `mittnite` docker-image and copy your stuff on top.
```dockerfile
FROM            quay.io/mittwald/mittnite:stable
COPY            mittnite.d/ /etc/mittnite.d/
COPY            myApplication /usr/local/bin/
# ENTRYPOINT and CMD are optional, because they are inherited by parent image
```

#### Download `mittnite` in your own custom `Dockerfile`
If you'd like to use `mittnite` for non-static applications like `node` or similar, you can download the `mittnite`-binary from Github.
```dockerfile
FROM        node:12-alpine
ENV         MITTNITE_VERSION="1.1.2"
RUN         wget -qO- https://github.com/mittwald/mittnite/releases/download/v${MITTNITE_VERSION}/mittnite_${MITTNITE_VERSION}_linux_x86_64.tar.gz \
                | tar xvz -C /usr/bin mittnite && \
            chmod +x /usr/bin/mittnite
COPY        mittnite.d/ /etc/mittnite.d/
ENTRYPOINT  ["/usr/bin/mittnite"]
CMD         ["up","--config-dir", "/etc/mittnite.d"]
```

## Configuration

The directory specified with `--config-dir`, or the shorthand `-c`, can contain any number of `.hcl` configuration files.

All files in that directory are loaded by `mittnite` on startup and can contain any of the configuration directives.

### Directives

#### Job

A `Job` describes a runnable process that should be started by mittnite on startup.

A `Job` consists of a `command` and (optional) `args`. When a process started by a Job fails, it will be restarted for a maximum of `maxAttempts` attempts. If it fails for more than `maxAttempts` time, mittnite itself will terminate to allow your container runtime to handle the failure.

```hcl
job "foo" {
  command = "/usr/local/bin/foo"
  args = ["bar"]
  maxAttempts = 3
  canFail = false
  workingDirectory = "/some/path"
}
```

Set `maxAttempts` to `-1` will restart the process "forever".

```hcl
job "foo" {
  command = "/usr/local/bin/foo"
  args = ["bar"]
  maxAttempts = -1
  canFail = false
  workingDirectory = "/some/path"
}
```

You can append a custom environment to the process by setting `env`:

```hcl
job "foo" {
  command = "/usr/local/bin/foo"
  args = ["bar"]
  env = ["ENABLED=1", "BAR=\"BAZ\""]
  maxAttempts = 3
  canFail = false
}
```

To redirect the output of a job to a separate file, `stdout` and/or `stderr` can be specified:

```hcl
job "foo" {
  command = "/usr/local/bin/foo"
  args = ["bar"]
  stdout = "/tmp/foo.log"
  stderr = "/tmp/foo-errors.log"
}
```

Additionally, you can enable timestamps for the output of a job using `enableTimestamps` and specify a custom format using `timestampFormat`.

Formats are named after their constant name in the Golang [`time` package](https://pkg.go.dev/time#pkg-constants) (lookup table at the bottom).

You can also specify your own format by setting `customTimestampFormat` to a custom format string like "2006-01-02 15:04:05". Whatever is set in `timestampFormat` will be ignored in that case.

```hcl
job "foo" {
  command = "/usr/local/bin/foo"
  args = ["bar"]
  stdout = "/tmp/foo.log"
  stderr = "/tmp/foo-errors.log"
  enableTimestamps = true
  timestampFormat = "RFC3339"  # default
  customTimestampFormat = ""  # default
}
```

You can configure a Job to watch files and to send a signal to the managed process if that file changes. This can be used, for example, to send a `SIGHUP` to a process to reload its configuration file when it changes.
  
```hcl
job "foo" {
  // ...

  watch "/etc/conf.d/barfoo" {
    signal = 12
  }
}
```

If the running process cannot handle a signal to reload the configuration, the `restart` parameter can be set to `true`. In this case the specified `signal` will be sent to gracefully terminate the process and restart it afterwards:

```hcl
job "foo" {
  // ...

  watch "/etc/conf.d/barfoo" {
    signal = 15 # SIGTERM
    restart = true
  }
}
```

In addition, it is possible to execute a command before and/or after signaling.
This command should not modify the file being watched, otherwise the watcher might enter an infinite loop.

```hcl
job "foo" {
  // ...

  watch "/etc/conf.d/barfoo" {
    signal = 12

    preCommand {
      command = "echo"
      args = ["before"]
    }

    postCommand {
      command = "echo"
      args = ["after"]
    }
  }
}
```

You can also configure a Job to start its process only on the first incoming request (a bit like [systemd's socket activation](https://www.freedesktop.org/software/systemd/man/systemd.socket.html)). In order to configure this, you need a `listener` and a `lazy` configuration:

```hcl
job "foo" {
  command = "http-server"
  args = ["-p8081", "-a127.0.0.1"]

  lazy {
    spinUpTimeout = "5s"
    coolDownTimeout = "15m"
  }

  listen "0.0.0.0:8080" {
    forward = "127.0.0.1:8081"
  }
}
```

The `listen` block will instruct mittnite itself to listen on the specified address; each connection accepted on that port will be forwarded to the address specified by `forward` (**NOTE**: mittnite will do some simple layer-4 forwarding; if your upstream service depends on working with the actual client IP addresses, you'll only see the local IP address).

If there is a `lazy` block present, the process itself will only be started when the first connection is opened. If the process takes some time to start up, the connection will be held for that time (the client will not notice any of this, except for the time the process needs to spin up). mittnite will wait for a duration of at most `.lazy.spinUpTimeout` for the process to accept connection; if this timeout is exceeded, the client connection will be closed.

When no connections have been active for a duration of at least `.lazy.coolDownTimeout`, mittnite will terminate the process again.

#### Boot Jobs

Boot jobs are "special" jobs that are executed before regular `job` definitions. Boot jobs are required to run to completion before any regular jobs are started.

```hcl
boot "setup" {
  command = "/bin/bash"
  args = ["/init-script.sh"]
  timeout = "30s"
  env = [
    "FOO=bar"
  ]
}
```

#### File

Possible directives to use in a file definition.

```hcl
file "/path/to/file.txt" {
  from = "examples/test.d/test.txt.tpl"
  params = {
    foo = "bar"
  }
}

file "/path/to/second_file.txt" {
  from = "examples/test.d/second_test.txt.tpl"
  overwrite = false
  params = {
    foo = "bar"
  }
}
```

Files are rendered using Golang's [text rendering engine][].
The rendering capabilities have been enhanced with
[sprig][] functions. The `env` function has been removed
as environment variables can be accessed via the *Env*
template variable.

```
some.param = {{ .Params.foo | default "bac" }}
some.env = {{ .Env.USER | upper }}
secret = {{ randAlphaNum 16 }}
```

[text rendering engine]: https://pkg.go.dev/text/template
[sprig]: https://github.com/Masterminds/sprig

#### Probe

Possible directives to use in a probe definition.

```hcl
probe "redis" {
  wait = true
  redis {
    host = {
      hostname = "localhost"
      port = 6379
    }
    password = "secret"
  }
}

probe "smtp" {
  wait = true
  smtp {
    host = {
      hostname = "localhost"
      port = 25
    }
  }
}

probe "mysql" {
  wait = true
  mysql {
    host = {
      hostname = "localhost"
      port = 3306
    }
    credentials = {
      user = "foo"
      password = "bar"
    }
  }
}
 
probe "amqp" { 
  wait = true
  amqp {
    host = {
      hostname = "localhost"
      port = 5672
    }
    credentials = {
      user = "foo"
      password = "bar"
    }
    virtualhost = "amqp.localhost.com"
  }
}
 
probe "mongodb" {
  wait = true 
  mongodb {
    url = "mongodb://localhost:27017/mongo"
  }
}
  
probe "http" {
  wait = true
  http {
    scheme = "http"
    host = {
        hostname = "localhost"
        port = 8080
    }
    path = "/status"
    timeout = "5s"
  }
}

probe "some-file" {
  wait = true
  filesystem = "/path/to/some/dir"
}
```

Specifying a `port` is optional and defaults to the services default port.

### HCL examples

#### Start a process

```hcl
job webserver {
  command = "/usr/bin/http-server"

  watch "/etc/conf.d/*.conf" {
    signal = 12 # USR2
  }
}
```

#### Start a process lazily on first request

```hcl
job webserver {
  command = "/usr/bin/http-server"
  args = ["-p8081", "-a127.0.0.1"]

  lazy {
    spinUpTimeout = "5s"
    coolDownTimeout = "15m"
  }

  listen "0.0.0.0:8080" {
    forward = "127.0.0.1:8081"
  }
}
```

#### Render a file on startup

```hcl
file "/etc/conf.d/test.txt" {
  from = "examples/test.d/test.txt.tpl"

  params = {
    foo = "bar"
  }
}
```

#### Wait until a Redis connection is possible

```hcl
probe redis {
  wait = true
  redis {
    host = {
      hostname = "localhost"
      port = 6379
    }
  }
}
```

### More examples
More example files can be found in the [examples directory](examples/)

## mittnitectl

`mittnitectl` can be used to control the mittnite process as long as the required API is enabled (`mittnite up --api`).
Currently, it is possible to _start_, _stop_, _restart_ jobs and display the current _status_.

```shell
$ mittnitectl --help
This command can be used to control mittnite by command line.

Usage:
  mittnitectl [command]

Available Commands:
  help        Help about any command
  job         Control a job via command line
  version     Show extended information about the current version of mittnite

Flags:
      --api-address string   write mittnites process id to this file (default "unix:///tmp/mittnite/mittnite.sock")
  -h, --help                 help for mittnitectl

Use "mittnitectl [command] --help" for more information about a command.
```
```shell
$ mittnitectl job --help
This command can be used to control a managed job.

Usage:
  mittnitectl job [command]

Available Commands:
  list        List jobs
  logs        Get logs from job
  restart     Restart a job
  start       Start a job
  status      Show job status
  stop        Stop a job

Flags:
  -h, --help   help for job

Global Flags:
      --api-address string   write mittnites process id to this file (default "unix:///var/run/mittnite.sock")

Use "mittnitectl job [command] --help" for more information about a command.
```


### job

To control a job, it must have set the `controllable` flag:

```hcl
job webserver {
  # ...
  controllable = true
  # ...
}
```

### Timestamp Formats

| Name        | Format                              |
|-------------|-------------------------------------|
| Layout      | 01/02 03:04:05PM '06 -0700          |
| ANSIC       | Mon Jan _2 15:04:05 2006            |                                             
| UnixDate    | Mon Jan _2 15:04:05 MST 2006        |                                         
| RubyDate    | Mon Jan 02 15:04:05 -0700 2006      |                                       
| RFC822      | 02 Jan 06 15:04 MST                 |                                                  
| RFC822Z     | 02 Jan 06 15:04 -0700               |                    
| RFC850      | Monday, 02-Jan-06 15:04:05 MST      |                                       
| RFC1123     | Mon, 02 Jan 2006 15:04:05 MST       |                                        
| RFC1123Z    | Mon, 02 Jan 2006 15:04:05 -0700     |         
| RFC3339     | 2006-01-02T15:04:05Z07:00           |                                            
| RFC3339Nano | 2006-01-02T15:04:05.999999999Z07:00 |                                  
| Kitchen     | 3:04PM                              |                                                               
| Stamp       | Jan _2 15:04:05                     |                                                       
| StampMilli  | Jan _2 15:04:05.000                 |                                                   
| StampMicro  | Jan _2 15:04:05.000000              |                                                
| StampNano   | Jan _2 15:04:05.000000000           |                                             
| DateTime    | 2006-01-02 15:04:05                 |                                                   
| DateOnly    | 2006-01-02                          |                                                            
| TimeOnly    | 15:04:05                            |         
