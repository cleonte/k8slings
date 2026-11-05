test:
	@go test -coverprofile=coverage.out -v ./...

watch:
	@go run ./k8slings watch

run:
	@go run ./k8slings run next

# Installs k8slings into a temporary folder and runs the first commands a learner types.
smoke:
	@scripts/smoke.sh
cluster-test:
	@go test -tags cluster -run TestRealCluster -timeout 20m -v ./k8slings/cluster/
