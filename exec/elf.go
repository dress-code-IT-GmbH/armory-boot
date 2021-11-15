// https://github.com/f-secure-foundry/armory-boot
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package exec

import (
	"bytes"
	"debug/elf"
	"errors"
	"fmt"

	"github.com/f-secure-foundry/tamago/dma"
)

// ELFImage represents a bootable bare-metal ELF image.
type ELFImage struct {
	// Region is the memory area for image loading.
	Region *dma.Region
	// Kernel is the bootable bare-metal ELF image.
	Kernel []byte

	entry  uint32
	loaded bool
}

// Load loads a bare-metal ELF image in memory.
//
// The ELF loader is _very_ simple, suitable for loading unikernels like those
// produced by TamaGo.
func (image *ELFImage) Load() (err error) {
	if image.Region == nil {
		return errors.New("image memory Region must be assigned")
	}

	f, err := elf.NewFile(bytes.NewReader(image.Kernel))

	if err != nil {
		return
	}

	for idx, prg := range f.Progs {
		if prg.Type != elf.PT_LOAD {
			continue
		}

		b := make([]byte, prg.Memsz)

		if _, err := prg.ReadAt(b[0:prg.Filesz], 0); err != nil {
			return fmt.Errorf("failed to read LOAD section at idx %d, %q", idx, err)
		}

		if uint32(prg.Paddr) < image.Region.Start {
			return fmt.Errorf("incompatible memory layout (paddr:%x)", prg.Paddr)
		}

		off := uint32(prg.Paddr) - image.Region.Start

		if off > uint32(image.Region.Size) {
			return fmt.Errorf("incompatible memory layout (paddr:%x off:%x)", prg.Paddr, off)
		}

		image.Region.Write(image.Region.Start, int(off), b)
	}

	image.entry = uint32(f.Entry)
	image.loaded = true

	return
}

// BootELF calls a loaded bare-metal ELF image.
func (image *ELFImage) Boot(cleanup func()) (err error) {
	if !image.loaded {
		return errors.New("Load() kernel before Boot()")
	}

	return boot(image.entry, 0, cleanup)
}
