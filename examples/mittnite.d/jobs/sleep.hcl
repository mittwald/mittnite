job sleep {
  command = "/bin/sleep"
  args = ["500"]
  canFail = true
  oneTime = false
  controllable = true
  maxAttempts = 3
}

job sleep2 {
   command = "/bin/sleep"
   args = ["500"]
   canFail = false
   oneTime = false
   controllable = false
}

#probe foo {
#    wait = true
#    filesystem = "/Users/mhelmich/Git/Github/mittnite/test"
#}

#boot hello {
#  command = "cowsay"
#  args = ["moo"]
#}
