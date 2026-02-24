package prime

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/futurehomeno/fimpgo"
)

const (
	CmdPD7Request  = "cmd.pd7.request"
	EvtPD7Response = "evt.pd7.response"
	EvtPD7Notify   = "evt.pd7.notify"

	NotifyTopic = "pt:j1/mt:evt/rt:app/rn:vinculum/ad:1"
)

type Request struct {
	Cmd       string        `json:"cmd"`
	Component any           `json:"component"`
	Param     *RequestParam `json:"param,omitempty"`
	RequestID any           `json:"requestId,omitempty"`
	ID        any           `json:"id,omitempty"`
}

type RequestParam struct {
	ID         int      `json:"id,omitempty"`
	Components []string `json:"components,omitempty"`
}

type Response struct {
	Errors    any                        `json:"errors"`
	Cmd       string                     `json:"cmd"`
	ParamRaw  map[string]json.RawMessage `json:"param"`
	RequestID any                        `json:"requestId"`
	Success   bool                       `json:"success"`
	ID        any                        `json:"id,omitempty"`
}

func ResponseFromMessage(msg *fimpgo.FimpMessage) (*Response, error) {
	response := &Response{}

	if err := msg.GetObjectValue(response); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal message value: %w", err)
	}

	return response, nil
}

func (r *Response) GetDevices() (Devices, error) {
	param, ok := r.ParamRaw[ComponentDevice]
	if !ok {
		return nil, nil
	}

	var result Devices

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentDevice, err)
	}

	return result, nil
}

func (r *Response) GetThings() (Things, error) {
	param, ok := r.ParamRaw[ComponentThing]
	if !ok {
		return nil, nil
	}

	var result Things

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentThing, err)
	}

	return result, nil
}

func (r *Response) GetAreas() (Areas, error) {
	param, ok := r.ParamRaw[ComponentArea]
	if !ok {
		return nil, nil
	}

	var result Areas

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentArea, err)
	}

	return result, nil
}

func (r *Response) GetRooms() (Rooms, error) {
	param, ok := r.ParamRaw[ComponentRoom]
	if !ok {
		return nil, nil
	}

	var result Rooms

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentRoom, err)
	}

	return result, nil
}

func (r *Response) GetHouse() (*House, error) {
	param, ok := r.ParamRaw[ComponentHouse]
	if !ok {
		return nil, nil
	}

	result := &House{}

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentHouse, err)
	}

	return result, nil
}

func (r *Response) GetHub() (*Hub, error) {
	param, ok := r.ParamRaw[ComponentHub]
	if !ok {
		return nil, nil
	}

	result := &Hub{}

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentHub, err)
	}

	return result, nil
}

func (r *Response) GetShortcuts() (Shortcuts, error) {
	param, ok := r.ParamRaw[ComponentShortcut]
	if !ok {
		return nil, nil
	}

	var result Shortcuts

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentShortcut, err)
	}

	return result, nil
}

func (r *Response) GetModes() (Modes, error) {
	param, ok := r.ParamRaw[ComponentMode]
	if !ok {
		return nil, nil
	}

	var result Modes

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentMode, err)
	}

	return result, nil
}

func (r *Response) GetTimers() (Timers, error) {
	param, ok := r.ParamRaw[ComponentTimer]
	if !ok {
		return nil, nil
	}

	var result Timers

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentTimer, err)
	}

	return result, nil
}

func (r *Response) GetVinculumServices() (*VinculumServices, error) {
	param, ok := r.ParamRaw[ComponentService]
	if !ok {
		return nil, nil
	}

	result := &VinculumServices{}

	if err := json.Unmarshal(param, result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentService, err)
	}

	return result, nil
}

func (r *Response) GetState() (*State, error) {
	param, ok := r.ParamRaw[ComponentState]
	if !ok {
		return nil, nil
	}

	result := &State{}

	if err := json.Unmarshal(param, &result); err != nil {
		return nil, fmt.Errorf("response: failed to unmarshal component %s: %w", ComponentState, err)
	}

	return result, nil
}

