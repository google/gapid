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

// Error is an enumerator of error codes returned by JDWP.
type Error uint16

const (
	ErrNone                                = Error(0)
	ErrInvalidThread                       = Error(10)
	ErrInvalidThreadGroup                  = Error(11)
	ErrInvalidPriority                     = Error(12)
	ErrThreadNotSuspended                  = Error(13)
	ErrThreadSuspended                     = Error(14)
	ErrInvalidObject                       = Error(20)
	ErrInvalidClass                        = Error(21)
	ErrClassNotPrepared                    = Error(22)
	ErrInvalidMethodID                     = Error(23)
	ErrInvalidLocation                     = Error(24)
	ErrInvalidFieldID                      = Error(25)
	ErrInvalidFrameID                      = Error(30)
	ErrNoMoreFrames                        = Error(31)
	ErrOpaqueFrame                         = Error(32)
	ErrNotCurrentFrame                     = Error(33)
	ErrTypeMismatch                        = Error(34)
	ErrInvalidSlot                         = Error(35)
	ErrDuplicate                           = Error(40)
	ErrNotFound                            = Error(41)
	ErrInvalidMonitor                      = Error(50)
	ErrNotMonitorOwner                     = Error(51)
	ErrInterrupt                           = Error(52)
	ErrInvalidClassFormat                  = Error(60)
	ErrCircularClassDefinition             = Error(61)
	ErrFailsVerification                   = Error(62)
	ErrAddMethodNotImplemented             = Error(63)
	ErrSchemaChangeNotImplemented          = Error(64)
	ErrInvalidTypestate                    = Error(65)
	ErrHierarchyChangeNotImplemented       = Error(66)
	ErrDeleteMethodNotImplemented          = Error(67)
	ErrUnsupportedVersion                  = Error(68)
	ErrNamesDontMatch                      = Error(69)
	ErrClassModifiersChangeNotImplemented  = Error(70)
	ErrMethodModifiersChangeNotImplemented = Error(71)
	ErrNotImplemented                      = Error(99)
	ErrNullPointer                         = Error(100)
	ErrAbsentInformation                   = Error(101)
	ErrInvalidEventType                    = Error(102)
	ErrIllegalArgument                     = Error(103)
	ErrOutOfMemory                         = Error(110)
	ErrAccessDenied                        = Error(111)
	ErrVMDead                              = Error(112)
	ErrInternal                            = Error(113)
	ErrUnattachedThread                    = Error(115)
	ErrInvalidTag                          = Error(500)
	ErrAlreadyInvoking                     = Error(502)
	ErrInvalidIndex                        = Error(503)
	ErrInvalidLength                       = Error(504)
	ErrInvalidString                       = Error(506)
	ErrInvalidClassLoader                  = Error(507)
	ErrInvalidArray                        = Error(508)
	ErrTransportLoad                       = Error(509)
	ErrTransportInit                       = Error(510)
	ErrNativeMethod                        = Error(511)
	ErrInvalidCount                        = Error(512)
)

