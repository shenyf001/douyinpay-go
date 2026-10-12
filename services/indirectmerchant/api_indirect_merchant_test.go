package indirectmerchant

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/douyinpay/douyinpay-go/client"
	"github.com/douyinpay/douyinpay-go/services/callback"
	"github.com/douyinpay/douyinpay-go/tools/auth"
	"github.com/douyinpay/douyinpay-go/tools/consts"
)

type recordedRequest struct {
	method string
	path   string
	query  url.Values
	header http.Header
	body   string
}

type recordingTransport struct {
	request recordedRequest
	body    string
}

func (t *recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	t.request = recordedRequest{
		method: r.Method, path: r.URL.Path, query: r.URL.Query(),
		header: r.Header.Clone(), body: string(body),
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			consts.DouyinPaySerial:    []string{"platform-serial"},
			consts.DouyinPaySignature: []string{"signature"},
			consts.DouyinPayTimestamp: []string{strconv.FormatInt(time.Now().Unix(), 10)},
			consts.DouyinPayNonce:     []string{"nonce"},
		},
		Body:    ioutil.NopCloser(strings.NewReader(t.body)),
		Request: r,
	}, nil
}

type testSigner struct{}

func (*testSigner) Sign(_ context.Context, _ string) (*auth.SignatureResult, error) {
	return &auth.SignatureResult{MchID: "mchid", CertificateSerialNo: "merchant-serial", Signature: "signature"}, nil
}
func (*testSigner) Algorithm() string { return "TEST" }

type testVerifier struct{}

func (*testVerifier) Verify(_ context.Context, _, _, _ string) error { return nil }
func (*testVerifier) GetSerial(_ context.Context) (string, error)    { return "platform-serial", nil }

func TestQueryMerchantUsesOneLocator(t *testing.T) {
	service := newTestService(t, new(recordingTransport))
	_, _, err := service.QueryMerchant(context.Background(), QueryMerchantRequest{})
	assert.Equal(t, true, err != nil)
	_, _, err = service.QueryMerchant(context.Background(), QueryMerchantRequest{
		MerchantLocator: MerchantLocator{
			IndirectMerchantID: "im_1", OutIndirectMerchantID: "out_1", OutPaymentInstID: "pi_1",
		},
	})
	assert.Equal(t, true, err != nil)
	_, _, err = service.QueryMerchant(context.Background(), QueryMerchantRequest{
		MerchantLocator: MerchantLocator{IndirectMerchantID: "im_1"},
		OutChannelSpID:  "channel_1",
	})
	assert.Equal(t, true, err != nil)
}

