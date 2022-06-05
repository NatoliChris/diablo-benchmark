#!/usr/bin/env python3

from pyteal import *

# def approval_program():
#     program = Return(Int(1))
#     return compileTeal(program, Mode.Application, version = 5)
# print(approval_program())

j = ScratchVar(TealType.uint64)

x_position1 = ScratchVar(TealType.uint64)
x_position2 = ScratchVar(TealType.uint64)
x_position3 = ScratchVar(TealType.uint64)
x_position4 = ScratchVar(TealType.uint64)
x_position5 = ScratchVar(TealType.uint64)
x_position6 = ScratchVar(TealType.uint64)
x_position7 = ScratchVar(TealType.uint64)
x_position8 = ScratchVar(TealType.uint64)
x_position9 = ScratchVar(TealType.uint64)
x_position10 = ScratchVar(TealType.uint64)

y_position1 = ScratchVar(TealType.uint64)
y_position2 = ScratchVar(TealType.uint64)
y_position3 = ScratchVar(TealType.uint64)
y_position4 = ScratchVar(TealType.uint64)
y_position5 = ScratchVar(TealType.uint64)
y_position6 = ScratchVar(TealType.uint64)
y_position7 = ScratchVar(TealType.uint64)
y_position8 = ScratchVar(TealType.uint64)
y_position9 = ScratchVar(TealType.uint64)
y_position10 = ScratchVar(TealType.uint64)


i = ScratchVar(TealType.uint64)

on_update = Seq([
	    x_position1.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position1.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position1.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position1.load() + Int(1)),

	    x_position2.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position2.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position2.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position2.load() + Int(1)),

	    x_position3.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position3.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position3.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position3.load() + Int(1)),

	    x_position4.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position4.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position4.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position4.load() + Int(1)),

	    x_position5.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position5.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position5.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position5.load() + Int(1)),

	    x_position6.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position6.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position6.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position6.load() + Int(1)),

	    x_position7.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position7.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position7.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position7.load() + Int(1)),

	    x_position8.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position8.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position8.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position8.load() + Int(1)),

	    x_position9.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position1.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position9.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position9.load() + Int(1)),

	    x_position10.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(x_position10.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), x_position10.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), x_position10.load() + Int(1)),

	    y_position1.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position1.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position1.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position1.load() + Int(1)),

	    y_position2.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position2.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position2.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position2.load() + Int(1)),

	    y_position3.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position3.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position3.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position3.load() + Int(1)),

	    y_position4.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position4.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position4.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position4.load() + Int(1)),

	    y_position5.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position5.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position5.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position5.load() + Int(1)),

	    y_position6.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position6.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position6.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position6.load() + Int(1)),

	    y_position7.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position7.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position7.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position7.load() + Int(1)),

	    y_position8.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position8.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position8.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position8.load() + Int(1)),

	    y_position9.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position1.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position9.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position9.load() + Int(1)),

	    y_position10.store(App.globalGet(Bytes("X_Y_Position"))),
	    If(y_position10.load() > Int(250),App.globalPut(Bytes("X_Y_Position"), y_position10.load() - Int(249))),
	    App.globalPut(Bytes("X_Y_Position"), y_position10.load() + Int(1)),

	Return(Int(1))
])

on_create = Seq([
    App.globalPut(Bytes("X_Y_Position"), Int(1)),
    Return(Int(1))
    ])


on_invoke = Cond(
    [And(
        Global.group_size() == Int(1),
        Txn.application_args[0] == Bytes("update")
        ), on_update],
)

program = Cond(
    [Txn.application_id() == Int(0), on_create],
    [Txn.on_completion() == OnComplete.NoOp, on_invoke]
)

print(compileTeal(program, Mode.Application, version=5))
