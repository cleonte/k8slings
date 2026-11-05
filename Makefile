test:
	@go test -coverprofile=coverage.out -v ./...

watch:
	@go run ./k8slings watch

run:
	@go run ./k8slings run next

# Installs k8slings into a temporary folder and runs the first commands a learner types.
smoke:
	@scripts/smoke.sh
# Needs Docker, kind and kubectl. Each test makes its own cluster, one after the other (-p 1).
cluster-test:
	@go test -tags cluster -p 1 -timeout 40m -v -run 'TestRealCluster|TestClusterSolutions' ./k8slings/...
