package damon

import (
	"fmt"
	"github.com/QQGoblin/damonctl/pkg/utils"
)

var defaultPaths = newSysfsPath(defaultDamonRoot)

type Kdamon struct {
	slotID int
	paths  PathBuilder
}

type SlotInfo struct {
	ID         int
	State      string
	KdamondPid int
}

func NewKdamon(slotID int) *Kdamon {
	return &Kdamon{slotID: slotID, paths: defaultPaths}
}

func Init(nrSlots int) error {
	return utils.WriteInt(defaultPaths.NrKdamonds(), nrSlots)
}

func ReadNrKdamonds() (int, error) {
	return utils.ReadInt(defaultPaths.NrKdamonds())
}

func FindFreeSlot() (int, error) {
	nr, err := ReadNrKdamonds()
	if err != nil {
		return -1, fmt.Errorf("read nr_kdamonds: %w", err)
	}
	if nr == 0 {
		return -1, fmt.Errorf("no kdamond slots available, run 'init' first")
	}

	for i := 0; i < nr; i++ {
		kd := NewKdamon(i)
		running, err := kd.IsRunning()
		if err != nil {
			return -1, fmt.Errorf("check kdamond %d state: %w", i, err)
		}
		if !running {
			return i, nil
		}
	}
	return -1, fmt.Errorf("all %d kdamond slots are in use", nr)
}

func ListSlots() ([]SlotInfo, error) {
	nr, err := ReadNrKdamonds()
	if err != nil {
		return nil, fmt.Errorf("read nr_kdamonds: %w", err)
	}

	slots := make([]SlotInfo, nr)
	for i := 0; i < nr; i++ {
		kd := NewKdamon(i)
		state, err := kd.ReadState()
		if err != nil {
			state = "unknown"
		}
		var kdamondPid int
		if state == "on" {
			kdamondPid, _ = kd.ReadPid()
		}
		slots[i] = SlotInfo{
			ID:         i,
			State:      state,
			KdamondPid: kdamondPid,
		}
	}
	return slots, nil
}
