# Garden Integration Tests

**Note**: This repository should be imported as `code.cloudfoundry.org/garden-integration-tests`.

Tests that run against a remote garden server.

## How to run

1. Set `GARDEN_ADDRESS` and `GARDEN_PORT` to the address/port of your running garden server.

```
export GARDEN_ADDRESS=10.244.0.2
export GARDEN_PORT=7777
```

1. Run the tests against the deployed garden.

```
ginkgo -p -nodes=4
```
