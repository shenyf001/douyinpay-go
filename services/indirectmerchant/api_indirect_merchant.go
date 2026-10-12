package indirectmerchant

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	neturl "net/url"

	"github.com/douyinpay/douyinpay-go/client"
	"github.com/douyinpay/douyinpay-go/services/callback"
	"github.com/douyinpay/douyinpay-go/tools/consts"
)

// ApiService 提供间连商户入驻、资料、应用绑定和图片上传能力。
type ApiService struct {
	Client          *client.Client
	SensitiveFields *SensitiveFieldConfig
}

// ParseApplymentStatusNotify verifies and decrypts an indirect merchant
// application status notification.
func ParseApplymentStatusNotify(ctx context.Context, handler *callback.Handler, request *nethttp.Request) (*callback.Request, *ApplymentStatusNotification, error) {
	if handler == nil {
		return nil, nil, fmt.Errorf("handler is required")
	}
	content := new(ApplymentStatusNotification)
	notifyRequest, err := handler.ParseCallback(ctx, request, content)
	if err != nil {
		return nil, nil, err
	}
	return notifyRequest, content, nil
}

// CreateApply 提交间连商户入驻申请，返回抖音支付分配的申请单号。
// IdentificationInfo 中的姓名与证件号由 SDK 使用配置的平台证书加密，
// Douyinpay-Serial 必须与加密证书一致。调用方对象不被修改。
func (a *ApiService) CreateApply(ctx context.Context, req CreateApplyRequest) (*CreateApplyResponse, *client.APIResult, error) {
	var err error
	req, err = a.encryptIdentification(req)
	if err != nil {
		return nil, nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}
	resp := new(CreateApplyResponse)
	result, err := a.doRequest(ctx, nethttp.MethodPost, consts.CreateIndirectMerchantApplyPath, nil, req, resp, nethttp.Header{
		consts.DouyinpaySerial: []string{req.DouyinpaySerial},
	})
	if err != nil {
		return nil, result, err
	}
	return resp, result, nil
}

// QueryApply 通过申请单号或业务申请编号查询申请状态及资料审核结果。
func (a *ApiService) QueryApply(ctx context.Context, req QueryApplyRequest) (*QueryApplyResponse, *client.APIResult, error) {
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}
	query := neturl.Values{}
	addApplymentReference(query, req.ApplymentID, req.BusinessCode)
	resp := new(QueryApplyResponse)
	result, err := a.doRequest(ctx, nethttp.MethodGet, consts.QueryIndirectMerchantApplyPath, query, nil, resp, nil)
	if err != nil {
		return nil, result, err
	}
	return resp, result, nil
}

// CancelApply 撤销待提交或审核中的入驻申请；申请当前状态由服务端校验。
func (a *ApiService) CancelApply(ctx context.Context, req CancelApplyRequest) (*client.APIResult, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return a.doRequest(ctx, nethttp.MethodPost, consts.CancelIndirectMerchantApplyPath, nil, req, nil, nil)
}

// QueryMerchant 查询商户资料，响应验签成功后解密证件持有人姓名和证件号码。
func (a *ApiService) QueryMerchant(ctx context.Context, req QueryMerchantRequest) (*QueryMerchantResponse, *client.APIResult, error) {
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}
	query := merchantLocatorValues(req.MerchantLocator)
	if req.OutChannelSpID != "" {
		query.Add("out_channel_sp_id", req.OutChannelSpID)
	}
	resp := new(QueryMerchantResponse)
	result, err := a.doRequest(ctx, nethttp.MethodGet, consts.QueryIndirectMerchantPath, query, nil, resp, nil)
	if err != nil {
		return nil, result, err
	}
	if resp.IdentificationInfo != nil {
		var key *rsa.PrivateKey
		if a.SensitiveFields != nil {
			key = a.SensitiveFields.MerchantPrivateKey
		}
		info := resp.IdentificationInfo
		if info.IdentificationName, err = decryptField(info.IdentificationName, key); err != nil {
			return nil, result, err
		}
		if info.IdentificationNumber, err = decryptField(info.IdentificationNumber, key); err != nil {
			return nil, result, err
		}
	}
	return resp, result, nil
}

