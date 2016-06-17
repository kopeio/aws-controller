package(default_visibility = ["//visibility:public"])

load("@bazel_tools//tools/build_defs/docker:docker.bzl", "docker_build")

docker_build(
    name = "minimal",
    debs = [ "@pkg_coreutils//file",
             "@pkg_libacl1//file",
             "@pkg_libattr1//file",
             "@pkg_libc6//file",
             "@pkg_libselinux1//file",
             "@pkg_libpcre3//file",
             "@pkg_libgcc1//file",
             "@pkg_gcc_4_9_base//file"
    ],
    cmd = [ "/bin/ls", "/" ],
)

docker_build(
    name = "dash",
    base = "minimal",
    debs = [ "@pkg_dash//file",
    ],
    cmd = [ "/bin/ls", "/" ],
)
