package main

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

type Point struct {
	X, Y int32
}

type Rect struct {
	Left, Top, Right, Bottom int32
}

type MonitorInfo struct {
	CbSize    uint32
	RcMonitor Rect
	RcWork    Rect
	DwFlags   uint32
}

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	GetCursorPos        = user32.NewProc("GetCursorPos")
	MonitorFromPoint    = user32.NewProc("MonitorFromPoint")
	GetMonitorInfo      = user32.NewProc("GetMonitorInfoW")
	SetCursorPos        = user32.NewProc("SetCursorPos")
	enumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
)

// Confirmed this works nicely, thanks chatgpt
func getCursorPos() (Point, error) {
	var point Point
	r1, _, err := GetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	if r1 == 0 {
		return point, err
	}
	return point, nil
}

func getMonitorInfo(hMonitor syscall.Handle) (MonitorInfo, error) {
	var mi MonitorInfo
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	ret, _, err := GetMonitorInfo.Call(
		uintptr(hMonitor),
		uintptr(unsafe.Pointer(&mi)),
	)
	if ret == 0 {
		return mi, err
	}
	return mi, nil
}

func getMonitorFromPoint(point Point) (monitorHandle HMonitor) {
	// The real function takes the point struct passed in. Which is two 32bit numbers -> 1 64 bit number, for some reason in Go need to
	// swap the x and y. When in C, it was also in big endian
	value := point.toUintptrStructYX()
	monitorHandlePTR, _, err := MonitorFromPoint.Call(
		value,
		0, // If not in a monitor, returns null
	)
	monitorHandle = HMonitor(monitorHandlePTR)
	if monitorHandle == 0 {
		fmt.Println(err)
	}
	return
}

func setCursorPosition(point Point) {
	SetCursorPos.Call(uintptr(point.X), uintptr(point.Y))
}

func main() {
	EnumDisplayMonitorsF()
	SwitchToNextMonitorClockWise()
}

func (p Point) toUintptrStruct() (out uintptr) {
	x := p.X
	k := int64(x) << 32
	k = k | int64(p.Y)
	out = uintptr(k)
	return out
}

// In total, needed to switch x and y for some reason, and then ensure that
func (p Point) toUintptrStructYX() (out uintptr) {
	y := p.Y
	k := (uint64(y) & 0xFFFFFFFF) << 32
	k = k | (uint64(p.X) & 0xFFFFFFFF)
	out = uintptr(k)
	return out
}

func Int64ToPoint(in uintptr) (out Point) {
	out.X = int32(in >> 32)
	out.Y = int32(in & 0xFFFFFFFF)
	return out
}

func TheFunFunction() {
	point, err := getCursorPos()
	if err != nil {
		fmt.Println("Failed to get cursor position:", err)
		return
	}

	monitorHandle := getMonitorFromPoint(point)
	if monitorHandle == 0 {
		fmt.Println("Failed to get monitor handle")
		time.Sleep(time.Millisecond * 50)
	}

	monitorInfo, err := getMonitorInfo(syscall.Handle(monitorHandle))
	if err != nil {
		fmt.Println("Failed to get monitor info:", err)
		return
	}

	fmt.Printf("Cursor is on monitor:\n")
	fmt.Printf("  Monitor Bounds: (%d, %d, %d, %d)\n", monitorInfo.RcMonitor.Left, monitorInfo.RcMonitor.Top, monitorInfo.RcMonitor.Right, monitorInfo.RcMonitor.Bottom)
	fmt.Printf("  Work Area: (%d, %d, %d, %d)\n", monitorInfo.RcWork.Left, monitorInfo.RcWork.Top, monitorInfo.RcWork.Right, monitorInfo.RcWork.Bottom)
}

// When monitors are turned off, their order changes
func SwitchToMonitorByNumber(number int) {
	EnumDisplayMonitorsF()
}

type DISPLAY_DEVICEA struct {
	Cb           uint32
	DeviceName   [32]byte
	DeviceString [128]byte
	StateFlags   uint32
	DeviceID     [128]byte
	DeviceKey    [128]byte
}
