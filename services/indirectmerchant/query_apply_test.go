package indirectmerchant

import (
	"context"
	"testing"
)

func TestQueryApply(t *testing.T) {
	transport := &recordingTransport{body: `{"business_code":"602323_10000","applyment_id":"NIM2000002124775691","applyment_state":"APPLYMENT_STATE_REJECTED","audit_detail":[{"field":"id_card_copy","reject_reason":"请上传清晰图片","future_field":true}]}`}
	service := newTestService(t, transport)
	for _, req := range []QueryApplyRequest{{ApplymentID: "NIM2000002124775691"}, {BusinessCode: "602323_10000"}} {
		response, _, err := service.QueryApply(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if response.BusinessCode != "602323_10000" || response.ApplymentState != "APPLYMENT_STATE_REJECTED" {
			t.Fatalf("incorrect response: %+v", response)
		}
		if len(response.AuditDetail) != 1 {
			t.Fatalf("incorrect audit detail: %+v", response.AuditDetail)
		}
		if response.AuditDetail[0].Field != "id_card_copy" || response.AuditDetail[0].RejectReason != "请上传清晰图片" {
			t.Fatalf("incorrect audit fields: %+v", response.AuditDetail[0])
		}
	}
}
