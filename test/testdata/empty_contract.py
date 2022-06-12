from pyteal import *

def approval():
  return Approve()

def clear():
  return Approve()

if __name__ == "__main__":
  with open("empty_approval.teal", "w") as f:
    compiled = compileTeal(approval(), mode=Mode.Application, version=MAX_TEAL_VERSION)
    f.write(compiled)

  with open("empty_clear.teal", "w") as f:
    compiled = compileTeal(clear(), mode=Mode.Application, version=MAX_TEAL_VERSION)
    f.write(compiled)
