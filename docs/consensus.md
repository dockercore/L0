模块初始化
	Init_Consensus()

向共识提交交易
	CommitTx(tx)

获取共识后的交易
	tx = GetCommitedTx()
	tx == nil表示区块分隔符