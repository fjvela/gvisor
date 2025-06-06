// Copyright 2018 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kvm

// KVM ioctls.
//
// Only the ioctls we need in Go appear here; some additional ioctls are used
// within the assembly stubs (KVM_INTERRUPT, etc.).
const (
	KVM_CREATE_VM              = 0xae01
	KVM_GET_VCPU_MMAP_SIZE     = 0xae04
	KVM_CREATE_VCPU            = 0xae41
	KVM_SET_TSS_ADDR           = 0xae47
	KVM_RUN                    = 0xae80
	KVM_NMI                    = 0xae9a
	KVM_CHECK_EXTENSION        = 0xae03
	KVM_GET_TSC_KHZ            = 0xaea3
	KVM_SET_TSC_KHZ            = 0xaea2
	KVM_INTERRUPT              = 0x4004ae86
	KVM_SET_MSRS               = 0x4008ae89
	KVM_SET_USER_MEMORY_REGION = 0x4020ae46
	KVM_IOEVENTFD              = 0x4040ae79
	KVM_SET_REGS               = 0x4090ae82
	KVM_SET_SREGS              = 0x4138ae84
	KVM_GET_MSRS               = 0xc008ae88
	KVM_GET_REGS               = 0x8090ae81
	KVM_GET_SREGS              = 0x8138ae83
	KVM_GET_SUPPORTED_CPUID    = 0xc008ae05
	KVM_SET_CPUID2             = 0x4008ae90
	KVM_SET_SIGNAL_MASK        = 0x4004ae8b
	KVM_GET_VCPU_EVENTS        = 0x8040ae9f
	KVM_SET_VCPU_EVENTS        = 0x4040aea0
	KVM_SET_DEVICE_ATTR        = 0x4018aee1
	KVM_ENABLE_CAP             = 0x4068aea3
)

// KVM exit reasons.
const (
	_KVM_EXIT_EXCEPTION       = 0x1
	_KVM_EXIT_IO              = 0x2
	_KVM_EXIT_HYPERCALL       = 0x3
	_KVM_EXIT_DEBUG           = 0x4
	_KVM_EXIT_HLT             = 0x5
	_KVM_EXIT_MMIO            = 0x6
	_KVM_EXIT_IRQ_WINDOW_OPEN = 0x7
	_KVM_EXIT_SHUTDOWN        = 0x8
	_KVM_EXIT_FAIL_ENTRY      = 0x9
	_KVM_EXIT_INTERNAL_ERROR  = 0x11
	_KVM_EXIT_SYSTEM_EVENT    = 0x18
	_KVM_EXIT_ARM_NISV        = 0x1c
)

// KVM capability options.
const (
	_KVM_CAP_MAX_MEMSLOTS          = 0x0a
	_KVM_CAP_MAX_VCPUS             = 0x42
	_KVM_CAP_ARM_VM_IPA_SIZE       = 0xa5
	_KVM_CAP_VCPU_EVENTS           = 0x29
	_KVM_CAP_ARM_INJECT_SERROR_ESR = 0x9e
	_KVM_CAP_TSC_CONTROL           = 0x3c
)

// KVM limits.
const (
	_KVM_NR_MEMSLOTS      = 0x100
	_KVM_NR_VCPUS         = 0xff
	_KVM_NR_INTERRUPTS    = 0x100
	_KVM_NR_CPUID_ENTRIES = 0x100
)

// KVM kvm_memory_region::flags.
const (
	_KVM_MEM_LOG_DIRTY_PAGES = uint32(1) << 0
	_KVM_MEM_READONLY        = uint32(1) << 1
	_KVM_MEM_FLAGS_NONE      = uint32(0)
)

// KVM kvm_ioeventfd::flags.
const (
	_KVM_IOEVENTFD_FLAG_DEASSIGN = 1 << 2
)

// KVM hypercall list.
//
// Canonical list of hypercalls supported.
const (
	// On amd64, it uses 'HLT' to leave the guest.
	//
	// Unlike amd64, arm64 can only uses mmio_exit/psci to leave the guest.
	//
	// _KVM_HYPERCALL_VMEXIT is only used on arm64 for now.
	_KVM_HYPERCALL_VMEXIT int = iota
	_KVM_HYPERCALL_MAX
)
