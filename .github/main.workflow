workflow "Test" {
  on = "push"
  resolves = ["Build"]
}

action "Build" {
  uses = "./Dockerfile"
}
