checks "pre_flight" {
    retry {
        maximum = 10
        interval_seconds = 2
    }
    command = [
        "argocd",
        "cluster",
        "list"
    ]
}

rules "foo" {
    path = "prepare-environment1.sh"
    tag = "latest"
}

rules "bar" {
    path = "prepare-environment2.sh"
    tag = "latest"
    tag_suffix_file_ref {
        file = "filename"
        regexp = "re"
        regexp_group = 10
    }
}
