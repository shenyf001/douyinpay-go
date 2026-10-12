package indirectmerchant

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxUploadFileSize = 5 * 1024 * 1024

var businessCodePattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// MerchantLocator identifies an indirect merchant by either the platform ID or
// the external merchant and payment institution IDs.
type MerchantLocator struct {
	// 字段含义：抖音支付间连商户号。
	// 格式规则：字符串。
	// 业务规则：由抖音支付分配，与间连商户外部商户号和从业机构外部商户号组合二选一。
	// 示例：2026045587123820
	IndirectMerchantID string `json:"indirect_merchant_id,omitempty"`
	// 字段含义：间连商户外部商户号。
	// 格式规则：字符串。
	// 业务规则：从业机构在清算机构侧登记的间连商户唯一标识，与从业机构外部商户号组合定位。
	// 示例：10434920032
	OutIndirectMerchantID string `json:"out_indirect_merchant_id,omitempty"`
	// 字段含义：从业机构外部商户号。
	// 格式规则：字符串。
	// 业务规则：从业机构在清算机构侧登记的商户号，与间连商户外部商户号同时传入。
	// 示例：38802399238
	OutPaymentInstID string `json:"out_payment_inst_id,omitempty"`
}

func (l MerchantLocator) Validate() error {
	hasPlatformID := l.IndirectMerchantID != ""
	hasExternalID := l.OutIndirectMerchantID != "" || l.OutPaymentInstID != ""
	if hasPlatformID == hasExternalID {
		return errors.New("specify exactly one merchant locator")
	}
	if hasExternalID && (l.OutIndirectMerchantID == "" || l.OutPaymentInstID == "") {
		return errors.New("out_indirect_merchant_id and out_payment_inst_id must be specified together")
	}
	return nil
}

type CreateApplyRequest struct {
	// 字段含义：业务申请编号。
	// 格式规则：字符串，长度为1-64字符，只能由数字、小写字母或下划线组成。
	// 业务规则：【业务申请编号】1、只能由数字、小写字母或下划线组成，长度为1-64字符；2、自定义的唯一编号；3、每个编号对应一个申请单，每个申请单审核通过后会生成一个抖音支付商户号。
	// 示例：602323_10000
	BusinessCode string `json:"business_code"`
	// 字段含义：平台证书序列号。
	// 格式规则：字符串，通过 Douyinpay-Serial 请求头传递，不参与请求体序列化。
	// 业务规则：用于敏感字段加解密，详见 https://partner.douyinpay.com/wiki/682561fbd3e93f04ff85a04f/682563b39faab1051f8f5abc；敏感字段传入明文，由 SDK 加密。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	DouyinpaySerial string `json:"-"`
	// 字段含义：间连商户外部商户号。
	// 格式规则：字符串。
	// 业务规则：从业机构在清算机构侧登记的间连商户唯一标识。
	// 示例：10434920032
	OutIndirectMerchantID string `json:"out_indirect_merchant_id"`
	// 字段含义：从业机构外部商户号。
	// 格式规则：字符串。
	// 业务规则：从业机构在清算机构侧登记的商户号。
	// 示例：41110000
	OutPaymentInstID string `json:"out_payment_inst_id"`
	// 字段含义：渠道商外部商户号。
	// 格式规则：字符串。
	// 业务规则：渠道商在清算机构侧登记的商户号。
	// 示例：C023828488
	OutChannelSpID string `json:"out_channel_sp_id,omitempty"`
	// 字段含义：子从业机构外部商户号。
	// 格式规则：字符串。
	// 业务规则：【子从业机构外部商户号】支持银联下同一个从业机构商户按照地区拆分子从业机构的场景，不同地区是不同的子从业机构；若传入该字段，系统会校验子从业机构号与从业机构号（out_payment_inst_id）前4位是否一致；会按照子从业机构号进行间连商户的创建，后续交易发起需要使用该子从业机构。
	// 示例：41110100
	OutPaymentInstSubID string `json:"out_payment_inst_sub_id,omitempty"`
	// 字段含义：抖音支付间连商户号。
	// 格式规则：字符串。
	// 业务规则：【抖音支付间连商户号】如传入该商户在抖音支付已有的商户号（另一个清算机构下进件的商户），则会校验与已有商户号的一致性，包括：本申请单的商户外部商户号需与该抖音支付商户号的外部商户号一致；本申请单的商户主体类型和主体名称需与该抖音支付商户号的信息一致。校验通过后，本商户入驻申请返回的抖音支付间连商户号就是该传入的间连商户号；如未写入，则按新入驻申请创建执行。
	// 示例：2026045587123820
	IndirectMerchantID string `json:"indirect_merchant_id,omitempty"`
	// 字段含义：主体资料。
	// 格式规则：对象。
	// 业务规则：【主体资料】请填写商家的营业执照/登记证书、经营者/法人的证件等信息。
	// 示例：{"subject_type":"SUBJECT_TYPE_ENTERPRISE","certificate_info":{"register_address":"湖南省长沙市xx区xx路xx号","period_begin":"2019-08-01","period_end":"2019-08-02"}}
	SubjectInfo *SubjectInfo `json:"subject_info,omitempty"`
	// 字段含义：经营者/法人身份证件。
	// 格式规则：对象。
	// 业务规则：【经营者/法人身份信息】1、个体户：请上传经营者的身份证件。2、企业/社会组织：请上传法人的身份证件。3、政府机关/事业单位：请上传法人/经办人的身份证件。4、小微商户：请上传经营者的身份证件。
	// 示例：{"identification_type":"IDENTIFICATION_TYPE_IDCARD","period_begin":"2019-08-01","period_end":"2019-08-02"}
	IdentificationInfo *IdentificationInfo `json:"identification_info,omitempty"`
	// 字段含义：经营信息。
	// 格式规则：对象。
	// 业务规则：【经营资料】请填写商家的经营业务信息、售卖商品/提供服务场景信息。
	// 示例：{"merchant_short_name":"个体户张三","service_phone":"18042002458","mcc":"7300"}
	BusinessInfo *BusinessInfo `json:"business_info,omitempty"`
	// 字段含义：补充信息。
	// 格式规则：对象。
	// 业务规则：根据实际审核情况，额外要求商家提供指定的补充资料。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	AdditionInfo *AdditionInfo `json:"addition_info,omitempty"`
}

