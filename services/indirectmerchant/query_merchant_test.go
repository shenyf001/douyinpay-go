package indirectmerchant

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
)

func TestQueryMerchantDecryptsSensitiveFields(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	name, _ := encryptField("张三", &key.PublicKey)
	number, _ := encryptField("110101199001011234", &key.PublicKey)
	body := `{"out_channel_sp_id":"channel","business_code":"apply_1","subject_info":{"certificate_info":{"legal_person_name":"法人"}},"identification_info":{"identification_name":"` + name + `","identification_number":"` + number + `"},"business_info":{"mcc":"7300","special_industry_pics":["media"]}}`
	for _, scenario := range []string{"success", "missing_key", "wrong_key", "invalid_ciphertext"} {
		t.Run(scenario, func(t *testing.T) {
			transport := &recordingTransport{body: body}
			service := newTestService(t, transport)
			service.SensitiveFields = &SensitiveFieldConfig{MerchantPrivateKey: key}
			switch scenario {
			case "missing_key":
				service.SensitiveFields = nil
			case "wrong_key":
				other, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					t.Fatal(err)
				}
				service.SensitiveFields.MerchantPrivateKey = other
			case "invalid_ciphertext":
				transport.body = `{"identification_info":{"identification_name":"invalid!"}}`
			}
			response, result, err := service.QueryMerchant(context.Background(), QueryMerchantRequest{
				MerchantLocator: MerchantLocator{IndirectMerchantID: "merchant"},
			})
			if scenario != "success" {
				if err == nil || response != nil || result == nil {
					t.Fatalf("expected failed decryption with API result: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(response)
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatal(err)
			}
			info := parsed["identification_info"].(map[string]interface{})
			if info["identification_name"] != "张三" || info["identification_number"] != "110101199001011234" ||
				parsed["out_channel_sp_id"] != "channel" || parsed["business_code"] != "apply_1" {
				t.Fatalf("incomplete or encrypted response: %s", data)
			}
		})
	}
}
