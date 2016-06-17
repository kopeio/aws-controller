package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_prefix", "go_library")

go_prefix("github.com/jmespath/go-jmespath")

go_library(
    name = "go_default_library",
    srcs = glob(
      include = ["*.go"],
      exclude = ["*_test.go"],
    ),
)