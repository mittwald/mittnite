probe redis {
  wait = true
  redis {
    host = {
      url = "localhost"
      port = 6379
    }
  }
}