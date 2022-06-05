#!/usr/bin/env python3

from pyteal import *

@Subroutine(TealType.uint64)
def sqrt(n):
    z = ScratchVar(TealType.uint64)
    y = ScratchVar(TealType.uint64)

    return Seq([
        z.store((n + Int(1)) / Int(2)),
        y.store(n),

        While(z.load() < y.load()).Do(Seq([
            y.store(z.load()),
            z.store((n / z.load() + z.load()) / Int(2))
        ])),

        Return(z.load())
    ])

@Subroutine(TealType.none)
def find():
    client_x = Int(5000)
    client_y = Int(5000)
    driver_x = ScratchVar(TealType.uint64)
    driver_y = ScratchVar(TealType.uint64)
    distance = ScratchVar(TealType.uint64)
    min_driver = ScratchVar(TealType.uint64)
    min_distance = ScratchVar(TealType.uint64)
    i = ScratchVar(TealType.uint64)

    return Seq([
        min_distance.store(Int(0)),

        For(i.store(Int(0)), i.load() < Int(10000),
            i.store(i.load() + Int(1))).Do(Seq([

            driver_x.store(App.globalGet(Bytes("Driver_pos"))),
            If(driver_x.load() < client_x)
                .Then(driver_x.store(client_x - driver_x.load()))
                .Else(driver_x.store(driver_x.load() - client_x)),
            driver_x.store(driver_x.load() * driver_x.load()),

            driver_y.store(App.globalGet(Bytes("Driver_pos"))),
            If(driver_y.load() < client_y)
                .Then(driver_y.store(client_y - driver_y.load()))
                .Else(driver_y.store(driver_y.load() - client_y)),
            driver_y.store(driver_y.load() * driver_y.load()),

            distance.store(sqrt(driver_x.load() + driver_y.load())),

            If(Or(i.load() == Int(0), distance.load() < min_distance.load()))
                .Then(Seq([
                    min_driver.store(i.load()),
                    min_distance.store(distance.load())
                ]))
        ]))
    ])

on_check_distance = Seq([
    find(),
    Return(Int(1))
])

on_create = Seq([
    App.globalPut(Bytes("Driver_pos"), Int(7500)),
    Return(Int(1))
])

on_invoke = Cond(
    [And(
        Global.group_size() == Int(1),
    ), on_check_distance]
)

program = Cond(
    [Txn.application_id() == Int(0), on_create],
    [Txn.on_completion() == OnComplete.NoOp, on_invoke]
)

print(compileTeal(program, Mode.Application, version=5))