func (r CreateApplyRequest) Validate() error {
	if !businessCodePattern.MatchString(r.BusinessCode) {
		return errors.New("business_code must contain 1-64 lowercase letters, digits, or underscores")
	}
	if r.DouyinpaySerial == "" {
		return errors.New("douyinpay_serial is required")
	}
	if r.OutIndirectMerchantID == "" {
		return errors.New("out_indirect_merchant_id is required")
	}
	if r.OutPaymentInstID == "" {
		return errors.New("out_payment_inst_id is required")
	}
	return nil
}

type CreateApplyResponse struct {
	// 字段含义：抖音支付分配的申请单号。
	ApplymentID string `json:"applyment_id"`
}

type QueryApplyRequest struct {
	// 字段含义：抖音支付申请单号。
	// 格式规则：字符串。
	// 业务规则：抖音支付生成并下发的申请单号，与业务申请编号选择一种定位方式。
	// 示例：NIM2000002124775691
	ApplymentID string
	// 字段含义：业务申请编号。
	// 格式规则：字符串。
	// 业务规则：自定义唯一编号，每个编号对应一个申请单。
	// 示例：1900013511_10000
	BusinessCode string
}

func (r QueryApplyRequest) Validate() error {
	return validateApplymentReference(r.ApplymentID, r.BusinessCode)
}

type QueryApplyResponse struct {
	// 字段含义：业务申请编号；驳回后可用相同编号覆盖修改原申请单。
	BusinessCode string `json:"business_code"`
	// 字段含义：抖音支付分配的申请单号。
	ApplymentID string `json:"applyment_id"`
	// 字段含义：抖音支付间连商户号，申请完成时返回。
	IndirectMerchantID string `json:"indirect_merchant_id"`
	// 字段含义：申请单状态；APPLYMENT_STATE_WAITTING_FOR_AUDIT 审核中、
	// APPLYMENT_STATE_REJECTED 已驳回、APPLYMENT_STATE_PASSED 已完成、
	// APPLYMENT_STATE_CANCELED 已作废；保留未知状态原值。
	ApplymentState string `json:"applyment_state"`
	// 字段含义：各项资料审核驳回原因，申请驳回时返回。
	AuditDetail []AuditDetail `json:"audit_detail"`
}

