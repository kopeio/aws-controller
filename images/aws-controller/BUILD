load("@bazel_tools//tools/build_defs/docker:docker.bzl", "docker_build")

docker_build(
    name = "aws-controller",
    base = "//images/jessie:dash",
    files = [ "//cmd/aws-controller", ],
    cmd = [ "/aws-controller" ],
)
