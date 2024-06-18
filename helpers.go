package main

import (
	"fmt"
	"math"
	"sort"
	"syscall"
	"time"
)

// HMonitor handle
type HMonitor syscall.Handle

// MonitorEnumProc callback function type
type MonitorEnumProc func(hMonitor HMonitor, hdc syscall.Handle, lprcMonitor *Rect, dwData uintptr) uintptr

// EnumDisplayMonitors function wrapper
func EnumDisplayMonitors(lpfnEnum MonitorEnumProc, dwData uintptr) bool {
	callback := syscall.NewCallback(lpfnEnum)
	ret, _, _ := enumDisplayMonitors.Call(
		0,
		0,
		callback,
		dwData,
	)
	return ret != 0
}

// Callback function to enumerate monitors
func monitorEnumProc(hMonitor HMonitor, hdc syscall.Handle, lprcMonitor *Rect, dwData uintptr) uintptr {
	fmt.Printf("Monitor handle: %v\n", hMonitor)
	fmt.Printf("Monitor Rect: %+v\n", *lprcMonitor)
	return 1 // Continue enumeration
}

// When a monitor is turned off, the order can change
// even though the monitors in settings keep the same order
func EnumDisplayMonitorsF() {
	// Call EnumDisplayMonitors with our callback function
	success := EnumDisplayMonitors(monitorEnumProc, 0)
	if !success {
		fmt.Println("EnumDisplayMonitors failed")
	} else {
		fmt.Println("EnumDisplayMonitors succeeded")
	}
}

// We don't care about the monitor itself, we just need its bounding boxes to move into
// Some kind of strangeness goes on with these pointers we pass in in the call back. Not sure why but they don't change in the function we call from
// Need to think of an alternative. We can set the mouse directly from this function, but not to the first monitor
// I wonder if channels work
func nextMonitorEnumCallback(currentMonitorID HMonitor, rectChannel chan Rect) func(hMonitor HMonitor, hdc syscall.Handle, lprcMonitor *Rect, dwData uintptr) uintptr {
	foundFirst := false
	foundCurrent := false
	return func(hMonitor HMonitor, hdc syscall.Handle, lprcMonitor *Rect, dwData uintptr) uintptr {
		if !foundFirst {
			foundFirst = true
			rectChannel <- *lprcMonitor
		}
		if foundCurrent {
			rectChannel <- *lprcMonitor
			return 0
		}
		if hMonitor == currentMonitorID {
			foundCurrent = true
			return 1
		}
		return 1
	}
}

func SwitchToNextMonitor() {
	monitorID := GetCurrentMouseMonitor()
	var nextMonitor Rect
	// Make the channel of size 2, it will first send the first monitor, and then possible the second
	// I do think this runs synch overall.... Pointers are probably better because its easy to see if set
	// But maybe doens't matter because its a channel and it will have a set length
	screenBoundsChannel := make(chan Rect, 2)

	EnumDisplayMonitors(nextMonitorEnumCallback(monitorID, screenBoundsChannel), 0)
	if len(screenBoundsChannel) == 1 {
		// Set next monitor to first monitor if we only recieve one screen
		nextMonitor = <-screenBoundsChannel
	} else {
		// Remove the first value, then get the first
		<-screenBoundsChannel
		nextMonitor = <-screenBoundsChannel
	}
	MoveMouseToCenterOfMonitor(nextMonitor)
}

func MoveMouseToCenterOfMonitor(monitor Rect) {
	var p Point
	// This shouldn't overflow in any foreseeable future unless people start having billion pixel wide screens
	p.X = (monitor.Left + monitor.Right) / 2
	p.Y = (monitor.Top + monitor.Bottom) / 2
	setCursorPosition(p)
}

// Return which monitor we are currently inside
func GetCurrentMouseMonitor() (monitorID HMonitor) {
	mousePos, err := getCursorPos()
	if err != nil {
		fmt.Println(err)
	}

	monitorID = getMonitorFromPoint(mousePos)
	return
}

func nextMonitorClickWiseEnumCallback(monitorChannel chan MonitorInfoBlock) func(hMonitor HMonitor, hdc syscall.Handle, lprcMonitor *Rect, dwData uintptr) uintptr {
	return func(hMonitor HMonitor, hdc syscall.Handle, lprcMonitor *Rect, dwData uintptr) uintptr {
		monitorToSend := MonitorInfoBlock{
			MonitorID:   hMonitor,
			BoundingBox: *lprcMonitor,
		}
		monitorChannel <- monitorToSend
		return 1
	}
}

// Contains the Monitor ID, and its bounding Box
type MonitorInfoBlock struct {
	MonitorID   HMonitor
	BoundingBox Rect
	Center      Point
}

func SwitchToNextMonitorClockWise() {
	currentMonitorID := GetCurrentMouseMonitor()
	// Make the channel of size 2, it will first send the first monitor, and then possible the second
	// I do think this runs synch overall.... Pointers are probably better because its easy to see if set
	// But maybe doens't matter because its a channel and it will have a set length
	monitorChannel := make(chan MonitorInfoBlock)
	monitorArray := make([]MonitorInfoBlock, 0, 1)
	// Turn the monitor channel into monitor array
	go func() {
		for {
			monitor, more := <-monitorChannel
			if !more {
				return
			}
			monitor.Center = Point{X: (monitor.BoundingBox.Left + monitor.BoundingBox.Right) / 2, Y: (monitor.BoundingBox.Top + monitor.BoundingBox.Bottom) / 2}
			monitorArray = append(monitorArray, monitor)
		}

	}()
	EnumDisplayMonitors(nextMonitorClickWiseEnumCallback(monitorChannel), 0)
	time.Sleep(time.Millisecond * 100) // Without this, sometimes I only get my first two monitors. 
	close(monitorChannel)
	monitorArray = sortBoxesClockwise(monitorArray)
	found := false
	var nextBox *Rect
	for x := range monitorArray {
		if monitorArray[x].MonitorID == currentMonitorID {
			found = true
			continue
		}
		if found {
			nextBox = &monitorArray[x].BoundingBox
			break
		}
	}
	if nextBox == nil {
		nextBox = &monitorArray[0].BoundingBox
	}
	MoveMouseToCenterOfMonitor(*nextBox)
}

func sortBoxesClockwise(boxes []MonitorInfoBlock) []MonitorInfoBlock {
	// Find the center of all boxes
	var totalX, totalY float64
	for _, box := range boxes {
		totalX += float64(box.Center.X)
		totalY += float64(box.Center.Y)
	}
	centerX := totalX / float64(len(boxes))
	centerY := totalY / float64(len(boxes))

	// Sort boxes based on angle from center point
	sort.Slice(boxes, func(i, j int) bool {
		p1 := boxes[i].Center
		p2 := boxes[j].Center
		angle1 := math.Atan2((float64(p1.Y) - centerY), (float64(p1.X) - centerX))
		angle2 := math.Atan2((float64(p2.Y) - centerY), (float64(p2.X) - centerX))

		return angle1 < angle2
	})

	return boxes
}
