load("//tools:defs.bzl", "go_library", "go_test", "proto_library")

package(
    default_applicable_licenses = ["//:license"],
    licenses = ["notice"],
)

proto_library(
    name = "api",
    srcs = [
        "runtimeoptions.proto",
    ],
)

go_library(
    name = "v14",
    srcs = [
        "runtimeoptions.go",
        "runtimeoptions_cri.go",
    ],
    visibility = ["//pkg/shim/v1/runsc:__pkg__"],
    deps = ["@com_github_gogo_protobuf//proto:go_default_library"],
)

go_test(
    name = "runtimeoptions_test",
    size = "small",
    srcs = ["runtimeoptions_test.go"],
    library = ":v14",
    deps = [
        "@com_github_containerd_containerd//runtime/v1/shim/v1:go_default_library",
        "@com_github_containerd_typeurl//:go_default_library",
        "@com_github_gogo_protobuf//proto:go_default_library",
    ],
)
