module go.chromium.org/chromiumos/test

go 1.22

// Replace version with latest and run `go mod tidy` to update.
replace (
	go.chromium.org/chromiumos/config/go => go.chromium.org/chromiumos/config/go/src/go.chromium.org/chromiumos/config/go v0.0.0-20240531012445-8857f03a0531
	go.chromium.org/chromiumos/lro => go.chromium.org/chromiumos/platform/dev-util/src/go.chromium.org/chromiumos/lro v0.0.0-20240603175638-a3a60d9d4bb5
	go.chromium.org/tast => go.chromium.org/tast/src/go.chromium.org/tast v0.0.0-20240530193934-ca093604b833
)

require (
	cloud.google.com/go/bigquery v1.61.0
	cloud.google.com/go/pubsub v1.38.0
	cloud.google.com/go/storage v1.41.0
	github.com/golang/glog v1.2.1
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.4
	github.com/google/go-cmp v0.6.0
	github.com/google/subcommands v1.2.0
	github.com/google/uuid v1.6.0
	github.com/maruel/subcommands v1.1.1
	github.com/mitchellh/hashstructure v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/goconvey v1.8.1
	github.com/stretchr/testify v1.9.0
	go.chromium.org/chromiumos/config/go v0.0.0-20240309015314-b8a183866804
	go.chromium.org/chromiumos/infra/proto/go v0.0.0-20240530000842-7d34be97f98c
	go.chromium.org/chromiumos/lro v0.0.0-00010101000000-000000000000
	go.chromium.org/luci v0.0.0-20240531181147-0c7c729b2fcf
	go.chromium.org/tast v0.0.0-00010101000000-000000000000
	go.starlark.net v0.0.0-20240520160348-046347dcd104
	golang.org/x/crypto v0.23.0
	golang.org/x/exp v0.0.0-20240531132922-fd00a4e0eefc
	golang.org/x/sync v0.7.0
	google.golang.org/api v0.182.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go v0.114.0 // indirect
	cloud.google.com/go/auth v0.4.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.2 // indirect
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	cloud.google.com/go/iam v1.1.8 // indirect
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/danjacques/gofslock v0.0.0-20240212154529-d899e02bfe22 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.4 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20240116145035-ef3ab179eed6 // indirect
	github.com/smarty/assertions v1.15.1 // indirect
	github.com/texttheater/golang-levenshtein v1.0.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/oauth2 v0.20.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/term v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.21.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto v0.0.0-20240401170217-c3f982113cda // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240513163218-0867130af1f8 // indirect
)
