package prime

import (
	"fmt"
	"strings"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

type SyncClient interface {
	SendReqRespFimp(cmdTopic, responseTopic string, reqMsg *fimpgo.FimpMessage, timeout int, autoSubscribe bool) (*fimpgo.FimpMessage, error)
}

type Client interface {
	GetDevices() (Devices, error)
	GetThings() (Things, error)
	GetRooms() (Rooms, error)
	GetAreas() (Areas, error)
	GetHouse() (*House, error)
	GetHub() (*Hub, error)
	GetShortcuts() (Shortcuts, error)
	GetModes() (Modes, error)
	GetTimers() (Timers, error)
	GetVinculumServices() (*VinculumServices, error)
	GetState() (*State, error)
	GetComponents(components ...string) (*ComponentSet, error)
	GetAll() (*ComponentSet, error)
	RunShortcut(shortcutID int) (*Response, error)
	ChangeMode(mode string) (*Response, error)
}

func NewClient(syncClient SyncClient, resourceName fimptype.ResourceNameT, defaultTimeout time.Duration) Client {
	responseAddress := fimpgo.Address{
		PayloadType:     fimpgo.DefaultPayload,
		MsgType:         fimptype.MsgTypeRsp,
		ResourceType:    fimptype.ResourceTypeApp,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}

	requestAddress := fimpgo.Address{
		PayloadType:     fimpgo.DefaultPayload,
		MsgType:         fimptype.MsgTypeCmd,
		ResourceType:    fimptype.ResourceTypeApp,
		ResourceName:    fimptype.VinculumRn,
		ResourceAddress: "1",
	}

	return &client{
		clientName:      resourceName.Str(),
		requestAddress:  requestAddress,
		responseAddress: responseAddress,
		syncClient:      syncClient,
		defaultTimeout:  int(defaultTimeout / time.Second),
	}
}

func NewCloudClient(syncClient SyncClient, cloudServiceName string, siteUUID string, defaultTimeout time.Duration) Client {
	responseAddress := fimpgo.Address{
		GlobalPrefix:    siteUUID,
		PayloadType:     fimpgo.DefaultPayload,
		MsgType:         fimptype.MsgTypeRsp,
		ResourceType:    fimptype.ResourceTypeCloud,
		ResourceName:    fimptype.BackendServiceRn,
		ResourceAddress: cloudServiceName,
	}

	requestAddress := fimpgo.Address{
		GlobalPrefix:    siteUUID,
		PayloadType:     fimpgo.DefaultPayload,
		MsgType:         fimptype.MsgTypeCmd,
		ResourceType:    fimptype.ResourceTypeApp,
		ResourceName:    fimptype.VinculumRn,
		ResourceAddress: "1",
	}

	return &client{
		clientName:      cloudServiceName,
		requestAddress:  requestAddress,
		responseAddress: responseAddress,
		syncClient:      syncClient,
		defaultTimeout:  int(defaultTimeout / time.Second),
	}
}

type client struct {
	clientName      string
	requestAddress  fimpgo.Address
	responseAddress fimpgo.Address
	syncClient      SyncClient
	defaultTimeout  int
}

func (c *client) GetDevices() (Devices, error) {
	response, err := c.sendGetRequest([]string{ComponentDevice})
	if err != nil {
		return nil, err
	}

	return response.GetDevices()
}

func (c *client) GetThings() (Things, error) {
	response, err := c.sendGetRequest([]string{ComponentThing})
	if err != nil {
		return nil, err
	}

	return response.GetThings()
}

func (c *client) GetRooms() (Rooms, error) {
	response, err := c.sendGetRequest([]string{ComponentRoom})
	if err != nil {
		return nil, err
	}

	return response.GetRooms()
}

func (c *client) GetAreas() (Areas, error) {
	response, err := c.sendGetRequest([]string{ComponentArea})
	if err != nil {
		return nil, err
	}

	return response.GetAreas()
}

func (c *client) GetHouse() (*House, error) {
	response, err := c.sendGetRequest([]string{ComponentHouse})
	if err != nil {
		return nil, err
	}

	return response.GetHouse()
}

func (c *client) GetHub() (*Hub, error) {
	response, err := c.sendGetRequest([]string{ComponentHub})
	if err != nil {
		return nil, err
	}

	return response.GetHub()
}

func (c *client) GetShortcuts() (Shortcuts, error) {
	response, err := c.sendGetRequest([]string{ComponentShortcut})
	if err != nil {
		return nil, err
	}

	return response.GetShortcuts()
}

func (c *client) GetModes() (Modes, error) {
	response, err := c.sendGetRequest([]string{ComponentMode})
	if err != nil {
		return nil, err
	}

	return response.GetModes()
}

func (c *client) GetTimers() (Timers, error) {
	response, err := c.sendGetRequest([]string{ComponentTimer})
	if err != nil {
		return nil, err
	}

	return response.GetTimers()
}

func (c *client) GetVinculumServices() (*VinculumServices, error) {
	response, err := c.sendGetRequest([]string{ComponentService})
	if err != nil {
		return nil, err
	}

	return response.GetVinculumServices()
}

func (c *client) GetState() (*State, error) {
	response, err := c.sendGetRequest([]string{ComponentState})
	if err != nil {
		return nil, err
	}

	return response.GetState()
}

func (c *client) GetComponents(components ...string) (*ComponentSet, error) {
	err := c.validateComponents(components)
	if err != nil {
		return nil, err
	}

	response, err := c.sendGetRequest(components)
	if err != nil {
		return nil, err
	}

	return response.GetAll()
}

func (c *client) GetAll() (*ComponentSet, error) {
	return c.GetComponents(validComponents...)
}

func (c *client) RunShortcut(shortcutID int) (*Response, error) {
	response, err := c.sendSetRequest(ComponentShortcut, shortcutID)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *client) ChangeMode(mode string) (*Response, error) {
	response, err := c.sendSetRequest(ComponentMode, mode)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *client) sendGetRequest(components []string) (*Response, error) {
	request := &Request{
		Cmd: CmdGet,
		Param: &RequestParam{
			Components: components,
		},
	}

	message := fimpgo.NewObjectMessage(CmdPD7Request, "vinculum", request, nil, nil, nil)
	message.ResponseToTopic = c.responseAddress.Serialize()
	message.Source = fimptype.ResourceNameT(c.clientName)

	response, err := c.syncClient.SendReqRespFimp(c.requestAddress.Serialize(), c.responseAddress.Serialize(), message, c.defaultTimeout, true)
	if err != nil {
		return nil, fmt.Errorf("prime client: error while sending get request for components %s: %w", strings.Join(components, ", "), err)
	}

	primeResponse, err := ResponseFromMessage(response)
	if err != nil {
		return nil, fmt.Errorf("prime client: error while parsing get response for components %s: %w", strings.Join(components, ", "), err)
	}

	return primeResponse, nil
}

func (c *client) sendSetRequest(component string, value interface{}) (*Response, error) {
	request := &Request{
		Cmd:       CmdSet,
		Component: component,
		ID:        value,
	}

	message := fimpgo.NewObjectMessage(CmdPD7Request, "vinculum", request, nil, nil, nil)
	message.ResponseToTopic = c.responseAddress.Serialize()
	message.Source = fimptype.ResourceNameT(c.clientName)

	response, err := c.syncClient.SendReqRespFimp(c.requestAddress.Serialize(), c.responseAddress.Serialize(), message, c.defaultTimeout, true)
	if err != nil {
		return nil, fmt.Errorf("prime client: error while sending set request for component %s: %w", component, err)
	}

	primeResponse, err := ResponseFromMessage(response)
	if err != nil {
		return nil, fmt.Errorf("prime client: error while parsing set response for component %s: %w", component, err)
	}

	return primeResponse, nil
}

func (c *client) validateComponents(given []string) error {
	for _, component := range given {
		if err := c.validateComponent(component); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) validateComponent(component string) error {
	for _, c := range validComponents {
		if c == component {
			return nil
		}
	}

	return fmt.Errorf("prime client: invalid component %s", component)
}
