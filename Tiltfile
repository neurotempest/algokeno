# -*- mode: Python -*-

load('ext://restart_process', 'docker_build_with_restart')

docker_build(
  ref='algo_testnet',
  context='./algorand',
  dockerfile='deployments/Dockerfile.algo',
)

docker_build(
  ref='algo_indexer',
  context='./algorand',
  dockerfile='deployments/Dockerfile.algo_indexer',
)

k8s_yaml('deployments/algo_testnet_k8s.yaml')

k8s_resource(
  'algorand',
  port_forwards = [
      port_forward(4001, name = "Algorand RPC", host = 'localhost'),
      port_forward(4002, name = "Algorand KMD RPC", host = 'localhost'),
      port_forward(4003, name = "Algorand Indexer", host = 'localhost'),
  ],
)

