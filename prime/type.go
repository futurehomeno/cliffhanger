package prime

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/futurehomeno/fimpgo"
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

	TypeChargepoint   = "chargepoint"
	TypeInverter      = "inverter"
	TypeEnergyStorage = "energy_storage"
	TypeBoiler        = "boiler"
	TypeHeatPump      = "heat_pump"
	TypeThermostat    = "thermostat"
	TypeFan           = "fan"
	TypeDoorLock      = "door_lock"
	TypeMediaPlayer   = "media_player"
	TypeLight         = "light"
	TypeBlinds        = "blinds"
	TypeGarageDoor    = "garage_door"
	TypeGate          = "gate"
	TypeFireDetector  = "fire_detector"
	TypeGasDetector   = "gas_detector"
	TypeWaterValve    = "water_valve"
	TypeLeakDetector  = "leak_detector"
	TypeSiren         = "siren"
	TypeAppliance     = "appliance"
	TypeHeater        = "heater"
	TypeMeter         = "meter"
	TypeSensor        = "sensor"
	TypeHeatDetector  = "heat_detector"
	TypeInput         = "input"
	TypeBattery       = "battery"

	SubTypeCarCharger = "car_charger"
	SubTypeInverter   = "inverter"
	SubTypeMainElec   = "main_elec"
	SubTypeDoor       = "door"
	SubTypeDoorLock   = "door_lock"
	SubTypeGarage     = "garage"
	SubTypeLock       = "lock"
	SubTypeOther      = "other"
	SubTypeWindow     = "window"
	SubTypeWindowLock = "window_lock"
	SubTypePresence   = "presence"
	SubTypeScene      = "scene"
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
	Devices   Devices           `json:"attribute,omitempty"`
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

func (d Devices) FilterByThingID(thingID int) Devices {
	if thingID == 0 {
		return nil
	}

	var devices Devices

	for _, device := range d {
		if device.GetThingID() != thingID {
			continue
		}

		devices = append(devices, device)
	}

	return devices
}

func (d Devices) FilterByIDs(id ...int) Devices {
	var devices Devices

	for _, device := range d {
		for _, i := range id {
			if device.ID == i {
				devices = append(devices, device)
			}
		}
	}

	return devices
}

func (d Devices) FindByID(id int) *Device {
	for _, device := range d {
		if device.ID == id {
			return device
		}
	}

	return nil
}

func (d Devices) FindByTopic(topic string) *Device {
	for _, device := range d {
		if device.MatchesTopic(topic) {
			return device
		}
	}

	return nil
}

