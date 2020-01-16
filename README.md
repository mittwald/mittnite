[![Build Status](https://travis-ci.com/mittwald/mittnite.svg?branch=master)](https://travis-ci.com/mittwald/mittnite)

# Mittnite - Smart init system for containers

Mittnite is a small, but smart init system designed for usage as `ENTRYPOINT` in container images.

It offers the following features:

- Render configuration files from templates using Go's [`text/template`](https://golang.org/pkg/text/template/) engine.
- Start processes and manage their lifecycle
- Watch configuration files and send configurable signals to processes on change
- Wait until required services are up before starting processes (currently supporting filesystem mounts, HTTP services, MySQL, Redis, AMQP and MongoDB)

## Table of contents
  * [Table of contents](#table-of-contents)
  * [Getting started](#getting-started)
    + [CLI usage](#cli-usage)
      - [Basic](#basic)
      - [Render templates and execute custom command](#render-templates-and-execute-custom-command)
    + [Docker](#docker)
      - [Build your (go) application on top of the `mittnite` docker-image](#build-your--go--application-on-top-of-the--mittnite--docker-image)
      - [Download `mittnite` in your own custom `Dockerfile`](#download--mittnite--in-your-own-custom--dockerfile-)
  * [Configuration](#configuration)
    + [Directives](#directives)
      - [Job](#job)
      - [File](#file)
      - [Probe](#probe)
    + [HCL examples](#hcl-examples)
      - [Start a process](#start-a-process)
      - [Render a file on startup](#render-a-file-on-startup)
      - [Wait until a Redis connection is possible](#wait-until-a-redis-connection-is-possible)
    + [More examples](#more-examples)

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
                | tar xvz mittnite -C /usr/bin && \
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
Possible directives to use in a job definition.

```hcl
job "foo" {
  command = "/usr/local/bin/foo"
  args = "bar"
  max_attempts = 3
  canFail = false
  
  watch "/etc/conf.d/barfoo" {
    signal = 12
  }
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
```

#### Probe
Possible directives to use in a probe definition.
```hcl
probe "probe-name" {
  wait = true

  redis {
    host = {
      hostname = "localhost"
      port = 6379
    }
    password = ""
  }
  
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
  
  mongodb {
    host = {
      hostname = "localhost"
      port = 27017
    }
    credentials = {
      user = "foo"
      password = "bar"
    }    
    database = "mongo"
  }
  
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
```

Specifying a `port` is optional and defaults to the services default port.

### HCL examples
#### Start a process

```hcl
job webserver {
  command = "/usr/bin/http-server"

  watch "/etc/conf.d/test.txt" {
    signal = 12 # USR2
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
