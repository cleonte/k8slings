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
# K8SLINGS_SHARD=i/n runs the exercises whose name hashes to i (CI uses five shards; locally, K8SLINGS_SHARD=0/5 make cluster-proofs-test
# checks a fifth in about ten minutes). Without it everything runs on one cluster: about 45 minutes with twenty
# fix exercises, so the timeout is 100m (it was 40m, then too short). -count=1 because a cached pass of a test that needs a live cluster proves nothing.
cluster-test:
	@go test -tags cluster -count=1 -p 1 -timeout 100m -v -run 'TestRealCluster|TestClusterSolutions|TestFixSolutions' ./k8slings/...
# What CI runs: the exercise proofs only (sharded with K8SLINGS_SHARD), and the safety rules on a cluster of their own.
cluster-proofs-test:
	@go test -tags cluster -count=1 -p 1 -timeout 100m -v -run 'TestClusterSolutions|TestFixSolutions' ./k8slings/...
cluster-safety-test:
	@go test -tags cluster -count=1 -p 1 -timeout 30m -v -run 'TestRealCluster' ./k8slings/...
