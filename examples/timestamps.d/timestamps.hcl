job "echoloop" {
  command = "/bin/bash"
  args = [
    "-c",
    "while true ; do echo 'test'; sleep 10; done"
  ]

  stdout = "test.log"
  stderr = "test_error.log"
  enableTimestamps = true
}