# gen: protoc -I proto --go_out=. --go-grpc_out=.  proto/*.proto
gen: protoc -I proto  --gogofaster_out=plugins=grpc:.  proto/*.proto