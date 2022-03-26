#!/bin/bash

set -e

algod_token=`cat algod.token`

echo "Starting indexer against algod."

for i in 1 2 3 4 5; do
  wget "algorand:4001"/genesis -O genesis.json && break
  echo "Algod not responding... waiting."
  sleep 15
done

if [ ! -f genesis.json ]; then
  echo "Failed to create genesis file!"
  exit 1
fi


./cmd/algorand-indexer/algorand-indexer daemon \
  -P "host=algorand port=5432 user=algorand password=indexer dbname=pgdb sslmode=disable" \
  --dev-mode \
  --server ":4003" \
  --algod-net "algorand:4001" \
  --algod-token "$algod_token" \
  --genesis "genesis.json"
