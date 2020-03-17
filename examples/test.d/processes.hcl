file "test.txt" {
  from = "examples/test.d/test.txt.tpl"

  params = {
    foo = "bar"
  }
}

job webserver {
  command = "/usr/local/bin/http-server"
  args = ["-p", "8080", "-a", "127.0.0.1"]

  watch "./test.txt" {
    signal = 12 # USR2
  }

//  lazy {
//    spinUpTimeout = "5s"
//    coolDownTimeout = "1m"
//  }

  listen "0.0.0.0:8081" {
    forward = "127.0.0.1:8080"
  }
}

probe http {
  wait = true
  http {
    host {
      hostname = "google.de"
    }
    timeout = "3s"
  }
}