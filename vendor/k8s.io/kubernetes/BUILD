package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_prefix", "go_library")

go_prefix("k8s.io/kubernetes")

go_library(
    name = "pkg/util/runtime",
    srcs = glob(
      include = ["pkg/util/runtime/*.go"],
      exclude = ["pkg/util/runtime/*_test.go"],
    ),
    deps = [
      "//vendor/github.com/golang/glog:go_default_library",
    ],
)

go_library(
    name = "pkg/util/wait",
    srcs = glob(
      include = ["pkg/util/wait/*.go"],
      exclude = ["pkg/util/wait/*_test.go"],
    ),
    deps = [
      ":pkg/util/runtime",
      "//vendor/github.com/golang/glog:go_default_library",
    ],
)
