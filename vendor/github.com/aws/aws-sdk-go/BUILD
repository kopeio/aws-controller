package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_prefix", "go_library")

go_prefix("github.com/aws/aws-sdk-go")

go_library(
    name = "go_default_library",
    srcs = glob(
      include = ["*.go"],
      exclude = ["*_test.go"],
    ),
    deps = [
      ":aws",
      ":aws/awserr",
      ":aws/awsutil",
    ],
)

go_library(
    name = "aws",
    srcs = glob(
      include = ["aws/*.go"],
      exclude = ["aws/*_test.go"],
    ),
    deps = [
      ":aws/credentials",
      ":aws/awserr",
    ],
)


go_library(
    name = "aws/awsutil",
    srcs = glob(
      include = ["aws/awsutil/*.go"],
      exclude = ["aws/awsutil/*_test.go"],
    ),
    deps = [
        "//vendor/github.com/jmespath/go-jmespath:go_default_library",
    ],
)


go_library(
    name = "aws/awserr",
    srcs = glob(
      include = ["aws/awserr/*.go"],
      exclude = ["aws/awserr/*_test.go"],
    ),
)


go_library(
    name = "aws/client",
    srcs = glob(
      include = ["aws/client/*.go"],
      exclude = ["aws/client/*_test.go"],
    ),
    deps = [
        ":aws",
        ":aws/client/metadata",
        ":aws/request",
    ],
)

go_library(
    name = "aws/client/metadata",
    srcs = glob(
      include = ["aws/client/metadata/*.go"],
      exclude = ["aws/client/metadata/*_test.go"],
    ),
    deps = [
        ":aws/awserr",
    ],
)

go_library(
    name = "aws/corehandlers",
    srcs = glob(
      include = ["aws/corehandlers/*.go"],
      exclude = ["aws/corehandlers/*_test.go"],
    ),
    deps = [
        ":aws",
        ":aws/awserr",
        ":aws/request",
    ],
)


go_library(
    name = "aws/credentials",
    srcs = glob(
      include = ["aws/credentials/*.go"],
      exclude = ["aws/credentials/*_test.go"],
    ),
    deps = [
        ":aws/awserr",
        "//vendor/github.com/go-ini/ini:go_default_library",
    ],
)


go_library(
    name = "aws/credentials/ec2rolecreds",
    srcs = glob(
      include = ["aws/credentials/ec2rolecreds/*.go"],
      exclude = ["aws/credentials/ec2rolecreds/*_test.go"],
    ),
    deps = [
        ":aws/awserr",
        ":aws/client",
        ":aws/credentials",
        ":aws/ec2metadata",
    ],
)


go_library(
    name = "aws/defaults",
    srcs = glob(
      include = ["aws/defaults/*.go"],
      exclude = ["aws/defaults/*_test.go"],
    ),
    deps = [
        ":aws",
        ":aws/corehandlers",
        ":aws/credentials",
        ":aws/credentials/ec2rolecreds",
        ":aws/ec2metadata",
        ":aws/request",
        ":private/endpoints",
    ],
)


go_library(
    name = "aws/ec2metadata",
    srcs = glob(
      include = ["aws/ec2metadata/*.go"],
      exclude = ["aws/ec2metadata/*_test.go"],
    ),
    deps = [
        ":aws",
        ":aws/awserr",
        ":aws/client",
        ":aws/client/metadata",
        ":aws/request",
    ],
)


go_library(
    name = "aws/request",
    srcs = glob(
      include = ["aws/request/*.go"],
      exclude = ["aws/request/*_test.go"],
    ),
    deps = [
      ":aws",
      ":aws/awserr",
      ":aws/awsutil",
      ":aws/client/metadata",
    ],
)


go_library(
    name = "aws/session",
    srcs = glob(
      include = ["aws/session/*.go"],
      exclude = ["aws/session/*_test.go"],
    ),
    deps = [
      ":aws",
      ":aws/client",
      ":aws/corehandlers",
      ":aws/defaults",
      ":aws/request",
      ":private/endpoints",
    ],
)

