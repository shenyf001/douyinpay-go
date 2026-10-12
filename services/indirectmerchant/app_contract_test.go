package indirectmerchant

import (
	"context"
	"strings"
	"testing"
)

func TestMerchantChannelAndAppResponses(t *testing.T) {
	transport := &recordingTransport{body: `{"app_info":{"app_id":"app_1","app_type":"移动应用","bind_status":"WAIT_CONFIRM"},"app_info_list":[{"app_id":"app_1","app_type":"移动应用","bind_status":"FUTURE"}]}`}
	service := newTestService(t, transport)
	locator := MerchantLocator{OutIndirectMerchantID: "external", OutPaymentInstID: "institution"}
	_, err := service.ModifyMerchant(context.Background(), ModifyMerchantRequest{
		MerchantLocator: locator, OutChannelSpID: "channel", MerchantShortName: "商户",
	})
	if err != nil || !strings.Contains(transport.request.body, `"out_channel_sp_id":"channel"`) {
		t.Fatalf("missing modify channel: %v %s", err, transport.request.body)
	}
	bind, _, err := service.BindAppID(context.Background(), BindAppIDRequest{
		MerchantLocator: locator, OutChannelSpID: "channel", AppID: "app_1",
	})
	if err != nil || bind.AppInfo.AppID != "app_1" || bind.AppInfo.BindStatus != "WAIT_CONFIRM" {
		t.Fatalf("incorrect binding: %v %+v", err, bind)
	}
	apps, _, err := service.QueryAppID(context.Background(), QueryAppIDRequest{
		MerchantLocator: locator, OutChannelSpID: "channel", AppID: "app_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if transport.request.query.Get("out_channel_sp_id") != "channel" ||
		len(apps.AppInfoList) != 1 || apps.AppInfoList[0].BindStatus != "FUTURE" {
		t.Fatalf("incorrect app response or channel: %+v %v", apps, transport.request.query)
	}
}
