// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jdwp

import "fmt"

// cmdSet is the namespace for a command identifier.
type cmdSet uint8

// cmdID is a command in a command set.
type cmdID uint8

type cmd struct {
	set cmdSet
	id  cmdID
}

func (c cmd) String() string {
	return fmt.Sprintf("%v.%v", c.set, cmdNames[c])
}

const (
	cmdSetVirtualMachine       = cmdSet(1)
	cmdSetReferenceType        = cmdSet(2)
	cmdSetClassType            = cmdSet(3)
	cmdSetArrayType            = cmdSet(4)
	cmdSetInterfaceType        = cmdSet(5)
	cmdSetMethod               = cmdSet(6)
	cmdSetField                = cmdSet(8)
	cmdSetObjectReference      = cmdSet(9)
	cmdSetStringReference      = cmdSet(10)
	cmdSetThreadReference      = cmdSet(11)
	cmdSetThreadGroupReference = cmdSet(12)
	cmdSetArrayReference       = cmdSet(13)
	cmdSetClassLoaderReference = cmdSet(14)
	cmdSetEventRequest         = cmdSet(15)
	cmdSetStackFrame           = cmdSet(16)
	cmdSetClassObjectReference = cmdSet(17)
	cmdSetEvent                = cmdSet(64)
)

func (c cmdSet) String() string {
	switch c {
	case cmdSetVirtualMachine:
		return "VirtualMachine"
	case cmdSetReferenceType:
		return "ReferenceType"
	case cmdSetClassType:
		return "ClassType"
	case cmdSetArrayType:
		return "ArrayType"
	case cmdSetInterfaceType:
		return "InterfaceType"
	case cmdSetMethod:
		return "Method"
	case cmdSetField:
		return "Field"
	case cmdSetObjectReference:
		return "ObjectReference"
	case cmdSetStringReference:
		return "StringReference"
	case cmdSetThreadReference:
		return "ThreadReference"
	case cmdSetThreadGroupReference:
		return "ThreadGroupReference"
	case cmdSetArrayReference:
		return "ArrayReference"
	case cmdSetClassLoaderReference:
		return "ClassLoaderReference"
	case cmdSetEventRequest:
		return "EventRequest"
	case cmdSetStackFrame:
		return "StackFrame"
	case cmdSetClassObjectReference:
		return "ClassObjectReference"
	case cmdSetEvent:
		return "Event"
	}
	return fmt.Sprint(int(c))
}