func (e Error) Error() string {
	switch e {
	case ErrNone:
		return "No error has occurred."
	case ErrInvalidThread:
		return "Passed thread is null, is not a valid thread or has exited."
	case ErrInvalidThreadGroup:
		return "Thread group invalid."
	case ErrInvalidPriority:
		return "Invalid priority."
	case ErrThreadNotSuspended:
		return "The specified thread has not been suspended by an event."
	case ErrThreadSuspended:
		return "Thread already suspended."
	case ErrInvalidObject:
		return "This reference type has been unloaded and garbage collected."
	case ErrInvalidClass:
		return "Invalid class."
	case ErrClassNotPrepared:
		return "Class has been loaded but not yet prepared."
	case ErrInvalidMethodID:
		return "Invalid method."
	case ErrInvalidLocation:
		return "Invalid location."
	case ErrInvalidFieldID:
		return "Invalid field."
	case ErrInvalidFrameID:
		return "Invalid jframeID."
	case ErrNoMoreFrames:
		return "There are no more Java or JNI frames on the call stack."
	case ErrOpaqueFrame:
		return "Information about the frame is not available."
	case ErrNotCurrentFrame:
		return "Operation can only be performed on current frame."
	case ErrTypeMismatch:
		return "The variable is not an appropriate type for the function used."
	case ErrInvalidSlot:
		return "Invalid slot."
	case ErrDuplicate:
		return "Item already set."
	case ErrNotFound:
		return "Desired element not found."
	case ErrInvalidMonitor:
		return "Invalid monitor."
	case ErrNotMonitorOwner:
		return "This thread doesn't own the monitor."
	case ErrInterrupt:
		return "The call has been interrupted before completion."
	case ErrInvalidClassFormat:
		return "The virtual machine attempted to read a class file and determined that the file is malformed or otherwise cannot be interpreted as a class file."
	case ErrCircularClassDefinition:
		return "A circularity has been detected while initializing a class."
	case ErrFailsVerification:
		return "The verifier detected that a class file, though well formed, contained some sort of internal inconsistency or security problem."
	case ErrAddMethodNotImplemented:
		return "Adding methods has not been implemented."
	case ErrSchemaChangeNotImplemented:
		return "Schema change has not been implemented."
	case ErrInvalidTypestate:
		return "The state of the thread has been modified, and is now inconsistent."
	case ErrHierarchyChangeNotImplemented:
		return "A direct superclass is different for the new class version, or the set of directly implemented interfaces is different and canUnrestrictedlyRedefineClasses is false."
	case ErrDeleteMethodNotImplemented:
		return "The new class version does not declare a method declared in the old class version and canUnrestrictedlyRedefineClasses is false."
	case ErrUnsupportedVersion:
		return "A class file has a version number not supported by this VM."
	case ErrNamesDontMatch:
		return "The class name defined in the new class file is different from the name in the old class object."
	case ErrClassModifiersChangeNotImplemented:
		return "The new class version has different modifiers and and canUnrestrictedlyRedefineClasses is false."
	case ErrMethodModifiersChangeNotImplemented:
		return "A method in the new class version has different modifiers than its counterpart in the old class version and and canUnrestrictedlyRedefineClasses is false."
	case ErrNotImplemented:
		return "The functionality is not implemented in this virtual machine."
	case ErrNullPointer:
		return "Invalid pointer."
	case ErrAbsentInformation:
		return "Desired information is not available."
	case ErrInvalidEventType:
		return "The specified event type id is not recognized."
	case ErrIllegalArgument:
		return "Illegal argument."
	case ErrOutOfMemory:
		return "The function needed to allocate memory and no more memory was available for allocation."
	case ErrAccessDenied:
		return "Debugging has not been enabled in this virtual machine. JVMDI cannot be used."
	case ErrVMDead:
		return "The virtual machine is not running."
	case ErrInternal:
		return "An unexpected internal error has occurred."
	case ErrUnattachedThread:
		return "The thread being used to call this function is not attached to the virtual machine. Calls must be made from attached threads."
	case ErrInvalidTag:
		return "object type id or class tag."
	case ErrAlreadyInvoking:
		return "Previous invoke not complete."
	case ErrInvalidIndex:
		return "Index is invalid."
	case ErrInvalidLength:
		return "The length is invalid."
	case ErrInvalidString:
		return "The string is invalid."
	case ErrInvalidClassLoader:
		return "The class loader is invalid."
	case ErrInvalidArray:
		return "The array is invalid."
	case ErrTransportLoad:
		return "Unable to load the transport."
	case ErrTransportInit:
		return "Unable to initialize the transport."
	case ErrNativeMethod:
		return "Error native method."
	case ErrInvalidCount:
		return "The count is invalid."
	}
	return fmt.Sprintf("Error<%v>", int(e))
}
