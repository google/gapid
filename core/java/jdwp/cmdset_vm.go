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

// Version describes the JDWP version
type Version struct {
	Description string //		Text information on the VM version
	JDWPMajor   int    //		Major JDWP Version number
	JDWPMinor   int    //		Minor JDWP Version number
	Version     string //		Target VM JRE version, as in the java.version property
	Name        string //		Target VM name, as in the java.vm.name property
}

// GetVersion returns the JDWP version from the server.
func (c *Connection) GetVersion() (Version, error) {
	res := Version{}
	err := c.get(cmdVirtualMachineVersion, struct{}{}, &res)
	return res, err
}

// ClassInfo describes a loaded classes matching the requested signature.
type ClassInfo struct {
	Kind      TypeTag         // Kind of reference type
	TypeID    ReferenceTypeID // Matching loaded reference type
	Signature string          // The class signature
	Status    ClassStatus     // The class status
}

// ClassID returns the class identifier for the ClassBySignature.
func (c ClassInfo) ClassID() ClassID {
	return ClassID(c.TypeID)
}

// GetClassesBySignature returns all the loaded classes matching the requested
// signature from the server.
func (c *Connection) GetClassesBySignature(signature string) ([]ClassInfo, error) {
	res := []struct {
		Kind   TypeTag
		TypeID ReferenceTypeID
		Status ClassStatus
	}{}
	err := c.get(cmdVirtualMachineClassesBySignature, &signature, &res)
	out := make([]ClassInfo, len(res))
	for i, c := range res {
		out[i] = ClassInfo{c.Kind, c.TypeID, signature, c.Status}
	}
	return out, err
}

// GetAllClasses returns all the active threads by ID.
func (c *Connection) GetAllClasses() ([]ClassInfo, error) {
	res := []ClassInfo{}
	err := c.get(cmdVirtualMachineAllClasses, struct{}{}, &res)
	return res, err
}

// GetAllThreads returns all the active threads by ID.
func (c *Connection) GetAllThreads() ([]ThreadID, error) {
	res := []ThreadID{}
	err := c.get(cmdVirtualMachineAllThreads, struct{}{}, &res)
	return res, err
}

// IDSizes describes the sizes of all the variably sized data types.
type IDSizes struct {
	FieldIDSize         int32 // FieldID size in bytes
	MethodIDSize        int32 // MethodID size in bytes
	ObjectIDSize        int32 // ObjectID size in bytes
	ReferenceTypeIDSize int32 // ReferenceTypeID size in bytes
	FrameIDSize         int32 // FrameID size in bytes
}

// GetIDSizes returns the sizes of all the variably sized data types.
func (c *Connection) GetIDSizes() (IDSizes, error) {
	res := IDSizes{}
	err := c.get(cmdVirtualMachineIDSizes, struct{}{}, &res)
	return res, err
}

// SuspendAll suspends all threads.
func (c *Connection) SuspendAll() error {
	return c.get(cmdVirtualMachineSuspend, struct{}{}, nil)
}

// ResumeAll resumes all threads.
func (c *Connection) ResumeAll() error {
	return c.get(cmdVirtualMachineResume, struct{}{}, nil)
}

// ResumeAllExcept resumes all threads except for the specified thread.
func (c *Connection) ResumeAllExcept(thread ThreadID) error {
	if err := c.Suspend(thread); err != nil {
		return err
	}
	return c.ResumeAll()
}

// CreateString returns the StringID for the given string.
func (c *Connection) CreateString(str string) (StringID, error) {
	res := StringID(0)
	err := c.get(cmdVirtualMachineCreateString, str, &res)
	return res, err
}
