job sleep {
  command = "/bin/sleep"
  args = ["500s"]
  canFail = true
  oneTime = true
}

boot hello {
  command = "cowsay"
  args = ["moo"]
}