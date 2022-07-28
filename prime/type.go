package prime

import (
	"time"
)

const (
	ComponentDevice   = "device"
	ComponentThing    = "thing"
	ComponentRoom     = "room"
	ComponentArea     = "area"
	ComponentHouse    = "house"
	ComponentHub      = "hub"
	ComponentShortcut = "shortcut"
	ComponentMode     = "mode"
	ComponentTimer    = "timer"
	ComponentService  = "service"
	ComponentState    = "state"

	CmdGet    = "get"
	CmdSet    = "set"
	CmdEdit   = "edit"
	CmdDelete = "delete"
	CmdAdd    = "add"
)

var validComponents = []string{
	ComponentDevice,
	ComponentThing,
	ComponentRoom,
	ComponentArea,
	ComponentHouse,
	ComponentHub,
	ComponentShortcut,
	ComponentMode,
	ComponentTimer,
	ComponentService,
	ComponentState,
}

type ComponentSet struct {
	Devices   Devices           `json:"device,omitempty"`
	Things    Things            `json:"thing,omitempty"`
	Rooms     Rooms             `json:"room,omitempty"`
	Areas     Areas             `json:"area,omitempty"`
	House     *House            `json:"house,omitempty"`
	Hub       *Hub              `json:"hub,omitempty"`
	Shortcuts Shortcuts         `json:"shortcut,omitempty"`
	Modes     Modes             `json:"mode,omitempty"`
	Timers    Timers            `json:"timer,omitempty"`
	Services  *VinculumServices `json:"service,omitempty"`
	State     *State            `json:"state,omitempty"`
}

type Devices []*Device

type Device struct {
	Fimp          Fimp                   `json:"fimp"`
	Client        ClientType             `json:"client"`
	Functionality *string                `json:"functionality"`
	Service       map[string]Service     `json:"services"`
	ID            int                    `json:"id"`
	Lrn           bool                   `json:"lrn"`
	Model         string                 `json:"model"`
	ModelAlias    string                 `json:"modelAlias"`
	Param         map[string]interface{} `json:"param"`
	Problem       bool                   `json:"problem"`
	Room          *int                   `json:"room"`
	Changes       map[string]interface{} `json:"changes"`
	ThingID       *int                   `json:"thing"`
	Type          map[string]interface{} `json:"type"`
}

type Service struct {
	Addr       string                 `json:"addr,omitempty"`
	Enabled    bool                   `json:"enabled,omitempty"`
	Interfaces []string               `json:"intf"`
	Props      map[string]interface{} `json:"props"`
}

type Fimp struct {
	Adapter string `json:"adapter"`
	Address string `json:"address"`
	Group   string `json:"group"`
}

type ClientType struct {
	Name          *string `json:"name,omitempty"`
	OpenStateType *string `json:"openStateType,omitempty"`
}

type Things []*Thing

type Thing struct {
	ID      int               `json:"id"`
	Address string            `json:"addr"`
	Name    string            `json:"name"`
	Devices []int             `json:"devices,omitempty"`
	Props   map[string]string `json:"props,omitempty"`
	RoomID  int               `json:"room"`
}

type Rooms []*Room

type Room struct {
	Alias   string     `json:"alias"`
	ID      int        `json:"id"`
	Param   RoomParams `json:"param"`
	Client  ClientType `json:"client"`
	Type    *string    `json:"type"`
	Area    *int       `json:"area"`
	Outside bool       `json:"outside"`
}

type RoomParams struct {
	Heating  RoomHeating `json:"heating"`
	Lighting interface{} `json:"lighting"`
	Security interface{} `json:"security"`
	Sensors  []string    `json:"sensors"`
	Shading  interface{} `json:"shading"`
	Triggers interface{} `json:"triggers"`
}

type RoomHeating struct {
	Desired    float64 `json:"desired"`
	Target     float64 `json:"target"`
	Thermostat bool    `json:"thermostat"`
	Actuator   bool    `json:"actuator"`
	Power      string  `json:"power"`
}

type Areas []*Area

type Area struct {
	ID    int       `json:"id"`
	Name  string    `json:"name"`
	Type  string    `json:"type"`
	Props AreaProps `json:"props"`
}

type AreaProps struct {
	HNumber string `json:"hNumber"`
	TransNr string `json:"transNr"`
}

type House struct {
	Learning interface{} `json:"learning"`
	Mode     string      `json:"mode"`
	Time     time.Time   `json:"time"`
}

type Hub struct {
	Mode HubMode `json:"mode"`
}

type HubMode struct {
	Current  string `json:"current"`
	Previous string `json:"prev"`
}

type UserInfo struct {
	UID  string   `json:"uuid,omitempty"`
	Name UserName `json:"name,omitempty"`
}

type UserName struct {
	Fullname string `json:"fullname,omitempty"`
}

type Shortcuts []*Shortcut

type Shortcut struct {
	ID     int            `json:"id"`
	Client ClientType     `json:"client"`
	Action ShortcutAction `json:"action"`
}

type ShortcutAction struct {
	Device map[int]ActionDevice `json:"device"`
	Room   map[int]ActionRoom   `json:"room"`
}

type ActionDevice map[string]interface{}

type ActionRoom map[string]interface{}

type Modes []*Mode

type Mode struct {
	ID     string     `json:"id"`
	Action ModeAction `json:"action"`
}

type ModeAction struct {
	Device ActionDevice `json:"device"`
	Room   ActionRoom   `json:"room"`
}

type Timers []*Timer

type Timer struct {
	Action  TimerAction
	Client  ClientType             `json:"client"`
	Enabled bool                   `json:"enabled"`
	Time    map[string]interface{} `json:"time"`
	ID      int                    `json:"id"`
}

type TimerAction struct {
	Type     string
	Shortcut int
	Mode     string
	Action   ShortcutAction
}

type VinculumServices struct {
	FireAlarm map[string]interface{} `json:"fireAlarm"`
}

type State struct {
	Devices StateDevices `json:"devices"`
}

type StateDevices []*StateDevice

type StateDevice struct {
	ID       int64           `json:"id"`
	Services []*StateService `json:"services"`
}

type StateService struct {
	Name       string           `json:"name"`
	Address    string           `json:"addr"`
	Attributes []StateAttribute `json:"attributes"`
}

type StateAttribute struct {
	Name   string                `json:"name"`
	Values []StateAttributeValue `json:"values"`
}

type StateAttributeValue struct {
	Timestamp string            `json:"ts"`
	ValType   string            `json:"val_t"`
	Val       interface{}       `json:"val"`
	Props     map[string]string `json:"props"`
}
