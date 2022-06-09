from pyteal import *
from pyteal.ast.bytes import Bytes
import json

def approval():

  global_num_tickets = GlobalUint("numTickets")

  global_draw = GlobalByteslice("draw")

  global_1_payouts_rem = GlobalUint("1s")
  global_2_payouts_rem = GlobalUint("2s")
  global_3_payouts_rem = GlobalUint("3s")
  global_4_payouts_rem = GlobalUint("4s")
  global_5_payouts_rem = GlobalUint("5s")
  global_6_payouts_rem = GlobalUint("6s")

  global_1_prize = GlobalUint("1p")
  global_2_prize = GlobalUint("2p")
  global_3_prize = GlobalUint("3p")
  global_4_prize = GlobalUint("4p")
  global_5_prize = GlobalUint("5p")
  global_6_prize = GlobalUint("6p")

  global_next = GlobalByteslice("next")
  global_curr = GlobalByteslice("curr")

  local_wager = LocalUint("wager")
  local_commitment = LocalByteslice("commitment")


  op_commit = Bytes("Commit")
  op_set_draw = Bytes("SetDraw")
  op_claim = Bytes("Claim")

  @Subroutine(TealType.none)
  def init():
    return Seq(
      App.globalPut(global_num_tickets, Int(0)),
      App.globalPut(global_draw, Bytes("base64", "")),
      App.globalPut(global_1_payouts_rem, Int(0)),
      App.globalPut(global_2_payouts_rem, Int(0)),
      App.globalPut(global_3_payouts_rem, Int(0)),
      App.globalPut(global_4_payouts_rem, Int(0)),
      App.globalPut(global_5_payouts_rem, Int(0)),
      App.globalPut(global_6_payouts_rem, Int(0)),
      App.globalPut(global_1_prize, Int(0)),
      App.globalPut(global_2_prize, Int(0)),
      App.globalPut(global_3_prize, Int(0)),
      App.globalPut(global_4_prize, Int(0)),
      App.globalPut(global_5_prize, Int(0)),
      App.globalPut(global_6_prize, Int(0)),
      Approve(),
    )

  @Subroutine(TealType.none)
  def opt_in():
    return Seq(
      App.localPut(Int(0), local_wager, Int(0)),
      App.localPut(Int(0), local_commitment, Bytes("base64", "")),
      Approve(),
    )

  @Subroutine(TealType.none)
  def commit():
    return Seq(
      Assert(
        And(
          Global.group_size() == Int(2),
          Txn.group_index() == Int(0),
          *[Gtxn[i].rekey_to() == Global.zero_address() for i in range(2)],

          # second transaction is wager payment
          Gtxn[1].type_enum() == TxnType.Payment,
          Gtxn[1].receiver() == Global.current_application_address(),
          Gtxn[1].close_remainder_to() == Global.zero_address(),

          Txn.application_args.length() == Int(2),
        ),
      ),
      App.localPut(Txn.sender(), local_wager, Gtxn[1].amount()),
      App.localPut(Txn.sender(), local_commitment, Txn.application_args[1]),
      App.globalPut(
        global_num_tickets,
        App.globalGet(global_num_tickets) + Int(1),
      ),
      Approve(),
    )

  @Subroutine(TealType.uint64)
  def rollover_amount():
    rollover_amt = ScratchVar()
    return Seq(
      rollover_amt.store(Int(0)),
      If(Btoi(Txn.application_args[2]) == Int(0)).Then(rollover_amt.store(rollover_amt.load() + Btoi(Txn.application_args[3]))),
      If(Btoi(Txn.application_args[4]) == Int(0)).Then(rollover_amt.store(rollover_amt.load() + Btoi(Txn.application_args[5]))),
      If(Btoi(Txn.application_args[6]) == Int(0)).Then(rollover_amt.store(rollover_amt.load() + Btoi(Txn.application_args[7]))),
      If(Btoi(Txn.application_args[8]) == Int(0)).Then(rollover_amt.store(rollover_amt.load() + Btoi(Txn.application_args[9]))),
      If(Btoi(Txn.application_args[10]) == Int(0)).Then(rollover_amt.store(rollover_amt.load() + Btoi(Txn.application_args[11]))),
      If(Btoi(Txn.application_args[12]) == Int(0)).Then(rollover_amt.store(rollover_amt.load() + Btoi(Txn.application_args[13]))),
      Return(rollover_amt.load()),
    )

  @Subroutine(TealType.none)
  def set_draw():

    escrow_bal = AccountParam.balance(Global.current_application_address())
    next_app_address = AppParam.address(Int(1))
    running_costs = ScratchVar()
    ro_amount = ScratchVar()

    return Seq(
      next_app_address,
      Assert(
        And(
          Txn.sender() == Global.creator_address(),
          Global.group_size() == Int(1),
          Txn.group_index() == Int(0),
          Gtxn[0].rekey_to() == Global.zero_address(),
          Txn.application_args.length() == Int(14),

          Txn.fee() >= Global.min_txn_fee() * Int(3),
          Txn.applications.length() == Int(1),
          Txn.applications[1] != Txn.application_id(),

          Txn.accounts[Int(1)] == next_app_address.value(),
        ),
      ),
      App.globalPut(global_draw, Txn.application_args[1]),
      App.globalPut(global_1_payouts_rem, Btoi(Txn.application_args[2])),
      App.globalPut(global_1_prize, Btoi(Txn.application_args[3])),
      App.globalPut(global_2_payouts_rem, Btoi(Txn.application_args[4])),
      App.globalPut(global_2_prize, Btoi(Txn.application_args[5])),
      App.globalPut(global_3_payouts_rem, Btoi(Txn.application_args[6])),
      App.globalPut(global_3_prize, Btoi(Txn.application_args[7])),
      App.globalPut(global_4_payouts_rem, Btoi(Txn.application_args[8])),
      App.globalPut(global_4_prize, Btoi(Txn.application_args[9])),
      App.globalPut(global_5_payouts_rem, Btoi(Txn.application_args[10])),
      App.globalPut(global_5_prize, Btoi(Txn.application_args[11])),
      App.globalPut(global_6_payouts_rem, Btoi(Txn.application_args[12])),
      App.globalPut(global_6_prize, Btoi(Txn.application_args[13])),

      escrow_bal,
      running_costs.store(
        escrow_bal.value() / Int(10),
      ),

      ro_amount.store(rollover_amount()),

      InnerTxnBuilder.Begin(),
      InnerTxnBuilder.SetFields(
        {
          TxnField.type_enum: TxnType.Payment,
          TxnField.receiver: Global.creator_address(),
          TxnField.amount: running_costs.load(),
          TxnField.fee: Int(0),
        }
      ),
      If(ro_amount.load() > Int(100000))
      .Then(
        Seq(
          InnerTxnBuilder.Next(),
          InnerTxnBuilder.SetFields(
            {
              TxnField.type_enum: TxnType.Payment,
              TxnField.receiver: Txn.accounts[Int(1)],
              #TxnField.amount: Int(200000),
              TxnField.amount: ro_amount.load(),
              TxnField.fee: Int(0),
            }
          ),
        ),
      ),
      InnerTxnBuilder.Submit(),

      Approve(),
    )

  @Subroutine(TealType.none)
  def claim():
    return Seq(
      Assert(
        And(
          Global.group_size() == Int(1),
          Txn.group_index() == Int(0),
          Gtxn[0].rekey_to() == Global.zero_address(),

          Txn.fee() >= Global.min_txn_fee() * Int(2),

          App.localGet(Int(0), local_wager) >= Int(1000000),
          App.localGet(Int(0), local_commitment) != Bytes(""),

          App.localGet(Int(0), local_commitment) == App.globalGet(global_draw),
        ),
      ),

      # TODO: Check how many number match and select prize amount based on this
      InnerTxnBuilder.Begin(),
      InnerTxnBuilder.SetFields(
        {
          TxnField.type_enum: TxnType.Payment,
          TxnField.receiver: Txn.accounts[Int(0)],
          TxnField.amount: App.globalGet(global_6_prize),
          TxnField.fee: Int(0),
        }
      ),
      InnerTxnBuilder.Submit(),

      Approve(),
    )


  return program(
    init=Seq(
      init(),
      Approve(),
    ),
    opt_in=Seq(
      opt_in(),
      Approve(),
    ),
    no_op=Seq(
      Cond(
        [
          Txn.application_args[0] == op_commit,
          commit(),
        ],
        [
          Txn.application_args[0] == op_set_draw,
          set_draw(),
        ],
        [
          Txn.application_args[0] == op_claim,
          claim(),
        ],
      ),
      Reject()
    ),
  )

