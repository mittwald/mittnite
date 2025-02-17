boot "oneshot" {
  command = "/bin/bash"
  args = [
    "examples/oneshots.d/job.sh"
  ]
}

boot "oneshot_two" {
  command = "/bin/bash"
  args = [
    "examples/oneshots.d/job2.sh"
  ]
}