var (
	cmdVirtualMachineVersion               = cmd{cmdSetVirtualMachine, 1}
	cmdVirtualMachineClassesBySignature    = cmd{cmdSetVirtualMachine, 2}
	cmdVirtualMachineAllClasses            = cmd{cmdSetVirtualMachine, 3}
	cmdVirtualMachineAllThreads            = cmd{cmdSetVirtualMachine, 4}
	cmdVirtualMachineTopLevelThreadGroups  = cmd{cmdSetVirtualMachine, 5}
	cmdVirtualMachineDispose               = cmd{cmdSetVirtualMachine, 6}
	cmdVirtualMachineIDSizes               = cmd{cmdSetVirtualMachine, 7}
	cmdVirtualMachineSuspend               = cmd{cmdSetVirtualMachine, 8}
	cmdVirtualMachineResume                = cmd{cmdSetVirtualMachine, 9}
	cmdVirtualMachineExit                  = cmd{cmdSetVirtualMachine, 10}
	cmdVirtualMachineCreateString          = cmd{cmdSetVirtualMachine, 11}
	cmdVirtualMachineCapabilities          = cmd{cmdSetVirtualMachine, 12}
	cmdVirtualMachineClassPaths            = cmd{cmdSetVirtualMachine, 13}
	cmdVirtualMachineDisposeObjects        = cmd{cmdSetVirtualMachine, 14}
	cmdVirtualMachineHoldEvents            = cmd{cmdSetVirtualMachine, 15}
	cmdVirtualMachineReleaseEvents         = cmd{cmdSetVirtualMachine, 16}
	cmdVirtualMachineCapabilitiesNew       = cmd{cmdSetVirtualMachine, 17}
	cmdVirtualMachineRedefineClasses       = cmd{cmdSetVirtualMachine, 18}
	cmdVirtualMachineSetDefaultStratum     = cmd{cmdSetVirtualMachine, 19}
	cmdVirtualMachineAllClassesWithGeneric = cmd{cmdSetVirtualMachine, 20}

	cmdReferenceTypeSignature            = cmd{cmdSetReferenceType, 1}
	cmdReferenceTypeClassLoader          = cmd{cmdSetReferenceType, 2}
	cmdReferenceTypeModifiers            = cmd{cmdSetReferenceType, 3}
	cmdReferenceTypeFields               = cmd{cmdSetReferenceType, 4}
	cmdReferenceTypeMethods              = cmd{cmdSetReferenceType, 5}
	cmdReferenceTypeGetValues            = cmd{cmdSetReferenceType, 6}
	cmdReferenceTypeSourceFile           = cmd{cmdSetReferenceType, 7}
	cmdReferenceTypeNestedTypes          = cmd{cmdSetReferenceType, 8}
	cmdReferenceTypeStatus               = cmd{cmdSetReferenceType, 9}
	cmdReferenceTypeInterfaces           = cmd{cmdSetReferenceType, 10}
	cmdReferenceTypeClassObject          = cmd{cmdSetReferenceType, 11}
	cmdReferenceTypeSourceDebugExtension = cmd{cmdSetReferenceType, 12}
	cmdReferenceTypeSignatureWithGeneric = cmd{cmdSetReferenceType, 13}
	cmdReferenceTypeFieldsWithGeneric    = cmd{cmdSetReferenceType, 14}
	cmdReferenceTypeMethodsWithGeneric   = cmd{cmdSetReferenceType, 15}

	cmdClassTypeSuperclass   = cmd{cmdSetClassType, 1}
	cmdClassTypeSetValues    = cmd{cmdSetClassType, 2}
	cmdClassTypeInvokeMethod = cmd{cmdSetClassType, 3}
	cmdClassTypeNewInstance  = cmd{cmdSetClassType, 4}

	cmdArrayTypeNewInstance = cmd{cmdSetArrayType, 1}

	cmdMethodTypeLineTable                = cmd{cmdSetMethod, 1}
	cmdMethodTypeVariableTable            = cmd{cmdSetMethod, 2}
	cmdMethodTypeBytecodes                = cmd{cmdSetMethod, 3}
	cmdMethodTypeIsObsolete               = cmd{cmdSetMethod, 4}
	cmdMethodTypeVariableTableWithGeneric = cmd{cmdSetMethod, 5}

	cmdObjectReferenceReferenceType     = cmd{cmdSetObjectReference, 1}
	cmdObjectReferenceGetValues         = cmd{cmdSetObjectReference, 2}
	cmdObjectReferenceSetValues         = cmd{cmdSetObjectReference, 3}
	cmdObjectReferenceMonitorInfo       = cmd{cmdSetObjectReference, 5}
	cmdObjectReferenceInvokeMethod      = cmd{cmdSetObjectReference, 6}
	cmdObjectReferenceDisableCollection = cmd{cmdSetObjectReference, 7}
	cmdObjectReferenceEnableCollection  = cmd{cmdSetObjectReference, 8}
	cmdObjectReferenceIsCollected       = cmd{cmdSetObjectReference, 9}

	cmdStringReferenceValue = cmd{cmdSetStringReference, 1}

	cmdThreadReferenceName                    = cmd{cmdSetThreadReference, 1}
	cmdThreadReferenceSuspend                 = cmd{cmdSetThreadReference, 2}
	cmdThreadReferenceResume                  = cmd{cmdSetThreadReference, 3}
	cmdThreadReferenceStatus                  = cmd{cmdSetThreadReference, 4}
	cmdThreadReferenceThreadGroup             = cmd{cmdSetThreadReference, 5}
	cmdThreadReferenceFrames                  = cmd{cmdSetThreadReference, 6}
	cmdThreadReferenceFrameCount              = cmd{cmdSetThreadReference, 7}
	cmdThreadReferenceOwnedMonitors           = cmd{cmdSetThreadReference, 8}
	cmdThreadReferenceCurrentContendedMonitor = cmd{cmdSetThreadReference, 9}
	cmdThreadReferenceStop                    = cmd{cmdSetThreadReference, 10}
	cmdThreadReferenceInterrupt               = cmd{cmdSetThreadReference, 11}
	cmdThreadReferenceSuspendCount            = cmd{cmdSetThreadReference, 12}

	cmdThreadGroupReferenceName     = cmd{cmdSetThreadGroupReference, 1}
	cmdThreadGroupReferenceParent   = cmd{cmdSetThreadGroupReference, 2}
	cmdThreadGroupReferenceChildren = cmd{cmdSetThreadGroupReference, 3}

	cmdArrayReferenceLength    = cmd{cmdSetArrayReference, 1}
	cmdArrayReferenceGetValues = cmd{cmdSetArrayReference, 2}
	cmdArrayReferenceSetValues = cmd{cmdSetArrayReference, 3}

	cmdClassLoaderReferenceVisibleClasses = cmd{cmdSetClassLoaderReference, 1}

	cmdEventRequestSet                 = cmd{cmdSetEventRequest, 1}
	cmdEventRequestClear               = cmd{cmdSetEventRequest, 2}
	cmdEventRequestClearAllBreakpoints = cmd{cmdSetEventRequest, 3}

	cmdStackFrameGetValues  = cmd{cmdSetStackFrame, 1}
	cmdStackFrameSetValues  = cmd{cmdSetStackFrame, 2}
	cmdStackFrameThisObject = cmd{cmdSetStackFrame, 3}
	cmdStackFramePopFrames  = cmd{cmdSetStackFrame, 4}

	cmdClassObjectReferenceReflectedType = cmd{cmdSetClassObjectReference, 1}

	cmdEventComposite = cmd{cmdSetEvent, 1}
)

