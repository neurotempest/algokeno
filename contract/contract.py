from pyteal import *
from pyteal.ast.bytes import Bytes

def approval():

  global_owner = Bytes("owner") # byteslice

  local_wager = Bytes("wager") # uint
  local_numbers = Bytes("numbers") # byteslice

  op_buy_ticket = Bytes("BuyTicket")

  @Subroutine(TealType.none)
  def init():
    return Seq(
      App.globalPut(global_owner, Txn.sender()),
      Approve(),
    )

  @Subroutine(TealType.none)
  def opt_in():
    return Seq(
      App.localPut(Int(0), local_wager, Int(0)),
      App.localPut(Int(0), local_numbers, Bytes("base64", "")),
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
    )
  )

  #return Cond(
  #  [Txn.application_id() == Int(0), init()],
  #  [Txn.on_completion() == OnComplete.DeleteApplication, Reject()],
  #  [Txn.on_completion() == OnComplete.UpdateApplication, Reject()],
  #  [Txn.on_completion() == OnComplete.OptIn, opt_in()],
  #  [Txn.on_completion() == OnComplete.CloseOut, Reject()],
  #  [Txn.on_completion() == OnComplete.NoOp, Reject()],
  #)

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