// AuditDetail 为单项资料的审核结果。
type AuditDetail struct {
	// 字段含义：提交申请单的资料项字段名。
	Field string `json:"field"`
	// 字段含义：提交资料项被驳回的原因。
	RejectReason string `json:"reject_reason"`
}

type CancelApplyRequest struct {
	// 字段含义：抖音支付申请单号。
	// 格式规则：字符串。
	// 业务规则：由抖音支付生成并下发；与业务申请编号选择一种定位方式。仅待提交、审核中可撤销。
	// 示例：NIM2000002124775691
	ApplymentID string `json:"applyment_id,omitempty"`
	// 字段含义：业务申请编号。
	// 格式规则：字符串，由数字、小写字母或下划线组成。
	// 业务规则：自定义唯一编号，每个编号对应一个申请单；仅待提交、审核中可撤销。
	// 示例：1900013511_10000
	BusinessCode string `json:"business_code,omitempty"`
}

func (r CancelApplyRequest) Validate() error {
	return validateApplymentReference(r.ApplymentID, r.BusinessCode)
}

type QueryMerchantRequest struct {
	MerchantLocator
	// 字段含义：渠道商外部商户号。
	// 格式规则：字符串。
	// 业务规则：渠道商在清算机构侧登记的商户号，仅与外部商户定位方式组合传入。
	// 示例：900023828488
	OutChannelSpID string
}

func (r QueryMerchantRequest) Validate() error {
	if err := r.MerchantLocator.Validate(); err != nil {
		return err
	}
	if r.IndirectMerchantID != "" && r.OutChannelSpID != "" {
		return errors.New("out_channel_sp_id is only supported with external merchant locator")
	}
	return nil
}

type QueryMerchantResponse struct {
	// 字段含义：外部间连商户号。
	OutIndirectMerchantID string `json:"out_indirect_merchant_id"`
	// 字段含义：外部从业机构商户号。
	OutPaymentInstID string `json:"out_payment_inst_id"`
	// 字段含义：外部渠道商户号。
	OutChannelSpID string `json:"out_channel_sp_id"`
	// 字段含义：业务申请编号。
	BusinessCode string `json:"business_code"`
	// 字段含义：申请单号。
	ApplymentID string `json:"applyment_id"`
	// 字段含义：抖音支付间连商户号。
	IndirectMerchantID string `json:"indirect_merchant_id"`
	// 字段含义：主体资料。
	SubjectInfo *QueryMerchantSubjectInfo `json:"subject_info"`
	// 字段含义：经营者/法人身份证件。
	IdentificationInfo *QueryMerchantIdentificationInfo `json:"identification_info"`
	// 字段含义：经营信息。
	BusinessInfo *QueryMerchantBusinessInfo `json:"business_info"`
}

// QueryMerchantSubjectInfo 为查询商户响应的主体资料。
type QueryMerchantSubjectInfo struct {
	// 字段含义：主体类型。
	SubjectType string `json:"subject_type"`
	// 字段含义：主体证件信息。
	CertificateInfo *QueryMerchantCertificateInfo `json:"certificate_info"`
}

// QueryMerchantCertificateInfo 为查询商户响应的主体证件信息。
type QueryMerchantCertificateInfo struct {
	// 字段含义：证件编号。
	CertNo string `json:"cert_no"`
	// 字段含义：商户名称。
	MerchantName string `json:"merchant_name"`
	// 字段含义：证件图片。
	CertCopy string `json:"cert_copy"`
	// 字段含义：注册地址。
	RegisterAddress string `json:"register_address"`
	// 字段含义：经营者/法人姓名。
	LegalPersonName string `json:"legal_person_name"`
	// 字段含义：有效期限开始日期。
	PeriodBegin string `json:"period_begin"`
	// 字段含义：有效期限结束日期。
	PeriodEnd string `json:"period_end"`
}

