#!/usr/bin/env python3

from pyteal import *

# def approval_program():
#     program = Return(Int(1))
#     return compileTeal(program, Mode.Application, version = 5)
# print(approval_program())


stockPriceTemp = ScratchVar(TealType.uint64)

on_buy = Seq([
    stockPriceTemp.store(App.globalGet(Bytes("GOOG"))),
    If(stockPriceTemp.load() > Int(1),
       App.globalPut(Bytes("GOOG"), stockPriceTemp.load() - Int(1))),
    Return(Int(1))
])

on_create = Seq([
    App.globalPut(Bytes("GOOG"), Int(1000000)),
    Return(Int(1))
])

on_invoke = Cond(
    [And(
        Global.group_size() == Int(1),
    ), on_buy]
)

program = Cond(
    [Txn.application_id() == Int(0), on_create],
    [Txn.on_completion() == OnComplete.NoOp, on_invoke]
)

print(compileTeal(program, Mode.Application, version=5))