func TestIndirectMerchantRequestContracts(t *testing.T) {
	transport := &recordingTransport{body: `{"applyment_id":"apply_1","media_id":"media_1"}`}
	service := newTestService(t, transport)
	ctx := context.Background()

	_, _, err := service.CreateApply(ctx, CreateApplyRequest{
		BusinessCode: "apply_1", DouyinpaySerial: "platform-serial",
		OutIndirectMerchantID: "out_merchant_1", OutPaymentInstID: "payment_inst_1",
		OutChannelSpID: "channel_sp_1", OutPaymentInstSubID: "payment_inst_sub_1",
		IndirectMerchantID: "merchant_1",
	})
	assert.Equal(t, nil, err)
	assertRequest(t, transport.request, http.MethodPost, consts.CreateIndirectMerchantApplyPath, url.Values{}, "{\"business_code\":\"apply_1\",\"out_indirect_merchant_id\":\"out_merchant_1\",\"out_payment_inst_id\":\"payment_inst_1\",\"out_channel_sp_id\":\"channel_sp_1\",\"out_payment_inst_sub_id\":\"payment_inst_sub_1\",\"indirect_merchant_id\":\"merchant_1\"}\n")
	assert.Equal(t, "platform-serial", transport.request.header.Get(consts.DouyinpaySerial))

	_, _, err = service.QueryApply(ctx, QueryApplyRequest{ApplymentID: "apply_1"})
	assert.Equal(t, nil, err)
	assertRequest(t, transport.request, http.MethodGet, consts.QueryIndirectMerchantApplyPath, url.Values{"applyment_id": {"apply_1"}}, "")

	_, err = service.CancelApply(ctx, CancelApplyRequest{BusinessCode: "apply_1"})
	assert.Equal(t, nil, err)
	assertRequest(t, transport.request, http.MethodPost, consts.CancelIndirectMerchantApplyPath, url.Values{}, "{\"business_code\":\"apply_1\"}\n")

	externalLocator := MerchantLocator{OutIndirectMerchantID: "out_merchant_1", OutPaymentInstID: "payment_inst_1"}
	_, _, err = service.QueryMerchant(ctx, QueryMerchantRequest{
		MerchantLocator: externalLocator, OutChannelSpID: "channel_1",
	})
	assert.Equal(t, nil, err)
	assertRequest(t, transport.request, http.MethodGet, consts.QueryIndirectMerchantPath, url.Values{
		"out_indirect_merchant_id": {"out_merchant_1"}, "out_payment_inst_id": {"payment_inst_1"},
		"out_channel_sp_id": {"channel_1"},
	}, "")

	locator := MerchantLocator{IndirectMerchantID: "im_1"}
	_, err = service.ModifyMerchant(ctx, ModifyMerchantRequest{MerchantLocator: locator, MerchantShortName: "merchant", ServicePhone: "13800138000"})
	assert.Equal(t, nil, err)
	assertRequest(t, transport.request, http.MethodPost, consts.ModifyIndirectMerchantPath, url.Values{}, "{\"indirect_merchant_id\":\"im_1\",\"merchant_short_name\":\"merchant\",\"service_phone\":\"13800138000\"}\n")

	_, _, err = service.BindAppID(ctx, BindAppIDRequest{MerchantLocator: locator, AppID: "app_1"})
	assert.Equal(t, nil, err)
	assertRequest(t, transport.request, http.MethodPost, consts.BindIndirectMerchantAppIDPath, url.Values{}, "{\"indirect_merchant_id\":\"im_1\",\"app_id\":\"app_1\"}\n")

	_, _, err = service.QueryAppID(ctx, QueryAppIDRequest{MerchantLocator: locator, AppID: "app_1"})
	assert.Equal(t, nil, err)
	assertRequest(t, transport.request, http.MethodGet, consts.QueryIndirectMerchantAppIDPath, url.Values{
		"indirect_merchant_id": {"im_1"}, "app_id": {"app_1"},
	}, "")

	uploadResponse, _, err := service.Upload(ctx, UploadRequest{FileName: "license.jpg", ContentType: "image/jpeg", FileContent: []byte("image-content")})
	assert.Equal(t, nil, err)
	assert.Equal(t, "media_1", uploadResponse.MediaID)
	assert.Equal(t, http.MethodPost, transport.request.method)
	assert.Equal(t, consts.UploadMerchantMediaPath, transport.request.path)
	assert.Equal(t, consts.MultipartFormData, transport.request.header.Get(consts.ContentType)[:len(consts.MultipartFormData)])
	_, params, err := mime.ParseMediaType(transport.request.header.Get(consts.ContentType))
	assert.Equal(t, nil, err)
	reader := multipart.NewReader(strings.NewReader(transport.request.body), params["boundary"])
	metaPart, err := reader.NextPart()
	assert.Equal(t, nil, err)
	assert.Equal(t, "meta", metaPart.FormName())
	assert.Equal(t, consts.ApplicationJSON, metaPart.Header.Get("Content-Type"))
	metaBytes, err := ioutil.ReadAll(metaPart)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, strings.Contains(string(metaBytes), `"filename":"license.jpg"`))
	filePart, err := reader.NextPart()
	assert.Equal(t, nil, err)
	assert.Equal(t, "file", filePart.FormName())
	assert.Equal(t, "image/jpeg", filePart.Header.Get("Content-Type"))
}