go_library(
    name = "private/endpoints",
    srcs = glob(
      include = ["private/endpoints/*.go"],
      exclude = ["private/endpoints/*_test.go"],
    ),
    deps = [
        ":aws",
    ],
)


go_library(
    name = "private/protocol/ec2query",
    srcs = glob(
      include = ["private/protocol/ec2query/*.go"],
      exclude = ["private/protocol/ec2query/*_test.go"],
    ),
    deps = [
        ":aws/awserr",
        ":aws/request",
        ":private/protocol/query/queryutil",
        ":private/protocol/xml/xmlutil",
    ],
)


go_library(
    name = "private/protocol/query",
    srcs = glob(
      include = ["private/protocol/query/*.go"],
      exclude = ["private/protocol/query/*_test.go"],
    ),
    deps = [
        ":aws/awserr",
        ":aws/request",
        ":private/protocol/query/queryutil",
        ":private/protocol/xml/xmlutil",
    ],
)

go_library(
    name = "private/protocol/query/queryutil",
    srcs = glob(
      include = ["private/protocol/query/queryutil/*.go"],
      exclude = ["private/protocol/query/queryutil/*_test.go"],
    ),
    deps = [
        ":aws/awserr",
    ],
)

go_library(
    name = "private/protocol/rest",
    srcs = glob(
      include = ["private/protocol/rest/*.go"],
      exclude = ["private/protocol/rest/*_test.go"],
    ),
    deps = [
        ":aws",
        ":aws/awserr",
        ":aws/request",
    ],
)


go_library(
    name = "private/protocol/restxml",
    srcs = glob(
      include = ["private/protocol/restxml/*.go"],
      exclude = ["private/protocol/restxml/*_test.go"],
    ),
    deps = [
        ":aws",
        ":aws/awserr",
        ":aws/request",
        ":private/protocol/query",
        ":private/protocol/rest",
        ":private/protocol/xml/xmlutil",
    ],
)


go_library(
    name = "private/protocol/xml/xmlutil",
    srcs = glob(
      include = ["private/protocol/xml/xmlutil/*.go"],
      exclude = ["private/protocol/xml/xmlutil/*_test.go"],
    ),
    deps = [
        ":aws/awserr",
    ],
)


go_library(
    name = "private/signer/v4",
    srcs = glob(
      include = ["private/signer/v4/*.go"],
      exclude = ["private/signer/v4/*_test.go"],
    ),
    deps = [
        ":aws",
        ":aws/credentials",
        ":aws/request",
        ":private/protocol/rest",
    ],
)


go_library(
    name = "private/waiter",
    srcs = glob(
      include = ["private/waiter/*.go"],
      exclude = ["private/waiter/*_test.go"],
    ),
    deps = [
      ":aws",
      ":aws/awserr",
      ":aws/awsutil",
      ":aws/request",
    ],
)


go_library(
    name = "service/ec2",
    srcs = glob(
      include = ["service/ec2/*.go"],
      exclude = ["service/ec2/*_test.go"],
    ),
    deps = [
        ":aws",
        ":aws/client",
        ":aws/client/metadata",
        ":aws/awsutil",
        ":aws/request",
        ":private/endpoints",
        ":private/signer/v4",
        ":private/protocol/ec2query",
        ":private/waiter",
    ],
)


go_library(
    name = "service/route53",
    srcs = glob(
      include = ["service/route53/*.go"],
      exclude = ["service/route53/*_test.go"],
    ),
    deps = [
        ":aws",
        ":aws/client",
        ":aws/client/metadata",
        ":aws/awsutil",
        ":aws/request",
        ":private/endpoints",
        ":private/signer/v4",
        ":private/protocol/restxml",
        ":private/waiter",
    ],
)

