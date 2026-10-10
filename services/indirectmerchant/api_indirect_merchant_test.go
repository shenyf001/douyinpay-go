package indirectmerchant

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
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
	assert.Equal(t, true, strings.Contains(transport.request.body, `name="meta"`))
	assert.Equal(t, true, strings.Contains(transport.request.body, `"filename":"license.jpg"`))
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
