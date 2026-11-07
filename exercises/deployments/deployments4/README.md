# deployments4: a deadline shorter than the wait

`minReadySeconds` is how long a new Pod has to stay Ready before the Deployment counts it as **available**. `progressDeadlineSeconds` is how long a rollout may go without progress before it is reported as failed. A deadline that is not longer than the wait can never be met, and the API server rejects the Deployment.

Keep the wait of 30 seconds (the application needs it to warm up) and make the Deployment valid.