//nolint:funlen,cyclop
func (r *Response) GetAll() (*ComponentSet, error) {
	devices, err := r.GetDevices()
	if err != nil {
		return nil, err
	}

	things, err := r.GetThings()
	if err != nil {
		return nil, err
	}

	rooms, err := r.GetRooms()
	if err != nil {
		return nil, err
	}

	areas, err := r.GetAreas()
	if err != nil {
		return nil, err
	}

	house, err := r.GetHouse()
	if err != nil {
		return nil, err
	}

	hub, err := r.GetHub()
	if err != nil {
		return nil, err
	}

	shortcuts, err := r.GetShortcuts()
	if err != nil {
		return nil, err
	}

	modes, err := r.GetModes()
	if err != nil {
		return nil, err
	}

	timers, err := r.GetTimers()
	if err != nil {
		return nil, err
	}

	services, err := r.GetVinculumServices()
	if err != nil {
		return nil, err
	}

	state, err := r.GetState()
	if err != nil {
		return nil, err
	}

	result := &ComponentSet{
		Devices:   devices,
		Things:    things,
		Rooms:     rooms,
		Areas:     areas,
		House:     house,
		Hub:       hub,
		Shortcuts: shortcuts,
		Modes:     modes,
		Timers:    timers,
		Services:  services,
		State:     state,
	}

	return result, nil
}

type Notify struct {
	Errors     any             `json:"errors"`
	Cmd        string          `json:"cmd"`
	Component  string          `json:"component"`
	ParamRaw   json.RawMessage `json:"param"`
	ChangesRaw json.RawMessage `json:"changes"`
	Success    bool            `json:"success"`
	ID         any             `json:"id,omitempty"`
}

func NotifyFromMessage(msg *fimpgo.Message) (*Notify, error) {
	notify := &Notify{}

	if err := msg.Payload.GetObjectValue(notify); err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal message value: %w", err)
	}

	return notify, nil
}

func (n *Notify) ParseIntegerID() (int, error) {
	if n.ID == nil {
		return 0, fmt.Errorf("notify: id is nil")
	}

	switch id := n.ID.(type) {
	case int:
		return id, nil
	case int64:
		return int(id), nil
	case float64:
		return int(id), nil
	case string:
		return strconv.Atoi(id)
	default:
		return 0, fmt.Errorf("notify: id is of unsupported type %T", id)
	}
}

func (n *Notify) GetDevice() (*Device, error) {
	if n.Component != ComponentDevice {
		return nil, nil
	}

	result := &Device{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentDevice, err)
	}

	return result, nil
}

func (n *Notify) GetThing() (*Thing, error) {
	if n.Component != ComponentThing {
		return nil, nil
	}

	result := &Thing{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentThing, err)
	}

	return result, nil
}

func (n *Notify) GetRoom() (*Room, error) {
	if n.Component != ComponentRoom {
		return nil, nil
	}

	result := &Room{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentRoom, err)
	}

	return result, nil
}

func (n *Notify) GetArea() (*Area, error) {
	if n.Component != ComponentArea {
		return nil, nil
	}

	result := &Area{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentArea, err)
	}

	return result, nil
}

func (n *Notify) GetHouse() (*House, error) {
	if n.Component != ComponentHouse {
		return nil, nil
	}

	result := &House{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentHouse, err)
	}

	return result, nil
}

func (n *Notify) GetHub() (*Hub, error) {
	if n.Component != ComponentHub {
		return nil, nil
	}

	result := &Hub{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentHub, err)
	}

	return result, nil
}

func (n *Notify) GetHubMode() (*HubMode, error) {
	if n.Component != ComponentHub || n.ID != "mode" {
		return nil, nil
	}

	result := &HubMode{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentHub, err)
	}

	return result, nil
}

func (n *Notify) GetShortcut() (*Shortcut, error) {
	if n.Component != ComponentShortcut {
		return nil, nil
	}

	result := &Shortcut{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentShortcut, err)
	}

	return result, nil
}

func (n *Notify) GetMode() (*Mode, error) {
	if n.Component != ComponentMode {
		return nil, nil
	}

	result := &Mode{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentMode, err)
	}

	return result, nil
}

func (n *Notify) GetTimer() (*Timer, error) {
	if n.Component != ComponentTimer {
		return nil, nil
	}

	result := &Timer{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentTimer, err)
	}

	return result, nil
}

func (n *Notify) GetService() (*VinculumServices, error) {
	if n.Component != ComponentService {
		return nil, nil
	}

	result := &VinculumServices{}

	err := json.Unmarshal(n.ParamRaw, result)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to unmarshal component %s: %w", ComponentService, err)
	}

	return result, nil
}