def clear():
  return Approve()

def program(
    init: Expr = Reject(),
    delete: Expr = Reject(),
    update: Expr = Reject(),
    opt_in: Expr = Reject(),
    close_out: Expr = Reject(),
    no_op: Expr = Reject(),
) -> Expr:
    return Cond(
        [Txn.application_id() == Int(0), init],
        [Txn.on_completion() == OnComplete.DeleteApplication, delete],
        [Txn.on_completion() == OnComplete.UpdateApplication, update],
        [Txn.on_completion() == OnComplete.OptIn, opt_in],
        [Txn.on_completion() == OnComplete.CloseOut, close_out],
        [Txn.on_completion() == OnComplete.NoOp, no_op],
    )

numGlobalByteslices = 0
def GlobalByteslice(name: str) -> Bytes:
  global numGlobalByteslices
  numGlobalByteslices += 1
  return Bytes(name)

numGlobalUints = 0
def GlobalUint(name: str) -> Bytes:
  global numGlobalUints
  numGlobalUints += 1
  return Bytes(name)

numLocalByteslices = 0
def LocalByteslice(name: str) -> Bytes:
  global numLocalByteslices
  numLocalByteslices += 1
  return Bytes(name)

numLocalUints = 0
def LocalUint(name: str) -> Bytes:
  global numLocalUints
  numLocalUints += 1
  return Bytes(name)

if __name__ == "__main__":

  with open("approval.teal", "w") as f:
    compiled = compileTeal(approval(), mode=Mode.Application, version=MAX_TEAL_VERSION)
    f.write(compiled)

  with open("clear.teal", "w") as f:
    compiled = compileTeal(clear(), mode=Mode.Application, version=MAX_TEAL_VERSION)
    f.write(compiled)

  with open("schema.json", "w") as f:
    schemad = {
      "global_byte_slices": numGlobalByteslices,
      "global_uints": numGlobalUints,
      "local_byte_slices": numLocalByteslices,
      "local_uints": numLocalUints,
    }
    schema = json.dumps(schemad)
    f.write(schema)

