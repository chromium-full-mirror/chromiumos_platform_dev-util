module go.chromium.org/chromiumos/ctp

go 1.24.0

require (
	github.com/golang/protobuf v1.5.4
	github.com/google/go-cmp v0.7.0
	go.chromium.org/chromiumos/config/go v0.0.0-20240309015314-b8a183866804
	go.chromium.org/chromiumos/infra/proto/go v0.0.0-20250623190020-bff867fc0474
	go.chromium.org/luci v0.0.0-20250918002626-41a4636920e8
	google.golang.org/genproto v0.0.0-20250818200422-3122310a409c
	google.golang.org/grpc v1.75.0
	google.golang.org/protobuf v1.36.8
)

require (
	cloud.google.com/go/compute/metadata v0.8.0 // indirect
	github.com/golang/mock v1.7.0-rc.1 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/term v0.34.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/api v0.248.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250818200422-3122310a409c // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250818200422-3122310a409c // indirect
)

replace go.chromium.org/chromiumos/config/go => go.chromium.org/chromiumos/config/go/src/go.chromium.org/chromiumos/config/go v0.0.0-20250623185307-69444630c628
