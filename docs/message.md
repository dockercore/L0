### 消息(msg)

字段          |       类型     | 备注 
----         |      ---      | ---
chainId      |      []byte   | 预留
payload      |      []byte   | 具体消息
cmd          |      uint8    | 消息类型
hash         |      Hash     | 哈希
sign         |      []byte   | 签名

### 消息类型(cmd uint8)

字段          | 数值     | 备注 
----         | ---     | ---
PingMsg      |  iota   | 心跳
TongMsg      |         | 心跳
Handshake    |         | 握手请求
HandshakeAck |         | 握手确认
GetPeerMsg   |         | 请求节点信息
PeerMsg      |         | 返回节点信息
GetBlockMsg  |         | 获取区块信息
BlockMsg     |         | 返回区块信息




### 消息分类
 - 握手消息(handshake)
   -  一次握手
   -  二次握手
   - ...
 - 广播(bostcast)
     - 交易信息(transaction)
        - 交易发出(ask)
        - 交易确认(ack)
     - 区块(block)
        - 区块详情
        - 区块打包
        - ...
     - ...
 - 节点同步(sync)
    - 区块同步
    - 数据同步
    - ...