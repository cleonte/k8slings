test:
	@go test -coverprofile=coverage.out -v ./...

watch:
	@go run ./k8slings watch

run:
	@go run ./k8slings run next

# Installs k8slings into a temporary folder and runs the first commands a learner types.
smoke:
	@scripts/smoke.sh
# Needs Docker, kind and kubectl. The packages run one after the other (-p 1), and
# K8SLINGS_SHARD=i/n runs every n-th exercise (CI uses three shards). -count=1 because a cached pass of a test that needs a live cluster proves nothing.
cluster-test:
	@go test -tags cluster -count=1 -p 1 -timeout 40m -v -run 'TestRealCluster|TestClusterSolutions|TestFixSolutions' ./k8slings/...
