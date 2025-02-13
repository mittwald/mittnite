boot "oneshotzero" {
  command = "/bin/bash"
  args = [
    "examples/bootjob.d/job2.sh"
  ]
}

boot "oneshot" {
  command = "/bin/bash"
  args = [
    "examples/bootjob.d/job.sh"
  ]
}

job "memes" {
  command = "/bin/bash"
  args = [
    "-c",
    "while true ; do echo 'memes'; sleep 10; done"
  ]

  stdout = "memes.log"
  stdout = "memes_error.log"
  enableTimestamps = true
}