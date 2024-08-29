package observer

import (
	"github.com/futurehomeno/cliffhanger/prime"
)

type set struct {
	*prime.ComponentSet
}

func newSet(componentSet *prime.ComponentSet) *set {
	if componentSet == nil {
		componentSet = &prime.ComponentSet{}
	}

	return &set{ComponentSet: componentSet}
}

func (s *set) getDevices() prime.Devices {
	devices := make(prime.Devices, len(s.Devices))
	copy(devices, s.Devices)

	return devices
}

func (s *set) addDevice(device *prime.Device) {
	if s.findDevice(device.ID) != -1 {
		return
	}

	s.Devices = append(s.Devices, device)
}

func (s *set) updateDevice(device *prime.Device) {
	if i := s.findDevice(device.ID); i != -1 {
		s.Devices[i] = device

		return
	}

	s.Devices = append(s.Devices, device)
}

func (s *set) deleteDevice(id int) {
	if i := s.findDevice(id); i != -1 {
		s.Devices[i] = s.Devices[len(s.Devices)-1]
		s.Devices = s.Devices[:len(s.Devices)-1]
	}
}

func (s *set) findDevice(id int) int {
	for k, v := range s.Devices {
		if id == v.ID {
			return k
		}
	}

	return -1
}

func (s *set) getThings() prime.Things {
	things := make(prime.Things, len(s.Things))
	copy(things, s.Things)

	return things
}

func (s *set) addThing(thing *prime.Thing) {
	if s.findThing(thing.ID) != -1 {
		return
	}

	s.Things = append(s.Things, thing)
}

func (s *set) updateThing(thing *prime.Thing) {
	if i := s.findThing(thing.ID); i != -1 {
		s.Things[i] = thing

		return
	}

	s.Things = append(s.Things, thing)
}

func (s *set) deleteThing(id int) {
	if i := s.findThing(id); i != -1 {
		s.Things[i] = s.Things[len(s.Things)-1]
		s.Things = s.Things[:len(s.Things)-1]
	}
}

func (s *set) findThing(id int) int {
	for k, v := range s.Things {
		if id == v.ID {
			return k
		}
	}

	return -1
}

func (s *set) getRooms() prime.Rooms {
	rooms := make(prime.Rooms, len(s.Rooms))
	copy(rooms, s.Rooms)

	return rooms
}

func (s *set) addRoom(room *prime.Room) {
	if s.findRoom(room.ID) != -1 {
		return
	}

	s.Rooms = append(s.Rooms, room)
}

func (s *set) updateRoom(room *prime.Room) {
	if i := s.findRoom(room.ID); i != -1 {
		s.Rooms[i] = room

		return
	}

	s.Rooms = append(s.Rooms, room)
}

func (s *set) deleteRoom(id int) {
	if i := s.findRoom(id); i != -1 {
		s.Rooms[i] = s.Rooms[len(s.Rooms)-1]
		s.Rooms = s.Rooms[:len(s.Rooms)-1]
	}
}

func (s *set) findRoom(id int) int {
	for k, v := range s.Rooms {
		if id == v.ID {
			return k
		}
	}

	return -1
}

func (s *set) getAreas() prime.Areas {
	areas := make(prime.Areas, len(s.Areas))
	copy(areas, s.Areas)

	return areas
}

func (s *set) addArea(area *prime.Area) {
	if s.findArea(area.ID) != -1 {
		return
	}

	s.Areas = append(s.Areas, area)
}

func (s *set) updateArea(area *prime.Area) {
	if i := s.findArea(area.ID); i != -1 {
		s.Areas[i] = area

		return
	}

	s.Areas = append(s.Areas, area)
}

func (s *set) deleteArea(id int) {
	if i := s.findArea(id); i != -1 {
		s.Areas[i] = s.Areas[len(s.Areas)-1]
		s.Areas = s.Areas[:len(s.Areas)-1]
	}
}

func (s *set) findArea(id int) int {
	for k, v := range s.Areas {
		if id == v.ID {
			return k
		}
	}

	return -1
}

func (s *set) getShortcuts() prime.Shortcuts {
	shortcuts := make(prime.Shortcuts, len(s.Shortcuts))
	copy(shortcuts, s.Shortcuts)

	return shortcuts
}

func (s *set) addShortcut(shortcut *prime.Shortcut) {
	if s.findShortcut(shortcut.ID) != -1 {
		return
	}

	s.Shortcuts = append(s.Shortcuts, shortcut)
}

func (s *set) updateShortcut(shortcut *prime.Shortcut) {
	if i := s.findShortcut(shortcut.ID); i != -1 {
		s.Shortcuts[i] = shortcut

		return
	}

	s.Shortcuts = append(s.Shortcuts, shortcut)
}

func (s *set) deleteShortcut(id int) {
	if i := s.findShortcut(id); i != -1 {
		s.Shortcuts[i] = s.Shortcuts[len(s.Shortcuts)-1]
		s.Shortcuts = s.Shortcuts[:len(s.Shortcuts)-1]
	}
}

func (s *set) findShortcut(id int) int {
	for k, v := range s.Shortcuts {
		if id == v.ID {
			return k
		}
	}

	return -1
}
