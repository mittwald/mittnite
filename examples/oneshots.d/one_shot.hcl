boot "oneshot" {
  command = "/bin/bash"
  args = [
    "examples/bootjob.d/job.sh"
  ]
}

boot "oneshot_two" {
  command = "/bin/bash"
  args = [
    "examples/bootjob.d/job2.sh"
  ]
}
