# Garden Integration Tests

Tests that run against a remote garden server.

## How to run

1. Set `GARDEN_ADDRESS` to the address of your running garden server. e.g. if you deployed garden-linux-release to bosh-lite, you would do:
    `export GARDEN_ADDRESS=10.244.16.6:7777`

1. Run the tests against the deployed garden.
    `ginkgo -p -nodes=4`