type Device struct {
	FIMP          DeviceFIMP             `json:"fimp"`
	Client        ClientType             `json:"client"`
	Functionality *string                `json:"functionality"`
	Services      map[string]*Service    `json:"services"`
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

func (d *Device) GetName() string {
	if d.Client.Name != nil {
		return *d.Client.Name
	}

	if d.ModelAlias != "" {
		return d.ModelAlias
	}

	return d.Model
}

func (d *Device) GetThingID() int {
	if d.ThingID == nil {
		return 0
	}

	return *d.ThingID
}

func (d *Device) GetRoomID() int {
	if d.Room == nil {
		return 0
	}

	return *d.Room
}

func (d *Device) GetType() string {
	v, ok := d.Type["type"]
	if !ok {
		return ""
	}

	s, ok := v.(string)
	if !ok {
		return ""
	}

	return s
}

func (d *Device) GetSubType() string {
	v, ok := d.Type["subtype"]
	if !ok {
		return ""
	}

	s, ok := v.(string)
	if !ok {
		return ""
	}

	return s
}

func (d *Device) SupportsSubType(mainType, subType string) bool {
	supportedRaw, ok := d.Type["supported"]
	if !ok {
		return false
	}

	supported, ok := supportedRaw.(map[string]interface{})
	if !ok {
		return false
	}

	subTypesRaw, ok := supported[mainType]
	if !ok {
		return false
	}

	subTypes, ok := subTypesRaw.([]interface{})
	if !ok {
		return false
	}

	for _, v := range subTypes {
		s, ok := v.(string)
		if !ok {
			continue
		}

		if s == subType {
			return true
		}
	}

	return false
}

func (d *Device) HasService(serviceName string) bool {
	return d.GetService(serviceName) != nil
}

func (d *Device) GetService(serviceName string) *Service {
	for srvName, srv := range d.Services {
		if srvName == serviceName {
			return srv
		}
	}

	return nil
}

func (d *Device) HasInterfaces(serviceName string, interfaceNames ...string) bool {
	srv := d.GetService(serviceName)
	if srv == nil {
		return false
	}

	for _, interfaceName := range interfaceNames {
		if !d.containsInterface(interfaceName, srv.Interfaces) {
			return false
		}
	}

	return true
}

func (d *Device) containsInterface(interfaceName string, interfaces []string) bool {
	for _, i := range interfaces {
		if i == interfaceName {
			return true
		}
	}

	return false
}

func (d *Device) GetServiceProperty(serviceName string, property string) interface{} {
	srv := d.GetService(serviceName)
	if srv == nil {
		return nil
	}

	value, ok := srv.Props[property]
	if !ok {
		return nil
	}

	return value
}

func (d *Device) GetServicePropertyString(serviceName string, property string) string {
	var val string

	ok := d.GetServicePropertyObject(serviceName, property, &val)
	if !ok {
		return ""
	}

	return val
}

func (d *Device) GetServicePropertyStrings(serviceName, property string) []string {
	var val []string

	ok := d.GetServicePropertyObject(serviceName, property, &val)
	if !ok {
		return nil
	}

	return val
}

func (d *Device) GetServicePropertyInteger(serviceName, property string) int {
	var val int

	ok := d.GetServicePropertyObject(serviceName, property, &val)
	if !ok {
		return 0
	}

	return val
}

func (d *Device) GetServicePropertyObject(serviceName, property string, object interface{}) (ok bool) {
	v := d.GetServiceProperty(serviceName, property)
	if v == nil {
		return false
	}

	b, err := json.Marshal(v)
	if err != nil {
		return false
	}

	err = json.Unmarshal(b, object)

	return err == nil
}

func (d *Device) MatchesTopic(topic string) bool {
	for _, srv := range d.Services {
		if strings.Contains(topic, srv.Addr) {
			return true
		}
	}

	return false
}

func (d *Device) GetAddresses() []string {
	var addresses []string

	for _, srv := range d.Services {
		addresses = append(addresses, srv.Addr)
	}

	sort.Strings(addresses)

	return addresses
}

type Service struct {
	Addr       string                 `json:"addr,omitempty"`
	Enabled    bool                   `json:"enabled,omitempty"`
	Interfaces []string               `json:"intf"`
	Props      map[string]interface{} `json:"props"`
}

type DeviceFIMP struct {
	Adapter         string `json:"adapter"`
	Address         string `json:"address"`
	AdapterResource string `json:"adapter_resource"`
	AdapterAddress  string `json:"adapter_address"`
	Group           string `json:"group"`
}

type ClientType struct {
	Name          *string `json:"name,omitempty"`
	OpenStateType *string `json:"openStateType,omitempty"`
}

type Things []*Thing

func (t Things) FindByID(id int) *Thing {
	for _, thing := range t {
		if thing.ID == id {
			return thing
		}
	}

	return nil
}

type Thing struct {
	ID      int                    `json:"id"`
	FIMP    ThingFIMP              `json:"fimp"`
	Address string                 `json:"addr"`
	Name    string                 `json:"name"`
	Devices []int                  `json:"devices,omitempty"`
	Props   map[string]interface{} `json:"props,omitempty"`
	RoomID  int                    `json:"room"`
}

type ThingFIMP struct {
	Address         string `json:"address"`
	AdapterService  string `json:"adapter_service"`
	AdapterResource string `json:"adapter_resource"`
	AdapterAddress  string `json:"adapter_address"`
}

type Rooms []*Room

func (r Rooms) FindByID(id int) *Room {
	for _, room := range r {
		if room.ID == id {
			return room
		}
	}

	return nil
}

type Room struct {
	Alias   string     `json:"alias"`
	ID      int        `json:"id"`
	Param   RoomParams `json:"param"`
	Client  ClientType `json:"client"`
	Type    *string    `json:"type"`
	Area    *int       `json:"area"`
	Outside bool       `json:"outside"`
}

func (d *Room) GetAreaID() int {
	if d.Area == nil {
		return 0
	}

	return *d.Area
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
	Device map[int]ActionDevice `json:"attribute"`
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
	Device ActionDevice `json:"attribute"`
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

func (d StateDevices) FindDevice(id int) *StateDevice {
	for _, device := range d {
		if device.ID == id {
			return device
		}
	}

	return nil
}

type StateDevice struct {
	ID       int             `json:"id"`
	Services []*StateService `json:"services"`
}

func (d *StateDevice) GetAttributeStringValue(serviceName, attributeName string, properties map[string]string) (string, time.Time) {
	var val string

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeIntValue(serviceName, attributeName string, properties map[string]string) (int, time.Time) {
	var val int

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeFloatValue(serviceName, attributeName string, properties map[string]string) (float64, time.Time) {
	var val float64

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeBoolValue(serviceName, attributeName string, properties map[string]string) (bool, time.Time) {
	var val bool

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeStringArrayValue(serviceName, attributeName string, properties map[string]string) ([]string, time.Time) {
	var val []string

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeIntArrayValue(serviceName, attributeName string, properties map[string]string) ([]int, time.Time) {
	var val []int

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeFloatArrayValue(serviceName, attributeName string, properties map[string]string) ([]float64, time.Time) {
	var val []float64

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeBoolArrayValue(serviceName, attributeName string, properties map[string]string) ([]bool, time.Time) {
	var val []bool

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeStringMapValue(serviceName, attributeName string, properties map[string]string) (map[string]string, time.Time) {
	var val map[string]string

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeIntMapValue(serviceName, attributeName string, properties map[string]string) (map[string]int, time.Time) {
	var val map[string]int

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeFloatMapValue(serviceName, attributeName string, properties map[string]string) (map[string]float64, time.Time) {
	var val map[string]float64

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeBoolMapValue(serviceName, attributeName string, properties map[string]string) (map[string]bool, time.Time) {
	var val map[string]bool

	return val, d.GetAttributeObjectValue(serviceName, attributeName, properties, &val)
}

func (d *StateDevice) GetAttributeObjectValue(serviceName, attributeName string, properties map[string]string, object interface{}) time.Time {
	if d == nil {
		return time.Time{}
	}

	value := d.FindAttributeValue(serviceName, attributeName, properties)
	if value == nil {
		return time.Time{}
	}

	err := value.GetObjectValue(object)
	if err != nil {
		return time.Time{}
	}

	t, err := value.GetTime()
	if err != nil {
		return time.Time{}
	}

	return t
}

func (d *StateDevice) FindAttributeValue(serviceName, attributeName string, properties map[string]string) *StateAttributeValue {
	if d == nil {
		return nil
	}

	service := d.FindService(serviceName)
	if service == nil {
		return nil
	}

	attribute := service.FindAttribute(attributeName)
	if attribute == nil {
		return nil
	}

	return attribute.FindValue(properties)
}

func (d *StateDevice) FindService(name string) *StateService {
	if d == nil {
		return nil
	}

	for _, s := range d.Services {
		if s.Name == name {
			return s
		}
	}

	return nil
}

type StateService struct {
	Name       string            `json:"name"`
	Address    string            `json:"addr"`
	Attributes []*StateAttribute `json:"attributes"`
}

func (s *StateService) FindAttribute(name string) *StateAttribute {
	if s == nil {
		return nil
	}

	segments := strings.Split(name, ".")
	if len(segments) == 3 {
		name = segments[1]
	}

	for _, a := range s.Attributes {
		if a.Name == name {
			return a
		}
	}

	return nil
}

type StateAttribute struct {
	Name   string                 `json:"name"`
	Values []*StateAttributeValue `json:"values"`
}

func (a *StateAttribute) FindValue(properties map[string]string) *StateAttributeValue {
	if a == nil || len(a.Values) == 0 {
		return nil
	}

	for _, v := range a.Values {
		if v.HasProperties(properties) {
			return v
		}
	}

	return nil
}

type StateAttributeValue struct {
	Timestamp string            `json:"ts"`
	ValueType string            `json:"val_t"`
	Value     interface{}       `json:"val"`
	Props     map[string]string `json:"props"`
}

func (v *StateAttributeValue) GetStringValue() (string, error) {
	var val string

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetIntValue() (int, error) {
	var val int

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetFloatValue() (float64, error) {
	var val float64

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetBoolValue() (bool, error) {
	var val bool

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetStringArrayValue() ([]string, error) {
	var val []string

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetIntArrayValue() ([]int, error) {
	var val []int

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetFloatArrayValue() ([]float64, error) {
	var val []float64

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetBoolArrayValue() ([]bool, error) {
	var val []bool

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetStringMapValue() (map[string]string, error) {
	var val map[string]string

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetIntMapValue() (map[string]int, error) {
	var val map[string]int

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetFloatMapValue() (map[string]float64, error) {
	var val map[string]float64

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetBoolMapValue() (map[string]bool, error) {
	var val map[string]bool

	return val, v.GetObjectValue(&val)
}

func (v *StateAttributeValue) GetObjectValue(object interface{}) error {
	if v == nil {
		return nil
	}

	b, err := json.Marshal(v.Value)
	if err != nil {
		return fmt.Errorf("state: failed to marshal value: %w", err)
	}

	err = json.Unmarshal(b, object)
	if err != nil {
		return fmt.Errorf("state: failed to unmarshal value: %w", err)
	}

	return nil
}

func (v *StateAttributeValue) GetTime() (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}

	t := fimpgo.ParseTime(v.Timestamp)
	if t.IsZero() {
		return time.Time{}, fmt.Errorf("state: failed to parse timestamp: %s", v.Timestamp)
	}

	return t, nil
}

func (v *StateAttributeValue) HasProperties(properties map[string]string) bool {
	for property, value := range properties {
		if !v.HasProperty(property, value) {
			return false
		}
	}

	return true
}

func (v *StateAttributeValue) HasProperty(property, value string) bool {
	if v == nil {
		return false
	}

	return v.Props[property] == value
}

func (v *StateAttributeValue) GetPropertyString(property string) string {
	if v == nil {
		return ""
	}

	return v.Props[property]
}

func (v *StateAttributeValue) GetPropertyInteger(property string) int {
	if v == nil {
		return 0
	}

	p, ok := v.Props[property]
	if !ok {
		return 0
	}

	i, err := strconv.ParseInt(p, 10, 64)
	if err != nil {
		return 0
	}

	return int(i)
}

func (v *StateAttributeValue) GetPropertyFloat(property string) float64 {
	if v == nil {
		return 0
	}

	p, ok := v.Props[property]
	if !ok {
		return 0
	}

	i, err := strconv.ParseFloat(p, 64)
	if err != nil {
		return 0
	}

	return i
}

func (v *StateAttributeValue) GetPropertyTimestamp(property string) time.Time {
	if v == nil {
		return time.Time{}
	}

	p, ok := v.Props[property]
	if !ok {
		return time.Time{}
	}

	return fimpgo.ParseTime(p)
}
