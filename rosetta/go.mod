module github.com/cosmos/cosmos-sdk/rosetta

go 1.15

require (
	github.com/coinbase/rosetta-sdk-go v0.6.10
	github.com/cosmos/cosmos-sdk v0.42.0-alpha1.0.20210406085225-a4ccc672efd5
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210309074719-68d13333faf2 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20210114201628-6edceaf6022f // indirect
	google.golang.org/grpc v1.36.1
	google.golang.org/protobuf v1.26.0
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
