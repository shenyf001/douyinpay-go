抖音支付Golang语言版本SDK说明
## 1. 总体说明
本SDK包含App支付和H5支付方式的SDK

## 2. 接口

参见源码目录共分为四部分。
1. 支付：主要包含了下单，关单，根据订单号查询，根据外部单号查询四个接口，代码路径：services/pay
2. 退款：主要包含了退款，根据外部单号查询退款订单两个接口，代码路径：services/refund
3. 回调：主要包含了通知的解析接口，代码路径：services/callback
4. 账单：主要包含了账单的接口，代码路径：services/bill
5. 间连商户：包含入驻申请、申请查询/撤销、商户查询/修改、AppID 绑定/查询和图片上传，代码路径：services/indirectmerchant

## 3. 间连商户回调

使用已有初始化流程创建 `cli` 后，构造 `indirectmerchant.ApiService` 调用间连商户接口；申请单状态通知通过 `ParseApplymentStatusNotify` 解析：

```go
service := &indirectmerchant.ApiService{
	Client: cli,
	SensitiveFields: &indirectmerchant.SensitiveFieldConfig{
		PlatformCertificate: platformCertificate,
		MerchantPrivateKey: merchantPrivateKey,
	},
}
_ = service

notifyRequest, content, err := indirectmerchant.ParseApplymentStatusNotify(ctx, handler, request)
if err != nil {
	// 拒绝本次通知。
	return
}

// 按 notifyRequest.ID 去重；未知 applyment_state 保留原值。
_ = content.ApplymentState
```

调用方不得记录解密后的回调内容或上传文件内容；应按通知 ID 去重，并在重复或乱序通知时调用查询接口校准状态。

请求和响应均提供完整强类型嵌套模型。创建申请的证件持有人姓名、证件号传明文，由 SDK 使用平台证书 RSA PKCS#1 v1.5 加密；查询商户响应验签后，使用商户私钥解密这两个字段。`DouyinpaySerial` 可由加密证书推导；显式传入时必须与证书一致。加密使用请求副本，不修改调用方对象。图片上传按官方参数表限制 5 MB；官方错误码表仍有 2 MB 旧描述，实际服务端限制需以返回结果为准。

接口的使用方法可以参考每处代码下的xxx_test.go的测试用例
