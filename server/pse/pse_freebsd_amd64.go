// Credit for The NATS.IO Authors
// Copyright 2021-2022 The Memphis Authors
// Licensed under the MIT License (the "License");
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// This license limiting reselling the software itself "AS IS".

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// This is the amd64-specific FreeBSD implementation, with hard-coded offset
// constants derived by running freebsd.txt; having this implementation allows
// us to compile without CGO, which lets us cross-compile for FreeBSD from our
// CI system and so supply binaries for FreeBSD amd64.
//
// To generate for other architectures:
//   1. Update pse_freebsd.go, change the build exclusion to exclude your arch
//   2. Copy this file to be built for your arch
//   3. Update `nativeEndian` below
//   4. Link `freebsd.txt` to have a .c filename and compile and run, then
//      paste the outputs into the const section below.

package pse

import (
	"encoding/binary"
	"syscall"

	"golang.org/x/sys/unix"
)

// On FreeBSD, to get proc information we read it out of the kernel using a
// binary sysctl.  The endianness of integers is thus explicitly "host", rather
// than little or big endian.
var nativeEndian = binary.LittleEndian

const (
	KIP_OFF_size   = 256
	KIP_OFF_rssize = 264
	KIP_OFF_pctcpu = 308
)

var pageshift int

func init() {
	// To get the physical page size, the C library checks two places:
	//   process ELF auxiliary info, AT_PAGESZ
	//   as a fallback, the hw.pagesize sysctl
	// In looking closely, I found that the Go runtime support is handling
	// this for us, and exposing that as syscall.Getpagesize, having checked
	// both in the same ways, at process start, so a call to that should return
	// a memory value without even a syscall bounce.
	pagesize := syscall.Getpagesize()
	pageshift = 0
	for pagesize > 1 {
		pageshift += 1
		pagesize >>= 1
	}
}

func ProcUsage(pcpu *float64, rss, vss *int64) error {
	rawdata, err := unix.SysctlRaw("kern.proc.pid", unix.Getpid())
	if err != nil {
		return err
	}

	r_vss_bytes := nativeEndian.Uint32(rawdata[KIP_OFF_size:])
	r_rss_pages := nativeEndian.Uint32(rawdata[KIP_OFF_rssize:])
	rss_bytes := r_rss_pages << pageshift

	// In C: fixpt_t ki_pctcpu
	// Doc: %cpu for process during ki_swtime
	// fixpt_t is __uint32_t
	// usr.bin/top uses pctdouble to convert to a double (float64)
	// define pctdouble(p) ((double)(p) / FIXED_PCTCPU)
	// FIXED_PCTCPU is _usually_ FSCALE (some architectures are special)
	// <sys/param.h> has:
	//   #define FSHIFT  11              /* bits to right of fixed binary point */
	//   #define FSCALE  (1<<FSHIFT)
	r_pcpu := nativeEndian.Uint32(rawdata[KIP_OFF_pctcpu:])
	f_pcpu := float64(r_pcpu) / float64(2048)

	*rss = int64(rss_bytes)
	*vss = int64(r_vss_bytes)
	*pcpu = f_pcpu

	return nil
}
