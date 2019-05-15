probe mysql {
  wait = true
  mysql {
    user = "test"
    password = "test"
    host = "localhost"
    database = "test"
  }
}

probe redis {
  wait = true
  redis {
    host = "localhost"
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