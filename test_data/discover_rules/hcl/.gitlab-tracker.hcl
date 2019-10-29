checks {
    retry {
        maximum = 10
        interval_seconds = 2
    }
    pre_flight_command = [
        "argocd",
        "cluster",
        "list"
    ]
}

rules = [
    {
        path = "prepare-environment.sh"
        tag = "latest"
    },
    {
        path = "prepare-environment.sh"
        tag = "latest"
        tag_suffix_file_ref {
            file = "filename"
            regexp = "re"
        }
    }
]
