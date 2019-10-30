rules "foo" {
  path = "prepare-environment.sh"
  label = "latest"
}

rules "bar" {
  path = "prepare-environment.sh"
  label = "latest"
  tagSuffixFileRef {
    file = "filename"
    regexp = "re"
  }
}
