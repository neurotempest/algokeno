from pyteal import *
from pyteal.ast.bytes import Bytes

def approval():

  global_owner = Bytes("owner") # byteslice
  global_num_tickets = Bytes("numTickets") # uint

  global_draw = Bytes("draw") # byteslice

  global_1_payouts_rem = Bytes("1s") # uint
  global_2_payouts_rem = Bytes("2s") # uint
  global_3_payouts_rem = Bytes("3s") # uint
  global_4_payouts_rem = Bytes("4s") # uint
  global_5_payouts_rem = Bytes("5s") # uint
  global_6_payouts_rem = Bytes("6s") # uint

  global_1_prize = Bytes("1p") # uint
  global_2_prize = Bytes("2p") # uint
  global_3_prize = Bytes("3p") # uint
  global_4_prize = Bytes("4p") # uint
  global_5_prize = Bytes("5p") # uint
  global_6_prize = Bytes("6p") # uint

  local_wager = Bytes("wager") # uint
  local_commitment = Bytes("commitment") # byteslice

  op_commit = Bytes("Commit")
  op_set_draw = Bytes("SetDraw")

  @Subroutine(TealType.none)
  def init():
    return Seq(
      App.globalPut(global_owner, Txn.sender()),
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

  @Subroutine(TealType.none)
  def set_draw():
    return Seq(
      Assert(
        And(
          Txn.sender() == App.globalGet(global_owner),
          Global.group_size() == Int(1),
          Txn.group_index() == Int(0),
          Gtxn[0].rekey_to() == Global.zero_address(),
          Txn.application_args.length() == Int(14),
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

if __name__ == "__main__":
  with open("approval.teal", "w") as f:
    compiled = compileTeal(approval(), mode=Mode.Application, version=MAX_TEAL_VERSION)
    f.write(compiled)

  with open("clear.teal", "w") as f:
    compiled = compileTeal(clear(), mode=Mode.Application, version=MAX_TEAL_VERSION)
    f.write(compiled)
