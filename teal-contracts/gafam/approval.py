#!/usr/bin/env python3

from pyteal import *

# def approval_program():
#     program = Return(Int(1))
#     return compileTeal(program, Mode.Application, version = 5)
# print(approval_program())


stockPriceTemp = ScratchVar(TealType.uint64)

on_buy_google = Seq([
    stockPriceTemp.store(App.globalGet(Bytes("GOOG"))),
    If(stockPriceTemp.load() > Int(1),
       App.globalPut(Bytes("GOOG"), stockPriceTemp.load() - Int(1))),
    Return(Int(1))
])

on_buy_apple = Seq([
    stockPriceTemp.store(App.globalGet(Bytes("APPL"))),
    If(stockPriceTemp.load() > Int(1),
       App.globalPut(Bytes("APPL"), stockPriceTemp.load() - Int(1))),
    Return(Int(1))
])

on_buy_facebook = Seq([
    stockPriceTemp.store(App.globalGet(Bytes("FB"))),
    If(stockPriceTemp.load() > Int(1),
       App.globalPut(Bytes("FB"), stockPriceTemp.load() - Int(1))),
    Return(Int(1))
])

on_buy_amazon = Seq([
    stockPriceTemp.store(App.globalGet(Bytes("AMZN"))),
    If(stockPriceTemp.load() > Int(1),
       App.globalPut(Bytes("AMZN"), stockPriceTemp.load() - Int(1))),
    Return(Int(1))
])

on_buy_microsoft = Seq([
    stockPriceTemp.store(App.globalGet(Bytes("MSFT"))),
    If(stockPriceTemp.load() > Int(1),
       App.globalPut(Bytes("MSFT"), stockPriceTemp.load() - Int(1))),
    Return(Int(1))
])


on_create = Seq([
    App.globalPut(Bytes("GOOG"), Int(1000000)),
    App.globalPut(Bytes("APPL"), Int(1000000)),
    App.globalPut(Bytes("FB"), Int(1000000)),
    App.globalPut(Bytes("AMZN"), Int(1000000)),
    App.globalPut(Bytes("MSFT"), Int(1000000)),
    Return(Int(1))
])

on_invoke = Cond(
    [And(
        Global.group_size() == Int(1),
        Txn.application_args[0] == Bytes("BuyGoogle")
    ), on_buy_google],
    [And(
        Global.group_size() == Int(1),
        Txn.application_args[0] == Bytes("BuyApple")
    ), on_buy_apple],
    [And(
        Global.group_size() == Int(1),
        Txn.application_args[0] == Bytes("BuyFacebook")
    ), on_buy_facebook],
    [And(
        Global.group_size() == Int(1),
        Txn.application_args[0] == Bytes("BuyAmazon")
    ), on_buy_amazon],
    [And(
        Global.group_size() == Int(1),
        Txn.application_args[0] == Bytes("BuyMicrosoft")
    ), on_buy_microsoft]
)

program = Cond(
    [Txn.application_id() == Int(0), on_create],
    [Txn.on_completion() == OnComplete.NoOp, on_invoke]
)

print(compileTeal(program, Mode.Application, version=5))
