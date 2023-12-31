module WorkflowInjector

go 1.15

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/gomodule/redigo v1.8.8
	github.com/kr/pretty v0.1.0 // indirect
	golang.org/x/net v0.0.0-20200520182314-0ba52f642ac2
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.24.0
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect

)

replace k8s.io/client-go v11.0.0+incompatible => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90 // indirect

replace k8s.io/apimachinery v0.18.3 => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
