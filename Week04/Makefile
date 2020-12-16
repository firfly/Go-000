GO:=${GOROOT}/bin/go

build:
	cd cmd/server && ${GO} build -o ../../bin/server && cd ../../

run:
	./bin/server

rpc:
	${GO} run test/user.go