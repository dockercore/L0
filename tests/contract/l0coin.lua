-- 用合约来完成一个数字货币系统
local L0 = require("L0")

-- 合约创建时会被调用一次，之后就不会被调用
function L0Init(func, args)
    print("in L0Init")
    L0.PutState("minter", L0.Account().Address)
    L0.PutState("balances", {})
    return true, "ok"
end

-- 每次合约执行都调用
function L0Invoke(func, args)
    print("in L0Invoke")
    local receiver = args[1]
    local amount = tonumber(args[2])
    
    if ("mint" == func) then
        mint(receiver, amount)
    elseif("send" == func) then
        send(receiver, amount)
    elseif("transfer" == func) then
        transfer(receiver, amount)
    end

    return true, "ok"
end

-- 查询
function L0Query(func,args)
    print("in L0Query",func,args)
    return true, "L0query ok"
end

function mint(receiver, amount)
    local sender = L0.Account().Address
    local minter = L0.GetState("minter")
    local balances = L0.GetState("balances")

    if (minter ~= sender) then
        return
    end

    balances[receiver] = balances[receiver] + amount
    L0.PutState("balances", balances)
end

function send(receiver, amount)
    local sender = L0.Account().Address
    local balances = L0.GetState("balances")

    print("sender: ",sender)
    print("sender balance: ",balances[sender])
    print("receiver balance: ",balances[receiver])
    print("amount: ",amount)

    if (balances[sender] < amount) then
        return
    end

    balances[sender] = balances[sender] - amount
    balances[receiver] = balances[receiver] + amount

    L0.PutState("balances", balances)
end

function transfer(receiver, amount)
    print("do transfer print by lua")
    L0.Transfer(receiver, amount)
end