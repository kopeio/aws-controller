package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "aws-controller",
    srcs = glob(
            include = ["*.go"],
            exclude = ["*_test.go"],
    ),
    deps = [
      "//vendor/github.com/golang/glog:go_default_library",
      "//vendor/github.com/spf13/pflag:go_default_library",
      "//pkg:awscontroller/instances",
      "//pkg:kope",
      "//pkg:kope/kopeaws",
    ],
)
