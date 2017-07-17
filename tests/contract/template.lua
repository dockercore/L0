-- 用合约来完成 ***
local L0 = require("L0")

-- 合约创建时会被调用一次, 完成数据初始化
function L0Init(args)

    return true
end

-- 每次合约执行都调用
function L0Invoke(func, args)

    return true,"ok"
end

-- 每次合约查询都调用
function L0Query(args)

    return true,"query detail"
end