// QueryMerchantIdentificationInfo 为查询商户响应的经营者/法人身份证件信息。
type QueryMerchantIdentificationInfo struct {
	// 字段含义：证件类型。
	IdentificationType string `json:"identification_type"`
	// 字段含义：证件持有人姓名。
	IdentificationName string `json:"identification_name"`
	// 字段含义：证件持有人证件号码。
	IdentificationNumber string `json:"identification_number"`
	// 字段含义：证件持有人有效期开始时间。
	PeriodBegin string `json:"period_begin"`
	// 字段含义：证件持有人有效期结束时间。
	PeriodEnd string `json:"period_end"`
	// 字段含义：证件正面照片。
	IdentificationFrontCopy string `json:"identification_front_copy"`
	// 字段含义：证件背面照片。
	IdentificationBackCopy string `json:"identification_back_copy"`
}

// QueryMerchantBusinessInfo 为查询商户响应的经营信息。
type QueryMerchantBusinessInfo struct {
	// 字段含义：商户简称。
	MerchantShortName string `json:"merchant_short_name"`
	// 字段含义：客服电话。
	ServicePhone string `json:"service_phone"`
	// 字段含义：所属行业。
	MCC string `json:"mcc"`
	// 字段含义：特殊资质图片。
	SpecialIndustryPics []string `json:"special_industry_pics"`
}

type ModifyMerchantRequest struct {
	MerchantLocator
	// 字段含义：渠道商外部商户号。
	// 格式规则：字符串。
	// 业务规则：渠道商在清算机构侧登记的商户号，仅与外部商户定位方式组合传入。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	OutChannelSpID string `json:"out_channel_sp_id,omitempty"`
	// 字段含义：商户简称。
	// 格式规则：UTF-8字符串，1-32个字符；前后不能有空格、制表符、换行符，不能仅含特殊字符。
	// 业务规则：支付完成页向买家展示，需与经营类目相关；支持数字、英文字母、汉字及特殊字符。
	// 示例：个体户张三
	MerchantShortName string `json:"merchant_short_name,omitempty"`
	// 字段含义：客服电话。
	// 格式规则：前后不能有空格、制表符、换行符；11位手机号码或5-20位数字、连字符“-”、加号“+”。
	// 业务规则：交易记录中向买家展示，需确保电话畅通以便平台回拨确认。
	// 示例：18042002458
	ServicePhone string `json:"service_phone,omitempty"`
}

func (r ModifyMerchantRequest) Validate() error {
	return validateMerchantChannel(r.MerchantLocator, r.OutChannelSpID)
}

type BindAppIDRequest struct {
	MerchantLocator
	// 字段含义：渠道商外部商户号。
	// 格式规则：字符串。
	// 业务规则：渠道商在清算机构侧登记的商户号，仅与外部商户定位方式组合传入。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	OutChannelSpID string `json:"out_channel_sp_id,omitempty"`
	// 字段含义：关联应用APPID。
	// 格式规则：字符串。
	// 业务规则：需要给间连商户绑定的应用；只支持新增配置，非同主体小程序需开发者确认授权。
	// 示例：tt2342ffw32fwef23r
	AppID string `json:"app_id"`
}

func (r BindAppIDRequest) Validate() error {
	if err := validateMerchantChannel(r.MerchantLocator, r.OutChannelSpID); err != nil {
		return err
	}
	if r.AppID == "" {
		return errors.New("app_id is required")
	}
	return nil
}

type BindAppIDResponse struct {
	// 字段含义：应用信息，绑定成功或待授权时返回。
	AppInfo *AppInfo `json:"app_info"`
}

type QueryAppIDRequest struct {
	MerchantLocator
	// 字段含义：渠道商外部商户号。
	// 格式规则：字符串。
	// 业务规则：渠道商在清算机构侧登记的商户号，仅与外部商户定位方式组合传入。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	OutChannelSpID string
	// 字段含义：应用APPID。
	// 格式规则：字符串。
	// 业务规则：传入时查询指定应用，未传入时查询商户配置的应用列表。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	AppID string
}

