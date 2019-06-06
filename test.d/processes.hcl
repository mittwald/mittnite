file test.txt {
  from = "test.d/test.txt.tpl"

  params = {
    foo = "bar"
  }
}

job webserver {
  command = "/usr/bin/http-server"

  watch "./test.txt" {
    signal = 12 # USR2
  }
}

probe http {
  wait = true
  http {
    url = "http://google.de"
    timeout = "3s"
  }
}