#!/usr/bin/env python3

from pyteal import *


# def approval_program():
#     program = Return(Int(1))
#     return compileTeal(program, Mode.Application, version = 5)
# print(approval_program())


rax = ScratchVar(TealType.uint64)

on_add = Seq([
    rax.store(App.globalGet(Bytes("Count"))),
    App.globalPut(Bytes("Count"), rax.load() + Int(1)),
    Return(Int(1))
])

on_sub = Seq([
    rax.store(App.globalGet(Bytes("Count"))),
    If(rax.load() > Int(0),
       App.globalPut(Bytes("Count"), rax.load() - Int(1))),
    Return(Int(1))
])


on_create = Seq([
    App.globalPut(Bytes("Count"), Int(0)),
    Return(Int(1))
    ])

on_invoke = Cond(
    [And(
        Global.group_size() == Int(1),
        Txn.application_args[0] == Bytes("Add")
        ), on_add],
    [And(
        Global.group_size() == Int(1),
        Txn.application_args[0] == Bytes("Sub")
        ), on_sub]
)

program = Cond(
    [Txn.application_id() == Int(0),         on_create],
    [Txn.on_completion() == OnComplete.NoOp, on_invoke]
)

print (compileTeal(program, Mode.Application, version = 5))
