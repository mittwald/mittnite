probe mysql {
  wait = true
  mysql {
    user = "test"
    password = "test"
    host {
      hostname = "localhost"
      port = 3306
    }
    database = "test"
  }
}

probe redis {
  wait = true
  redis {
    host {
      hostname = "localhost"
      port = 6379
    }
  }
}

probe mongodb {
  wait = true
  mongodb {
    host = "ENV:MONGODB_HOSTNAME"
    database = "ENV:MONGODB_DATABASE"
    user = "ENV:MONGODB_USERNAME"
    password = "ENV:MONGODB_PASSWORD"
  }
}

probe amqp {
  wait = true
  amqp {
    host = "ENV:AMQP_HOSTNAME"
    user = "ENV:AMQP_USERNAME"
    password = "ENV:AMQP_PASSWORD"
  }
}