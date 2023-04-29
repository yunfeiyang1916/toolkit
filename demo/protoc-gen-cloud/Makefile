p:
	protoc --proto_path=./proto ./proto/*.proto --cloud_out=./out --go_out=./out
is:
	go install .
all:
	make is
	make p