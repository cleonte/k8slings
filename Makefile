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
# K8SLINGS_SHARD=i/n runs every n-th exercise (CI uses four shards; locally, K8SLINGS_SHARD=0/4 make cluster-test
# checks a quarter in about ten minutes). Without it everything runs on one cluster: about 45 minutes with twenty
# fix exercises, so the timeout is 100m (it was 40m, then too short). -count=1 because a cached pass of a test that needs a live cluster proves nothing.
cluster-test:
	@go test -tags cluster -count=1 -p 1 -timeout 100m -v -run 'TestRealCluster|TestClusterSolutions|TestFixSolutions' ./k8slings/...