// ModifyMerchant 修改商户简称和客服电话。
func (a *ApiService) ModifyMerchant(ctx context.Context, req ModifyMerchantRequest) (*client.APIResult, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return a.doRequest(ctx, nethttp.MethodPost, consts.ModifyIndirectMerchantPath, nil, req, nil, nil)
}

// BindAppID 新增应用绑定；非同主体小程序可能需要开发者确认授权。
func (a *ApiService) BindAppID(ctx context.Context, req BindAppIDRequest) (*BindAppIDResponse, *client.APIResult, error) {
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}
	resp := new(BindAppIDResponse)
	result, err := a.doRequest(ctx, nethttp.MethodPost, consts.BindIndirectMerchantAppIDPath, nil, req, resp, nil)
	if err != nil {
		return nil, result, err
	}
	return resp, result, nil
}

// QueryAppID 查询商户绑定应用，可指定单个应用。
func (a *ApiService) QueryAppID(ctx context.Context, req QueryAppIDRequest) (*QueryAppIDResponse, *client.APIResult, error) {
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}
	query := merchantLocatorValues(req.MerchantLocator)
	if req.OutChannelSpID != "" {
		query.Add("out_channel_sp_id", req.OutChannelSpID)
	}
	if req.AppID != "" {
		query.Add("app_id", req.AppID)
	}
	resp := new(QueryAppIDResponse)
	result, err := a.doRequest(ctx, nethttp.MethodGet, consts.QueryIndirectMerchantAppIDPath, query, nil, resp, nil)
	if err != nil {
		return nil, result, err
	}
	return resp, result, nil
}

// Upload 上传资质图片，以meta JSON签名并返回media_id。
func (a *ApiService) Upload(ctx context.Context, req UploadRequest) (*UploadResponse, *client.APIResult, error) {
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}
	meta, err := json.Marshal(uploadMeta{
		Filename: req.FileName,
		SHA256:   sha256Hex(req.FileContent),
	})
	if err != nil {
		return nil, nil, err
	}
	result, err := a.Client.RequestMultipart(ctx, &client.MultipartRequestEntity{
		Method:            nethttp.MethodPost,
		RequestPath:       consts.DouyinPayServer + consts.UploadMerchantMediaPath,
		Header:            nethttp.Header{consts.Accept: []string{consts.ApplicationJSON}},
		SignBody:          string(meta),
		Fields:            map[string]string{"meta": string(meta)},
		FieldContentTypes: map[string]string{"meta": consts.ApplicationJSON},
		FileField:         "file",
		FileName:          req.FileName,
		ContentType:       req.ContentType,
		FileContent:       req.FileContent,
	})
	if err != nil {
		return nil, result, err
	}
	resp := new(UploadResponse)
	if err = client.UnMarshalResponse(result.Response, resp); err != nil {
		return nil, result, err
	}
	return resp, result, nil
}

func (a *ApiService) doRequest(ctx context.Context, method, path string, query neturl.Values, body, resp interface{}, header nethttp.Header) (*client.APIResult, error) {
	r := &client.RequestEntity{
		Method:      method,
		RequestPath: consts.DouyinPayServer + path,
		QueryParams: query,
		ContentType: consts.ApplicationJSON,
		Header:      header,
		PostBody:    body,
	}
	result, err := a.Client.Request(ctx, r)
	if err != nil || resp == nil {
		return result, err
	}
	if err = client.UnMarshalResponse(result.Response, resp); err != nil {
		return result, err
	}
	return result, nil
}

func merchantLocatorValues(locator MerchantLocator) neturl.Values {
	values := neturl.Values{}
	if locator.IndirectMerchantID != "" {
		values.Add("indirect_merchant_id", locator.IndirectMerchantID)
		return values
	}
	values.Add("out_indirect_merchant_id", locator.OutIndirectMerchantID)
	values.Add("out_payment_inst_id", locator.OutPaymentInstID)
	return values
}

func addApplymentReference(values neturl.Values, applymentID, businessCode string) {
	if applymentID != "" {
		values.Add("applyment_id", applymentID)
		return
	}
	values.Add("business_code", businessCode)
}

type uploadMeta struct {
	Filename string `json:"filename"`
	SHA256   string `json:"sha256"`
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
