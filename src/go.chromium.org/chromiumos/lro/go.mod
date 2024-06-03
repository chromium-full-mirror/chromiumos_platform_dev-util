module go.chromium.org/chromiumos/lro

go 1.21.5

// Replace version with latest and run `go mod tidy` to update.
replace go.chromium.org/chromiumos/config/go => go.chromium.org/chromiumos/config/go/src/go.chromium.org/chromiumos/config/go v0.0.0-20240531012445-8857f03a0531

require (
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	go.chromium.org/chromiumos/config/go v0.0.0-20240309015314-b8a183866804
	google.golang.org/grpc v1.64.0
)

require (
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240521202816-d264139d666e // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)
