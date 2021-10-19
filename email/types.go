package email

import "github.com/RagOfJoes/idp/user/identity"

// Base types
//

type Client interface {
	SendWelcome(to string, user identity.Identity, verificationURL string) error
	SendVerification(to string, user identity.Identity, verificationURL string) error
	SendRecovery(to []string, recoveryURL string) error
}

// A majority of Sendgrid's types
// Reference: https://github.com/sendgrid/sendgrid-go/blob/main/helpers/mail/mail_v3.go
//

// Email stores a person's name and email information
type Email struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// Payload defines all fields that can be passed
type Payload struct {
	From             Email              `json:"from"`
	Subject          string             `json:"subject,omitempty"`
	Personalizations []*Personalization `json:"personalizations,omitempty"`
	Attachments      []*Attachment      `json:"attachments,omitempty"`
	TemplateID       string             `json:"template_id,omitempty"`
	Sections         map[string]string  `json:"sections,omitempty"`
	Headers          map[string]string  `json:"headers,omitempty"`
	Categories       []string           `json:"categories,omitempty"`
	CustomArgs       map[string]string  `json:"custom_args,omitempty"`
	SendAt           int                `json:"send_at,omitempty"`
	BatchID          string             `json:"batch_id,omitempty"`
	IPPoolID         string             `json:"ip_pool_name,omitempty"`
	MailSettings     *MailSettings      `json:"mail_settings,omitempty"`
	TrackingSettings *TrackingSettings  `json:"tracking_settings,omitempty"`
	ReplyTo          *Email             `json:"reply_to,omitempty"`
}

// Personalization holds mail body struct
type Personalization struct {
	To                  []*Email               `json:"to,omitempty"`
	From                *Email                 `json:"from,omitempty"`
	CC                  []*Email               `json:"cc,omitempty"`
	BCC                 []*Email               `json:"bcc,omitempty"`
	Subject             string                 `json:"subject,omitempty"`
	Headers             map[string]string      `json:"headers,omitempty"`
	Substitutions       map[string]string      `json:"substitutions,omitempty"`
	CustomArgs          map[string]string      `json:"custom_args,omitempty"`
	DynamicTemplateData map[string]interface{} `json:"dynamic_template_data,omitempty"`
	Categories          []string               `json:"categories,omitempty"`
	SendAt              int                    `json:"send_at,omitempty"`
}

// Attachment holds attachment information
type Attachment struct {
	Content     string `json:"content,omitempty"`
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Disposition string `json:"disposition,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

// MailSettings defines mail and spamCheck settings
type MailSettings struct {
	BCC                         *BCCSetting       `json:"bcc,omitempty"`
	BypassListManagement        *Setting          `json:"bypass_list_management,omitempty"`
	BypassSpamManagement        *Setting          `json:"bypass_spam_management,omitempty"`
	BypassBounceManagement      *Setting          `json:"bypass_bounce_management,omitempty"`
	BypassUnsubscribeManagement *Setting          `json:"bypass_unsubscribe_management,omitempty"`
	SandboxMode                 *Setting          `json:"sandbox_mode,omitempty"`
	SpamCheckSetting            *SpamCheckSetting `json:"spam_check,omitempty"`
}

// TrackingSettings holds tracking settings and mail settings
type TrackingSettings struct {
	ClickTracking        *ClickTrackingSetting        `json:"click_tracking,omitempty"`
	OpenTracking         *OpenTrackingSetting         `json:"open_tracking,omitempty"`
	SubscriptionTracking *SubscriptionTrackingSetting `json:"subscription_tracking,omitempty"`
	GoogleAnalytics      *GASetting                   `json:"ganalytics,omitempty"`
	BCC                  *BCCSetting                  `json:"bcc,omitempty"`
	BypassListManagement *Setting                     `json:"bypass_list_management,omitempty"`
	SandboxMode          *SandboxModeSetting          `json:"sandbox_mode,omitempty"`
}

// BCCSetting holds email bcc settings  to enable of disable
// default is false
type BCCSetting struct {
	Enable *bool  `json:"enable,omitempty"`
	Email  string `json:"email,omitempty"`
}

// ClickTrackingSetting defines what it says it does
type ClickTrackingSetting struct {
	Enable     *bool `json:"enable,omitempty"`
	EnableText *bool `json:"enable_text,omitempty"`
}

// OpenTrackingSetting defines what it says it does
type OpenTrackingSetting struct {
	Enable          *bool  `json:"enable,omitempty"`
	SubstitutionTag string `json:"substitution_tag,omitempty"`
}

// SandboxModeSetting defines what it says it does
type SandboxModeSetting struct {
	Enable      *bool             `json:"enable,omitempty"`
	ForwardSpam *bool             `json:"forward_spam,omitempty"`
	SpamCheck   *SpamCheckSetting `json:"spam_check,omitempty"`
}

// SpamCheckSetting holds spam settings and
// which can be enable or disable and
// contains spamThreshold value
type SpamCheckSetting struct {
	Enable        *bool  `json:"enable,omitempty"`
	SpamThreshold int    `json:"threshold,omitempty"`
	PostToURL     string `json:"post_to_url,omitempty"`
}

// SubscriptionTrackingSetting defines what it says it does
type SubscriptionTrackingSetting struct {
	Enable          *bool  `json:"enable,omitempty"`
	Text            string `json:"text,omitempty"`
	Html            string `json:"html,omitempty"`
	SubstitutionTag string `json:"substitution_tag,omitempty"`
}

// GASetting defines Google Analytics settings
type GASetting struct {
	Enable          *bool  `json:"enable,omitempty"`
	CampaignSource  string `json:"utm_source,omitempty"`
	CampaignTerm    string `json:"utm_term,omitempty"`
	CampaignContent string `json:"utm_content,omitempty"`
	CampaignName    string `json:"utm_campaign,omitempty"`
	CampaignMedium  string `json:"utm_medium,omitempty"`
}

// Setting enables the mail settings
type Setting struct {
	Enable *bool `json:"enable,omitempty"`
}
