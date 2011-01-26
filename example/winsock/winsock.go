package main

import "ole"
import "ole/oleutil"
import "unsafe"
import "syscall"
import "os"

type EventReceiver struct {
	lpVtbl *EventReceiverVtbl
	ref int
	host *ole.IDispatch
}

type EventReceiverVtbl struct {
	pQueryInterface   uintptr
	pAddRef           uintptr
	pRelease          uintptr
	pGetTypeInfoCount uintptr
	pGetTypeInfo      uintptr
	pGetIDsOfNames    uintptr
	pInvoke           uintptr
}

func QueryInterface(args *uintptr) uintptr {
	p := (*[3]int32)(unsafe.Pointer(args))
	this := (*ole.IUnknown)(unsafe.Pointer(uintptr(p[0])))
	iid := (*ole.GUID)(unsafe.Pointer(uintptr(p[1])))
	punk := (**ole.IUnknown)(unsafe.Pointer(uintptr(p[2])))
	//s, _ := ole.StringFromCLSID(iid)
	//println(s)
	*punk = nil
	if ole.IsEqualGUID(iid, ole.IID_IUnknown) ||
		ole.IsEqualGUID(iid, ole.IID_IDispatch) {
		this.AddRef()
		*punk = this
		return uintptr(ole.S_OK)
	}
	return uintptr(ole.E_NOINTERFACE)
}

func AddRef(args *uintptr) uintptr {
	p := (*[1]int32)(unsafe.Pointer(args))
	this := (*EventReceiver)(unsafe.Pointer(uintptr(p[0])))
	this.ref++
	return uintptr(this.ref)
}

func Release(args *uintptr) uintptr {
	p := (*[1]int32)(unsafe.Pointer(args))
	this := (*EventReceiver)(unsafe.Pointer(uintptr(p[0])))
	this.ref--
	return uintptr(this.ref)
}

func GetIDsOfNames(args *uintptr) uintptr {
	p := (*[6]int32)(unsafe.Pointer(args))
	//this := (*ole.IDispatch)(unsafe.Pointer(uintptr(p[0])))
	//iid := (*ole.GUID)(unsafe.Pointer(uintptr(p[1])))
	//wnames := *(*[]*uint16)(unsafe.Pointer(uintptr(p[2])))
	namelen := int(uintptr(p[3]))
	//lcid := int(uintptr(p[4]))
	pdisp := *(*[]int32)(unsafe.Pointer(uintptr(p[5])))
	for n := 0; n < namelen; n++ {
		pdisp[n] = int32(n)
	}
	return uintptr(ole.S_OK)
}

func GetTypeInfoCount(args *uintptr) uintptr {
	p := (*[2]int32)(unsafe.Pointer(args))
	//this := (*ole.IDispatch)(unsafe.Pointer(uintptr(p[0])))
	pcount := (*int)(unsafe.Pointer(uintptr(p[1])))
	if pcount != nil {
		*pcount = 0
	}
	return uintptr(ole.S_OK)
}

func GetTypeInfo(args *uintptr) uintptr {
	return uintptr(ole.E_NOTIMPL)
}

func Invoke(args *uintptr) uintptr {
	p := (*[9]int32)(unsafe.Pointer(args))
	this := (*ole.IDispatch)(unsafe.Pointer(uintptr(p[0])))
	dispid := int(p[1])

	switch dispid {
	case 0:
		println("DataArrival")
		winsock := (*EventReceiver)(unsafe.Pointer(this)).host
		var data ole.VARIANT
		ole.VariantInit(&data)
		oleutil.CallMethod(winsock, "GetData", &data)
		array := (*ole.SAFEARRAY)(unsafe.Pointer(uintptr(data.Val)))
		s := ole.BytePtrToString((*byte)(unsafe.Pointer(uintptr(array.PvData))))
		println(s)
	case 1:
		println("Connected")
		winsock := (*EventReceiver)(unsafe.Pointer(this)).host
		oleutil.CallMethod(winsock, "SendData", "GET / HTTP/1.0\r\n\r\n")
	case 3:
		println("SendProgress")
	case 4:
		println("SendComplete")
	case 5:
		println("Close")
		this.Release()
	default:
		println(dispid)
	}
	return ole.E_NOTIMPL
}

func main() {
	ole.CoInitialize(0)

	unknown, _ := oleutil.CreateObject("{248DD896-BB45-11CF-9ABC-0080C7E7B78D}")
	winsock, _ := unknown.QueryInterface(ole.IID_IDispatch)
	iid, _ := ole.CLSIDFromString("{248DD893-BB45-11CF-9ABC-0080C7E7B78D}")

	dest := &EventReceiver{}
	dest.lpVtbl = &EventReceiverVtbl{}
	dest.lpVtbl.pQueryInterface = syscall.NewCallback(QueryInterface, 3).ExtFnEntry()
	dest.lpVtbl.pAddRef = syscall.NewCallback(AddRef, 1).ExtFnEntry()
	dest.lpVtbl.pRelease = syscall.NewCallback(Release, 1).ExtFnEntry()
	dest.lpVtbl.pGetTypeInfoCount = syscall.NewCallback(GetTypeInfoCount, 2).ExtFnEntry()
	dest.lpVtbl.pGetTypeInfo = syscall.NewCallback(GetTypeInfo, 4).ExtFnEntry()
	dest.lpVtbl.pGetIDsOfNames = syscall.NewCallback(GetIDsOfNames, 6).ExtFnEntry()
	dest.lpVtbl.pInvoke = syscall.NewCallback(Invoke, 9).ExtFnEntry()
	dest.host = winsock

	oleutil.ConnectObject(winsock, iid, (*ole.IUnknown)(unsafe.Pointer(dest)))
	_, err := oleutil.CallMethod(winsock, "Connect", "127.0.0.1", 80)
	if err != nil {
		println(err.String())
		os.Exit(0)
	}

	var m ole.Msg
	for dest.ref != 0 {
		ole.GetMessage(&m, 0, 0, 0)
		ole.DispatchMessage(&m)
	}
}