func TestCreateApply(t *testing.T) {
	const fixture = `{
		"business_code":"602323_10000","out_indirect_merchant_id":"10434920032",
		"out_payment_inst_id":"41110000","out_channel_sp_id":"C023828488",
		"out_payment_inst_sub_id":"41110100","indirect_merchant_id":"2026045587123820",
		"subject_info":{"subject_type":"SUBJECT_TYPE_ENTERPRISE","certificate_info":{
			"cert_no":"91310000TEST123456","merchant_name":"Contract Shop","cert_copy":"license_media",
			"register_address":"123 Contract Road","legal_person_name":"Alice Zhang",
			"period_begin":"2019-08-01","period_end":"2039-08-01"}},
		"identification_info":{"identification_type":"IDENTIFICATION_TYPE_IDCARD",
			"identification_name":"Alice Zhang","identification_number":"110101199001011234",
			"period_begin":"2019-08-01","period_end":"2039-08-01",
			"identification_front_copy":"front_media","identification_back_copy":"back_media"},
		"business_info":{"merchant_short_name":"Contract Shop","service_phone":"18042002458",
			"mcc":"7300","special_industry_pics":["industry_1","industry_2"]},
		"addition_info":{"addition_pics":["addition_1","addition_2"],"addition_remark":"Contract materials"}
	}`
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(0xABC123), PublicKey: &key.PublicKey}
	serial := fmt.Sprintf("%X", cert.SerialNumber)
	newRequest := func(t *testing.T) CreateApplyRequest {
		t.Helper()
		var req CreateApplyRequest
		if err := json.Unmarshal([]byte(fixture), &req); err != nil {
			t.Fatal(err)
		}
		return req
	}
	decrypt := func(t *testing.T, ciphertext string) string {
		t.Helper()
		data, err := base64.StdEncoding.DecodeString(ciphertext)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != key.Size() {
			t.Fatalf("ciphertext length = %d, want %d", len(data), key.Size())
		}
		plain, err := rsa.DecryptPKCS1v15(rand.Reader, key, data)
		if err != nil {
			t.Fatal(err)
		}
		return string(plain)
	}
	for _, explicitSerial := range []bool{false, true} {
		t.Run(fmt.Sprintf("complete_contract_explicit_serial_%t", explicitSerial), func(t *testing.T) {
			transport := &recordingTransport{body: `{"applyment_id":"NIM2000002124775691"}`}
			service := newTestService(t, transport)
			service.SensitiveFields = &SensitiveFieldConfig{PlatformCertificate: cert}
			req := newRequest(t)
			if explicitSerial {
				req.DouyinpaySerial = serial
			}
			original, err := json.Marshal(req)
			if err != nil {
				t.Fatal(err)
			}
			var expected map[string]interface{}
			if err := json.Unmarshal([]byte(fixture), &expected); err != nil {
				t.Fatal(err)
			}
			for attempt := 0; attempt < 2; attempt++ {
				response, result, err := service.CreateApply(context.Background(), req)
				if err != nil {
					t.Fatal(err)
				}
				if response == nil || result == nil {
					t.Fatal("expected response and API result")
				}
				assert.Equal(t, "NIM2000002124775691", response.ApplymentID)
				assert.Equal(t, http.MethodPost, transport.request.method)
				assert.Equal(t, consts.CreateIndirectMerchantApplyPath, transport.request.path)
				assert.Equal(t, url.Values{}, transport.request.query)
				assert.Equal(t, serial, transport.request.header.Get(consts.DouyinpaySerial))
				assert.Equal(t, consts.ApplicationJSON, transport.request.header.Get(consts.ContentType))
				var body map[string]interface{}
				if err := json.Unmarshal([]byte(transport.request.body), &body); err != nil {
					t.Fatal(err)
				}
				identification, ok := body["identification_info"].(map[string]interface{})
				if !ok {
					t.Fatal("identification_info must be an object")
				}
				for _, field := range []string{"identification_name", "identification_number"} {
					ciphertext, ok := identification[field].(string)
					if !ok {
						t.Fatalf("%s must be a string", field)
					}
					identification[field] = decrypt(t, ciphertext)
				}
				assert.Equal(t, expected, body)
				current, err := json.Marshal(req)
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, string(original), string(current))
				wantSerial := ""
				if explicitSerial {
					wantSerial = serial
				}
				assert.Equal(t, wantSerial, req.DouyinpaySerial)
			}
		})
	}
	for _, field := range []string{"name", "number"} {
		t.Run("encrypt_only_"+field, func(t *testing.T) {
			transport := &recordingTransport{body: `{"applyment_id":"apply_1"}`}
			service := newTestService(t, transport)
			service.SensitiveFields = &SensitiveFieldConfig{PlatformCertificate: cert}
			req := newRequest(t)
			jsonField, plaintext := "identification_name", req.IdentificationInfo.IdentificationName
			if field == "name" {
				req.IdentificationInfo.IdentificationNumber = ""
			} else {
				req.IdentificationInfo.IdentificationName = ""
				jsonField, plaintext = "identification_number", req.IdentificationInfo.IdentificationNumber
			}
			_, _, err := service.CreateApply(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			var body struct {
				Identification map[string]string `json:"identification_info"`
			}
			if err := json.Unmarshal([]byte(transport.request.body), &body); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, plaintext, decrypt(t, body.Identification[jsonField]))
			assert.Equal(t, serial, transport.request.header.Get(consts.DouyinpaySerial))
		})
	}
	for _, scenario := range []string{"missing_config", "missing_certificate", "serial_mismatch", "missing_serial", "oversized_plaintext"} {
		t.Run(scenario+"_does_not_send", func(t *testing.T) {
			transport := &recordingTransport{body: `{"applyment_id":"apply_1"}`}
			service := newTestService(t, transport)
			service.SensitiveFields = &SensitiveFieldConfig{PlatformCertificate: cert}
			req := newRequest(t)
			switch scenario {
			case "missing_config":
				service.SensitiveFields = nil
				req.DouyinpaySerial = serial
			case "missing_certificate":
				service.SensitiveFields = &SensitiveFieldConfig{}
				req.DouyinpaySerial = serial
			case "serial_mismatch":
				req.DouyinpaySerial = "OTHER_SERIAL"
			case "missing_serial":
				req.IdentificationInfo = nil
				service.SensitiveFields = nil
			case "oversized_plaintext":
				req.IdentificationInfo.IdentificationName = strings.Repeat("a", key.Size()-10)
			}
			original, err := json.Marshal(req)
			if err != nil {
				t.Fatal(err)
			}
			response, result, err := service.CreateApply(context.Background(), req)
			if err == nil {
				t.Fatal("expected create apply rejection")
			}
			assert.Equal(t, true, response == nil)
			assert.Equal(t, true, result == nil)
			assert.Equal(t, recordedRequest{}, transport.request)
			current, err := json.Marshal(req)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, string(original), string(current))
		})
	}
}