func (r QueryAppIDRequest) Validate() error {
	return validateMerchantChannel(r.MerchantLocator, r.OutChannelSpID)
}

type QueryAppIDResponse struct {
	// 字段含义：应用信息列表。
	AppInfoList []AppInfo `json:"app_info_list"`
}

type AppInfo struct {
	// 字段含义：应用APPID。
	AppID string `json:"app_id"`
	// 字段含义：应用类型，中文值包括小程序应用、移动应用、网站应用。
	AppType string `json:"app_type"`
	// 字段含义：绑定状态，BIND 已绑定，WAIT_CONFIRM 待开发者确认；保留未知状态原值。
	BindStatus string `json:"bind_status"`
}

type UploadRequest struct {
	// 字段含义：图片文件名称。
	// 格式规则：字符串，1-128字符，以jpg、jpeg、bmp或png为后缀。
	// 业务规则：商户自定义名称，由SDK写入meta.filename。
	// 示例：filea.jpg
	FileName string
	// 字段含义：图片分段MIME类型。
	// 格式规则：字符串，未设置时使用application/octet-stream。
	// 业务规则：用于file分段的Content-Type。
	// 示例：image/jpg
	ContentType string
	// 字段含义：图片文件二进制内容。
	// 格式规则：非空字节数组，不超过5MB。
	// 业务规则：只支持JPG、BMP、PNG；SDK计算SHA-256写入meta并使用同一份meta JSON签名。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	FileContent []byte
}

func (r UploadRequest) Validate() error {
	if len(r.FileContent) == 0 {
		return errors.New("file content is required")
	}
	if len(r.FileContent) > maxUploadFileSize {
		return fmt.Errorf("file content must not exceed %d bytes", maxUploadFileSize)
	}
	if !utf8.ValidString(r.FileName) || utf8.RuneCountInString(r.FileName) == 0 ||
		utf8.RuneCountInString(r.FileName) > 128 {
		return errors.New("filename must contain 1-128 UTF-8 characters")
	}
	switch {
	case strings.HasSuffix(strings.ToLower(r.FileName), ".jpg"),
		strings.HasSuffix(strings.ToLower(r.FileName), ".jpeg"),
		strings.HasSuffix(strings.ToLower(r.FileName), ".bmp"),
		strings.HasSuffix(strings.ToLower(r.FileName), ".png"):
		return nil
	default:
		return errors.New("filename must use a jpg, jpeg, bmp, or png extension")
	}
}

type UploadResponse struct {
	// 字段含义：媒体文件标识Id。
	MediaID string `json:"media_id"`
}

// ApplymentStatusNotification is the decrypted content of an indirect
// merchant application status notification.
type ApplymentStatusNotification struct {
	// 字段含义：由抖音支付生成并下发的从业机构户号。
	SpMchid string `json:"sp_mchid"`
	// 字段含义：抖音支付生成的申请单号。
	ApplymentID string `json:"applyment_id"`
	// 字段含义：商户自提交的申请单号，站点页面提交时为空。
	BusinessCode string `json:"business_code"`
	// 字段含义：抖音支付间连商户号，申请完成时返回。
	IndirectMerchantID string `json:"indirect_merchant_id"`
	// 字段含义：申请单状态，REJECTED 已驳回、PASSED 已完成、CANCELED 已作废；保留未知状态原值。
	ApplymentState string `json:"applyment_state"`
	// 字段含义：失败原因详情，驳回时返回原因，作废时返回申请单已撤销。
	FailReason string `json:"fail_reason"`
}

func validateMerchantChannel(locator MerchantLocator, channel string) error {
	if err := locator.Validate(); err != nil {
		return err
	}
	if locator.IndirectMerchantID != "" && channel != "" {
		return errors.New("out_channel_sp_id is only supported with external merchant locator")
	}
	return nil
}

func validateApplymentReference(applymentID, businessCode string) error {
	if (applymentID == "") == (businessCode == "") {
		return errors.New("specify exactly one of applyment_id or business_code")
	}
	return nil
}
