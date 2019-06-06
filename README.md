# Mittnite - Smart init system for containers

Mittnite is a small, but smart init system designed for usage as `ENTRYPOINT` in container images.

It offers the following features:

- Render configuration files from templates using Go's [`text/template`](https://golang.org/pkg/text/template/) engine.
- Start processes and manage their lifecycle
- Watch configuration files and send configurable signals to processes on change
- Wait until required services are up before starting before starting processes (currently supporting filesystem mounts, HTTP services, MySQL, Redis, AMQP and MongoDB)

## Starting

Start as follows:

```
$ mittnite -config-dir /etc/mittnite.d
```

Or, use it in a container image:

```dockerfile
FROM quay.io/mittwald/mittnite:stable
COPY nginx.hcl /etc/mittnite.d/webserver.hcl
COPY fpm.hcl /etc/mittnite.d/fpm.hcl
CMD ["-config-dir", "/etc/mittnite.d"]
```

The directory specified with `-config-dir` can contain any number of `.hcl` configuration files; all files in that directory are loaded by Mittnite on startup and can contain any of the configuration directives described in the following section:

## Configuration directives

Start a process:

```hcl
job webserver {
  command = "/usr/bin/http-server"

  watch "./test.txt" {
    signal = 12 # USR2
  }
}
```

Render a file on startup:

```hcl
file test.txt {
  from = "test.d/test.txt.tpl"

  params = {
    foo = "bar"
  }
}
```

Wait until a Redis connection is possible:

```hcl
probe redis {
  wait = true
  redis {
    host = "localhost"
  }
}
```