func TestApplymentReferenceAndUploadValidation(t *testing.T) {
	validCreateApplyRequest := CreateApplyRequest{
		BusinessCode: "apply_1", DouyinpaySerial: "platform-serial",
		OutIndirectMerchantID: "out_merchant_1", OutPaymentInstID: "payment_inst_1",
	}
	assert.Equal(t, true, CreateApplyRequest{
		BusinessCode: validCreateApplyRequest.BusinessCode, DouyinpaySerial: validCreateApplyRequest.DouyinpaySerial,
		OutPaymentInstID: validCreateApplyRequest.OutPaymentInstID,
	}.Validate() != nil)
	assert.Equal(t, true, CreateApplyRequest{
		BusinessCode: validCreateApplyRequest.BusinessCode, DouyinpaySerial: validCreateApplyRequest.DouyinpaySerial,
		OutIndirectMerchantID: validCreateApplyRequest.OutIndirectMerchantID,
	}.Validate() != nil)
	assert.Equal(t, nil, validCreateApplyRequest.Validate())
	assert.Equal(t, true, QueryApplyRequest{}.Validate() != nil)
	assert.Equal(t, true, QueryApplyRequest{ApplymentID: "apply_1", BusinessCode: "business_1"}.Validate() != nil)
	assert.Equal(t, true, CancelApplyRequest{}.Validate() != nil)
	assert.Equal(t, true, CancelApplyRequest{ApplymentID: "apply_1", BusinessCode: "business_1"}.Validate() != nil)
	assert.Equal(t, true, UploadRequest{FileName: "license.jpg"}.Validate() != nil)
	assert.Equal(t, true, UploadRequest{FileName: "license.gif", FileContent: []byte("file")}.Validate() != nil)
	assert.Equal(t, nil, UploadRequest{FileName: "license.png", FileContent: make([]byte, 5*1024*1024)}.Validate())
	assert.Equal(t, true, UploadRequest{FileName: "license.png", FileContent: make([]byte, 5*1024*1024+1)}.Validate() != nil)
}