var cmdNames = map[cmd]string{}

func init() {
	register := func(c cmd, n string) {
		if _, e := cmdNames[c]; e {
			panic("command already registered")
		}
		cmdNames[c] = n
	}
	register(cmdVirtualMachineVersion, "Version")
	register(cmdVirtualMachineClassesBySignature, "ClassesBySignature")
	register(cmdVirtualMachineAllClasses, "AllClasses")
	register(cmdVirtualMachineAllThreads, "AllThreads")
	register(cmdVirtualMachineTopLevelThreadGroups, "TopLevelThreadGroups")
	register(cmdVirtualMachineDispose, "Dispose")
	register(cmdVirtualMachineIDSizes, "IDSizes")
	register(cmdVirtualMachineSuspend, "Suspend")
	register(cmdVirtualMachineResume, "Resume")
	register(cmdVirtualMachineExit, "Exit")
	register(cmdVirtualMachineCreateString, "CreateString")
	register(cmdVirtualMachineCapabilities, "Capabilities")
	register(cmdVirtualMachineClassPaths, "ClassPaths")
	register(cmdVirtualMachineDisposeObjects, "DisposeObjects")
	register(cmdVirtualMachineHoldEvents, "HoldEvents")
	register(cmdVirtualMachineReleaseEvents, "ReleaseEvents")
	register(cmdVirtualMachineCapabilitiesNew, "CapabilitiesNew")
	register(cmdVirtualMachineRedefineClasses, "RedefineClasses")
	register(cmdVirtualMachineSetDefaultStratum, "SetDefaultStratum")
	register(cmdVirtualMachineAllClassesWithGeneric, "AllClassesWithGeneric")

	register(cmdReferenceTypeSignature, "Signature")
	register(cmdReferenceTypeClassLoader, "ClassLoader")
	register(cmdReferenceTypeModifiers, "Modifiers")
	register(cmdReferenceTypeFields, "Fields")
	register(cmdReferenceTypeMethods, "Methods")
	register(cmdReferenceTypeGetValues, "GetValues")
	register(cmdReferenceTypeSourceFile, "SourceFile")
	register(cmdReferenceTypeNestedTypes, "NestedTypes")
	register(cmdReferenceTypeStatus, "Status")
	register(cmdReferenceTypeInterfaces, "Interfaces")
	register(cmdReferenceTypeClassObject, "ClassObject")
	register(cmdReferenceTypeSourceDebugExtension, "SourceDebugExtension")
	register(cmdReferenceTypeSignatureWithGeneric, "SignatureWithGeneric")
	register(cmdReferenceTypeFieldsWithGeneric, "FieldsWithGeneric")
	register(cmdReferenceTypeMethodsWithGeneric, "MethodsWithGeneric")

	register(cmdClassTypeSuperclass, "Superclass")
	register(cmdClassTypeSetValues, "SetValues")
	register(cmdClassTypeInvokeMethod, "InvokeMethod")
	register(cmdClassTypeNewInstance, "NewInstance")

	register(cmdArrayTypeNewInstance, "NewInstance")

	register(cmdMethodTypeLineTable, "LineTable")
	register(cmdMethodTypeVariableTable, "VariableTable")
	register(cmdMethodTypeBytecodes, "Bytecodes")
	register(cmdMethodTypeIsObsolete, "IsObsolete")
	register(cmdMethodTypeVariableTableWithGeneric, "VariableTableWithGeneric")

	register(cmdObjectReferenceReferenceType, "ReferenceType")
	register(cmdObjectReferenceGetValues, "GetValues")
	register(cmdObjectReferenceSetValues, "SetValues")
	register(cmdObjectReferenceMonitorInfo, "MonitorInfo")
	register(cmdObjectReferenceInvokeMethod, "InvokeMethod")
	register(cmdObjectReferenceDisableCollection, "DisableCollection")
	register(cmdObjectReferenceEnableCollection, "EnableCollection")
	register(cmdObjectReferenceIsCollected, "IsCollected")

	register(cmdStringReferenceValue, "Value")

	register(cmdThreadReferenceName, "Name")
	register(cmdThreadReferenceSuspend, "Suspend")
	register(cmdThreadReferenceResume, "Resume")
	register(cmdThreadReferenceStatus, "Status")
	register(cmdThreadReferenceThreadGroup, "ThreadGroup")
	register(cmdThreadReferenceFrames, "Frames")
	register(cmdThreadReferenceFrameCount, "FrameCount")
	register(cmdThreadReferenceOwnedMonitors, "OwnedMonitors")
	register(cmdThreadReferenceCurrentContendedMonitor, "CurrentContendedMonitor")
	register(cmdThreadReferenceStop, "Stop")
	register(cmdThreadReferenceInterrupt, "Interrupt")
	register(cmdThreadReferenceSuspendCount, "SuspendCount")

	register(cmdThreadGroupReferenceName, "Name")
	register(cmdThreadGroupReferenceParent, "Parent")
	register(cmdThreadGroupReferenceChildren, "Children")

	register(cmdArrayReferenceLength, "Length")
	register(cmdArrayReferenceGetValues, "GetValues")
	register(cmdArrayReferenceSetValues, "SetValues")

	register(cmdClassLoaderReferenceVisibleClasses, "VisibleClasses")

	register(cmdEventRequestSet, "Set")
	register(cmdEventRequestClear, "Clear")
	register(cmdEventRequestClearAllBreakpoints, "ClearAllBreakpoints")

	register(cmdStackFrameGetValues, "GetValues")
	register(cmdStackFrameSetValues, "SetValues")
	register(cmdStackFrameThisObject, "ThisObject")
	register(cmdStackFramePopFrames, "PopFrames")

	register(cmdClassObjectReferenceReflectedType, "ReflectedType")

	register(cmdEventComposite, "Composite")
}
