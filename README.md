

# Contract design

1. Contract deployed
2. user opts-in
3. user calls `Commit(byteArray commitment)`
4. creator calls `SetDraw`:
```
[]Args{
  byteArray draw,
  int64 num_tickets_matching_1_number,
  int64 prize_pool_for_tickets_matching_1_number_in_microalgo,
  int64 num_tickets_matching_2_number,
  int64 prize_pool_for_tickets_matching_2_number_in_microalgo,
  int64 num_tickets_matching_3_number,
  int64 prize_pool_for_tickets_matching_3_number_in_microalgo,
  int64 num_tickets_matching_4_number,
  int64 prize_pool_for_tickets_matching_4_number_in_microalgo,
  int64 num_tickets_matching_5_number,
  int64 prize_pool_for_tickets_matching_5_number_in_microalgo,
  int64 num_tickets_matching_6_number,
  int64 prize_pool_for_tickets_matching_6_number_in_microalgo,
}

[]Accounts{
  rollover_destiation // Should be the address of the following lotto contract
}

```

  - Num. winning tickets and prize pools calculated by scanning history of transactions commiting to the contract (i.e. buying tickets)
  - Sends `total_escrow_balance*0.1` to creator address
  - Calculates rollover amount as sum of all the prize pools with `num_winning_tickets` set to zero
    - (TODO) It should fail if the sum of all prize pools is greater than the remaining escrow amount after the running costs have been removed.
  - Sends rollover amount to `rollover_destination`
  - (Stores number of winning tickets + prize pools to validate users claiming prizes and calc payout amounts)

5. user call `claim`


# Testing

```
tilt up
go test
```

Useful links spun up by algo indexer (inside tilt):

- http://localhost:4003/v2/accounts
- http://localhost:4003/v2/transactions
- http://localhost:4003/v2/applications