func TestParseApplymentStatusNotifyDelegatesToCallbackHandler(t *testing.T) {
	const key = "01234567890123456789012345678901"
	const plaintext = `{"sp_mchid":"sp_1","applyment_id":"apply_1","business_code":"business_1","indirect_merchant_id":"merchant_1","applyment_state":"APPLYMENT_STATE_PASSED"}`
	const associatedData = "transaction"
	const nonce = "0123456789ab"

	block, err := aes.NewCipher([]byte(key))
	assert.Equal(t, nil, err)
	gcm, err := cipher.NewGCMWithNonceSize(block, len(nonce))
	assert.Equal(t, nil, err)
	ciphertext := base64.StdEncoding.EncodeToString(gcm.Seal(nil, []byte(nonce), []byte(plaintext), []byte(associatedData)))
	body, err := json.Marshal(map[string]interface{}{
		"id": "notification_1",
		"resource": map[string]string{
			"algorithm": consts.EncryptTypeAES256GCM, "ciphertext": ciphertext,
			"associated_data": associatedData, "nonce": nonce,
		},
	})
	assert.Equal(t, nil, err)
	request := &http.Request{
		Header: http.Header{
			consts.DouyinPaySerial:    []string{"platform-serial"},
			consts.DouyinPaySignature: []string{"signature"},
			consts.DouyinPayTimestamp: []string{"1"},
			consts.DouyinPayNonce:     []string{"nonce"},
		},
		Body: ioutil.NopCloser(bytes.NewReader(body)),
	}

	notifyRequest, content, err := ParseApplymentStatusNotify(context.Background(), callback.NewNotifyHandler(key, new(testVerifier)), request)
	assert.Equal(t, nil, err)
	assert.Equal(t, "notification_1", notifyRequest.ID)
	assert.Equal(t, "APPLYMENT_STATE_PASSED", content.ApplymentState)
}

func newTestService(t *testing.T, transport http.RoundTripper) *ApiService {
	t.Helper()
	c, err := client.NewClientWithDialSettings(context.Background(), &client.DialSettings{
		HTTPClient: &http.Client{Transport: transport}, Signer: new(testSigner), Verifier: new(testVerifier),
	})
	assert.Equal(t, nil, err)
	return &ApiService{Client: c}
}

func assertRequest(t *testing.T, request recordedRequest, method, path string, query url.Values, body string) {
	t.Helper()
	assert.Equal(t, method, request.method)
	assert.Equal(t, path, request.path)
	assert.Equal(t, query, request.query)
	assert.Equal(t, body, request.body)
}
