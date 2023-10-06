package manifest

import (
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/lifecycle"
)

// Name is a default name used by manifest file.
const Name = "app-manifest.json"

// New creates new instance of a manifest.
func New() *Manifest {
	return &Manifest{}
}

type Manifest struct {
	Configs     []AppConfig         `json:"configs"`
	UIBlocks    []AppUBLock         `json:"ui_blocks"`
	UIButtons   []UIButton          `json:"ui_buttons"`
	Auth        AppAuth             `json:"auth"`
	InitFlow    []string            `json:"init_flow"`
	Services    []AppService        `json:"services"`
	AppState    lifecycle.AppStates `json:"app_state"`
	ConfigState any                 `json:"config_state"`
}

type AppConfig struct {
	ID          string            `json:"id"`
	Label       MultilingualLabel `json:"label"`
	ValT        string            `json:"val_t"`
	UI          AppConfigUI       `json:"ui"`
	Val         Value             `json:"val"`
	IsRequired  bool              `json:"is_required"`
	ConfigPoint string            `json:"config_point"`
	Hidden      bool              `json:"hidden"`
}

func (b *AppConfig) Hide() {
	b.Hidden = true
}

func (b *AppConfig) Show() {
	b.Hidden = false
}

type MultilingualLabel map[string]string

type AppAuth struct {
	Type                  string `json:"type"`
	CodeGrantLoginPageURL string `json:"code_grant_login_page_url"`
	RedirectURL           string `json:"redirect_url"`
	ClientID              string `json:"client_id"`
	Secret                string `json:"secret"`
	PartnerID             string `json:"partner_id"`
	AuthEndpoint          string `json:"auth_endpoint"`
}

type AppService struct {
	Name       string               `json:"name"`
	Alias      string               `json:"alias"`
	Address    string               `json:"address"`
	Interfaces []fimptype.Interface `json:"interfaces"`
}

type Value struct {
	Default interface{} `json:"default"`
}

type AppConfigUI struct {
	Type   string      `json:"type"`
	Select interface{} `json:"select"`
}

type SelectOption struct {
	Val   interface{}       `json:"val"`
	Label map[string]string `json:"label"`
}

type UIButton struct {
	ID    string            `json:"id"`
	Label MultilingualLabel `json:"label"`
	Req   struct {
		Serv  string `json:"serv"`
		IntfT string `json:"intf_t"`
		Val   string `json:"val"`
	} `json:"req"`
	Hidden bool `json:"hidden"`
}

func (b *UIButton) Hide() {
	b.Hidden = true
}

func (b *UIButton) Show() {
	b.Hidden = false
}

type ButtonActionResponse struct {
	Operation       string `json:"op"`
	OperationStatus string `json:"op_status"`
	Next            string `json:"next"`
	ErrorCode       string `json:"error_code"`
	ErrorText       string `json:"error_text"`
}

type AppUBLock struct {
	ID      string            `json:"id"`
	Header  MultilingualLabel `json:"header"`
	Text    MultilingualLabel `json:"text"`
	Configs []string          `json:"configs"`
	Buttons []string          `json:"buttons"`
	Footer  MultilingualLabel `json:"footer"`
	Hidden  bool              `json:"hidden"`
}

func (b *AppUBLock) Hide() {
	b.Hidden = true
}

func (b *AppUBLock) Show() {
	b.Hidden = false
}

func (m *Manifest) GetUIBlock(id string) *AppUBLock {
	for i, b := range m.UIBlocks {
		if b.ID == id {
			return &m.UIBlocks[i]
		}
	}

	return nil
}

func (m *Manifest) GetButton(id string) *UIButton {
	for i, b := range m.UIButtons {
		if b.ID == id {
			return &m.UIButtons[i]
		}
	}

	return nil
}

func (m *Manifest) GetAppConfig(id string) *AppConfig {
	for i, c := range m.Configs {
		if c.ID == id {
			return &m.Configs[i]
		}
	}

	return nil
}

type AuthResponse struct {
	Status    string `json:"status"`
	ErrorText string `json:"error_text"`
	ErrorCode string `json:"error_code"`
}
