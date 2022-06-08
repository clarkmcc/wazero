package compiler

import (
	"errors"

	"github.com/tetratelabs/wazero/internal/asm"
	"github.com/tetratelabs/wazero/internal/asm/amd64"
	"github.com/tetratelabs/wazero/internal/wazeroir"
)

// compileV128Const implements compiler.compileV128Const for amd64 architecture.
func (c *amd64Compiler) compileV128Const(o *wazeroir.OperationV128Const) error {
	c.maybeCompileMoveTopConditionalToFreeGeneralPurposeRegister()

	result, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	// We cannot directly load the value from memory to float regs,
	// so we move it to int reg temporarily.
	tmpReg, err := c.allocateRegister(registerTypeGeneralPurpose)
	if err != nil {
		return err
	}

	// Move the lower 64-bits.
	if o.Lo == 0 {
		c.assembler.CompileRegisterToRegister(amd64.XORQ, tmpReg, tmpReg)
	} else {
		c.assembler.CompileConstToRegister(amd64.MOVQ, int64(o.Lo), tmpReg)
	}
	c.assembler.CompileRegisterToRegister(amd64.MOVQ, tmpReg, result)

	if o.Lo != 0 && o.Hi == 0 {
		c.assembler.CompileRegisterToRegister(amd64.XORQ, tmpReg, tmpReg)
	} else if o.Hi != 0 {
		c.assembler.CompileConstToRegister(amd64.MOVQ, int64(o.Hi), tmpReg)
	}
	// Move the higher 64-bits with PINSRQ at the second element of 64x2 vector.
	c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRQ, tmpReg, result, 1)

	c.pushVectorRuntimeValueLocationOnRegister(result)
	return nil
}

// compileV128Add implements compiler.compileV128Add for amd64 architecture.
func (c *amd64Compiler) compileV128Add(o *wazeroir.OperationV128Add) error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}
	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		inst = amd64.PADDB
	case wazeroir.ShapeI16x8:
		inst = amd64.PADDW
	case wazeroir.ShapeI32x4:
		inst = amd64.PADDD
	case wazeroir.ShapeI64x2:
		inst = amd64.PADDQ
	case wazeroir.ShapeF32x4:
		inst = amd64.ADDPS
	case wazeroir.ShapeF64x2:
		inst = amd64.ADDPD
	}
	c.assembler.CompileRegisterToRegister(inst, x2.register, x1.register)

	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	c.locationStack.markRegisterUnused(x2.register)
	return nil
}

// compileV128Sub implements compiler.compileV128Sub for amd64 architecture.
func (c *amd64Compiler) compileV128Sub(o *wazeroir.OperationV128Sub) error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}
	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		inst = amd64.PSUBB
	case wazeroir.ShapeI16x8:
		inst = amd64.PSUBW
	case wazeroir.ShapeI32x4:
		inst = amd64.PSUBD
	case wazeroir.ShapeI64x2:
		inst = amd64.PSUBQ
	case wazeroir.ShapeF32x4:
		inst = amd64.SUBPS
	case wazeroir.ShapeF64x2:
		inst = amd64.SUBPD
	}
	c.assembler.CompileRegisterToRegister(inst, x2.register, x1.register)

	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	c.locationStack.markRegisterUnused(x2.register)
	return nil
}

// compileV128Load implements compiler.compileV128Load for amd64 architecture.
func (c *amd64Compiler) compileV128Load(o *wazeroir.OperationV128Load) error {
	result, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	switch o.Type {
	case wazeroir.LoadV128Type128:
		err = c.compileV128LoadImpl(amd64.MOVDQU, o.Arg.Offset, 16, result)
	case wazeroir.LoadV128Type8x8s:
		err = c.compileV128LoadImpl(amd64.PMOVSXBW, o.Arg.Offset, 8, result)
	case wazeroir.LoadV128Type8x8u:
		err = c.compileV128LoadImpl(amd64.PMOVZXBW, o.Arg.Offset, 8, result)
	case wazeroir.LoadV128Type16x4s:
		err = c.compileV128LoadImpl(amd64.PMOVSXWD, o.Arg.Offset, 8, result)
	case wazeroir.LoadV128Type16x4u:
		err = c.compileV128LoadImpl(amd64.PMOVZXWD, o.Arg.Offset, 8, result)
	case wazeroir.LoadV128Type32x2s:
		err = c.compileV128LoadImpl(amd64.PMOVSXDQ, o.Arg.Offset, 8, result)
	case wazeroir.LoadV128Type32x2u:
		err = c.compileV128LoadImpl(amd64.PMOVZXDQ, o.Arg.Offset, 8, result)
	case wazeroir.LoadV128Type8Splat:
		reg, err := c.compileMemoryAccessCeilSetup(o.Arg.Offset, 1)
		if err != nil {
			return err
		}
		c.assembler.CompileMemoryWithIndexToRegister(amd64.MOVBQZX, amd64ReservedRegisterForMemory, -1,
			reg, 1, reg)
		// pinsrb   $0, reg, result
		// pxor	    tmpVReg, tmpVReg
		// pshufb   tmpVReg, result
		c.locationStack.markRegisterUsed(result)
		tmpVReg, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRB, reg, result, 0)
		c.assembler.CompileRegisterToRegister(amd64.PXOR, tmpVReg, tmpVReg)
		c.assembler.CompileRegisterToRegister(amd64.PSHUFB, tmpVReg, result)
	case wazeroir.LoadV128Type16Splat:
		reg, err := c.compileMemoryAccessCeilSetup(o.Arg.Offset, 2)
		if err != nil {
			return err
		}
		c.assembler.CompileMemoryWithIndexToRegister(amd64.MOVWQZX, amd64ReservedRegisterForMemory, -2,
			reg, 1, reg)
		// pinsrw $0, reg, result
		// pinsrw $1, reg, result
		// pshufd $0, result, result (result = result[0,0,0,0])
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRW, reg, result, 0)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRW, reg, result, 1)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PSHUFD, result, result, 0)
	case wazeroir.LoadV128Type32Splat:
		reg, err := c.compileMemoryAccessCeilSetup(o.Arg.Offset, 4)
		if err != nil {
			return err
		}
		c.assembler.CompileMemoryWithIndexToRegister(amd64.MOVLQZX, amd64ReservedRegisterForMemory, -4,
			reg, 1, reg)
		// pinsrd $0, reg, result
		// pshufd $0, result, result (result = result[0,0,0,0])
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRD, reg, result, 0)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PSHUFD, result, result, 0)
	case wazeroir.LoadV128Type64Splat:
		reg, err := c.compileMemoryAccessCeilSetup(o.Arg.Offset, 8)
		if err != nil {
			return err
		}
		c.assembler.CompileMemoryWithIndexToRegister(amd64.MOVQ, amd64ReservedRegisterForMemory, -8,
			reg, 1, reg)
		// pinsrq $0, reg, result
		// pinsrq $1, reg, result
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRQ, reg, result, 0)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRQ, reg, result, 1)
	case wazeroir.LoadV128Type32zero:
		err = c.compileV128LoadImpl(amd64.MOVL, o.Arg.Offset, 4, result)
	case wazeroir.LoadV128Type64zero:
		err = c.compileV128LoadImpl(amd64.MOVQ, o.Arg.Offset, 8, result)
	}

	if err != nil {
		return err
	}

	c.pushVectorRuntimeValueLocationOnRegister(result)
	return nil
}

func (c *amd64Compiler) compileV128LoadImpl(inst asm.Instruction, offset uint32, targetSizeInBytes int64, dst asm.Register) error {
	offsetReg, err := c.compileMemoryAccessCeilSetup(offset, targetSizeInBytes)
	if err != nil {
		return err
	}
	c.assembler.CompileMemoryWithIndexToRegister(inst, amd64ReservedRegisterForMemory, -targetSizeInBytes,
		offsetReg, 1, dst)
	return nil
}

// compileV128LoadLane implements compiler.compileV128LoadLane for amd64.
func (c *amd64Compiler) compileV128LoadLane(o *wazeroir.OperationV128LoadLane) error {
	targetVector := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(targetVector); err != nil {
		return err
	}

	var insertInst asm.Instruction
	switch o.LaneSize {
	case 8:
		insertInst = amd64.PINSRB
	case 16:
		insertInst = amd64.PINSRW
	case 32:
		insertInst = amd64.PINSRD
	case 64:
		insertInst = amd64.PINSRQ
	}

	targetSizeInBytes := int64(o.LaneSize / 8)
	offsetReg, err := c.compileMemoryAccessCeilSetup(o.Arg.Offset, targetSizeInBytes)
	if err != nil {
		return err
	}
	c.assembler.CompileMemoryWithIndexAndArgToRegister(insertInst, amd64ReservedRegisterForMemory, -targetSizeInBytes,
		offsetReg, 1, targetVector.register, o.LaneIndex)

	c.pushVectorRuntimeValueLocationOnRegister(targetVector.register)
	return nil
}

// compileV128Store implements compiler.compileV128Store for amd64.
func (c *amd64Compiler) compileV128Store(o *wazeroir.OperationV128Store) error {
	val := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(val); err != nil {
		return err
	}

	const targetSizeInBytes = 16
	offsetReg, err := c.compileMemoryAccessCeilSetup(o.Arg.Offset, targetSizeInBytes)
	if err != nil {
		return err
	}

	c.assembler.CompileRegisterToMemoryWithIndex(amd64.MOVDQU, val.register,
		amd64ReservedRegisterForMemory, -targetSizeInBytes, offsetReg, 1)

	c.locationStack.markRegisterUnused(val.register, offsetReg)
	return nil
}

// compileV128StoreLane implements compiler.compileV128StoreLane for amd64.
func (c *amd64Compiler) compileV128StoreLane(o *wazeroir.OperationV128StoreLane) error {
	var storeInst asm.Instruction
	switch o.LaneSize {
	case 8:
		storeInst = amd64.PEXTRB
	case 16:
		storeInst = amd64.PEXTRW
	case 32:
		storeInst = amd64.PEXTRD
	case 64:
		storeInst = amd64.PEXTRQ
	}

	val := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(val); err != nil {
		return err
	}

	targetSizeInBytes := int64(o.LaneSize / 8)
	offsetReg, err := c.compileMemoryAccessCeilSetup(o.Arg.Offset, targetSizeInBytes)
	if err != nil {
		return err
	}

	c.assembler.CompileRegisterToMemoryWithIndexAndArg(storeInst, val.register,
		amd64ReservedRegisterForMemory, -targetSizeInBytes, offsetReg, 1, o.LaneIndex)

	c.locationStack.markRegisterUnused(val.register, offsetReg)
	return nil
}

// compileV128ExtractLane implements compiler.compileV128ExtractLane for amd64.
func (c *amd64Compiler) compileV128ExtractLane(o *wazeroir.OperationV128ExtractLane) error {
	val := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(val); err != nil {
		return err
	}
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		result, err := c.allocateRegister(registerTypeGeneralPurpose)
		if err != nil {
			return err
		}
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PEXTRB, val.register, result, o.LaneIndex)
		if o.Signed {
			c.assembler.CompileRegisterToRegister(amd64.MOVBQSX, result, result)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.MOVBLZX, result, result)
		}
		c.pushRuntimeValueLocationOnRegister(result, runtimeValueTypeI32)
		c.locationStack.markRegisterUnused(val.register)
	case wazeroir.ShapeI16x8:
		result, err := c.allocateRegister(registerTypeGeneralPurpose)
		if err != nil {
			return err
		}
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PEXTRW, val.register, result, o.LaneIndex)
		if o.Signed {
			c.assembler.CompileRegisterToRegister(amd64.MOVWLSX, result, result)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.MOVWLZX, result, result)
		}
		c.pushRuntimeValueLocationOnRegister(result, runtimeValueTypeI32)
		c.locationStack.markRegisterUnused(val.register)
	case wazeroir.ShapeI32x4:
		result, err := c.allocateRegister(registerTypeGeneralPurpose)
		if err != nil {
			return err
		}
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PEXTRD, val.register, result, o.LaneIndex)
		c.pushRuntimeValueLocationOnRegister(result, runtimeValueTypeI32)
		c.locationStack.markRegisterUnused(val.register)
	case wazeroir.ShapeI64x2:
		result, err := c.allocateRegister(registerTypeGeneralPurpose)
		if err != nil {
			return err
		}
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PEXTRQ, val.register, result, o.LaneIndex)
		c.pushRuntimeValueLocationOnRegister(result, runtimeValueTypeI64)
		c.locationStack.markRegisterUnused(val.register)
	case wazeroir.ShapeF32x4:
		if o.LaneIndex != 0 {
			c.assembler.CompileRegisterToRegisterWithArg(amd64.PSHUFD, val.register, val.register, o.LaneIndex)
		}
		c.pushRuntimeValueLocationOnRegister(val.register, runtimeValueTypeF32)
	case wazeroir.ShapeF64x2:
		if o.LaneIndex != 0 {
			// This case we can assume LaneIndex == 1.
			// We have to modify the val.register as, for example:
			//    0b11 0b10 0b01 0b00
			//     |    |    |    |
			//   [x3,  x2,  x1,  x0] -> [x0,  x0,  x3,  x2]
			// where val.register = [x3, x2, x1, x0] and each xN = 32bits.
			// Then, we interpret the register as float64, therefore, the float64 value is obtained as [x3, x2].
			arg := byte(0b00_00_11_10)
			c.assembler.CompileRegisterToRegisterWithArg(amd64.PSHUFD, val.register, val.register, arg)
		}
		c.pushRuntimeValueLocationOnRegister(val.register, runtimeValueTypeF64)
	}

	return nil
}

// compileV128ReplaceLane implements compiler.compileV128ReplaceLane for amd64.
func (c *amd64Compiler) compileV128ReplaceLane(o *wazeroir.OperationV128ReplaceLane) error {
	origin := c.locationStack.pop()
	if err := c.compileEnsureOnGeneralPurposeRegister(origin); err != nil {
		return err
	}

	vector := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(vector); err != nil {
		return err
	}

	switch o.Shape {
	case wazeroir.ShapeI8x16:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRB, origin.register, vector.register, o.LaneIndex)
	case wazeroir.ShapeI16x8:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRW, origin.register, vector.register, o.LaneIndex)
	case wazeroir.ShapeI32x4:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRD, origin.register, vector.register, o.LaneIndex)
	case wazeroir.ShapeI64x2:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRQ, origin.register, vector.register, o.LaneIndex)
	case wazeroir.ShapeF32x4:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.INSERTPS, origin.register, vector.register,
			// In INSERTPS instruction, the destination index is encoded at 4 and 5 bits of the argument.
			// See https://www.felixcloutier.com/x86/insertps
			o.LaneIndex<<4,
		)
	case wazeroir.ShapeF64x2:
		if o.LaneIndex == 0 {
			c.assembler.CompileRegisterToRegister(amd64.MOVSD, origin.register, vector.register)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.MOVLHPS, origin.register, vector.register)
		}
	}

	c.pushVectorRuntimeValueLocationOnRegister(vector.register)
	c.locationStack.markRegisterUnused(origin.register)
	return nil
}

// compileV128Splat implements compiler.compileV128Splat for amd64.
func (c *amd64Compiler) compileV128Splat(o *wazeroir.OperationV128Splat) (err error) {
	origin := c.locationStack.pop()
	if err = c.compileEnsureOnGeneralPurposeRegister(origin); err != nil {
		return
	}

	var result asm.Register
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		result, err = c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		c.locationStack.markRegisterUsed(result)

		tmp, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRB, origin.register, result, 0)
		c.assembler.CompileRegisterToRegister(amd64.PXOR, tmp, tmp)
		c.assembler.CompileRegisterToRegister(amd64.PSHUFB, tmp, result)
	case wazeroir.ShapeI16x8:
		result, err = c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		c.locationStack.markRegisterUsed(result)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRW, origin.register, result, 0)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRW, origin.register, result, 1)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PSHUFD, result, result, 0)
	case wazeroir.ShapeI32x4:
		result, err = c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		c.locationStack.markRegisterUsed(result)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRD, origin.register, result, 0)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PSHUFD, result, result, 0)
	case wazeroir.ShapeI64x2:
		result, err = c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		c.locationStack.markRegisterUsed(result)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRQ, origin.register, result, 0)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRQ, origin.register, result, 1)
	case wazeroir.ShapeF32x4:
		result = origin.register
		c.assembler.CompileRegisterToRegisterWithArg(amd64.INSERTPS, origin.register, result, 0)
		c.assembler.CompileRegisterToRegisterWithArg(amd64.PSHUFD, result, result, 0)
	case wazeroir.ShapeF64x2:
		result = origin.register
		c.assembler.CompileRegisterToRegister(amd64.MOVQ, origin.register, result)
		c.assembler.CompileRegisterToRegister(amd64.MOVLHPS, origin.register, result)
	}

	c.locationStack.markRegisterUnused(origin.register)
	c.pushVectorRuntimeValueLocationOnRegister(result)
	return nil
}

// compileV128Shuffle implements compiler.compileV128Shuffle for amd64.
func (c *amd64Compiler) compileV128Shuffle(o *wazeroir.OperationV128Shuffle) error {
	w := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(w); err != nil {
		return err
	}

	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	tmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	consts := [32]byte{}
	for i, lane := range o.Lanes {
		if lane < 16 {
			consts[i+16] = 0x80
			consts[i] = lane
		} else {
			consts[i+16] = lane - 16
			consts[i] = 0x80
		}
	}

	err = c.assembler.CompileLoadStaticConstToRegister(amd64.MOVDQU, consts[:16], tmp)
	if err != nil {
		return err
	}
	c.assembler.CompileRegisterToRegister(amd64.PSHUFB, tmp, v.register)
	err = c.assembler.CompileLoadStaticConstToRegister(amd64.MOVDQU, consts[16:], tmp)
	if err != nil {
		return err
	}
	c.assembler.CompileRegisterToRegister(amd64.PSHUFB, tmp, w.register)
	c.assembler.CompileRegisterToRegister(amd64.ORPS, v.register, w.register)

	c.pushVectorRuntimeValueLocationOnRegister(w.register)
	c.locationStack.markRegisterUnused(v.register)
	return nil
}

var swizzleConst = [16]byte{
	0x70, 0x70, 0x70, 0x70, 0x70, 0x70, 0x70, 0x70,
	0x70, 0x70, 0x70, 0x70, 0x70, 0x70, 0x70, 0x70,
}

// compileV128Swizzle implements compiler.compileV128Swizzle for amd64.
func (c *amd64Compiler) compileV128Swizzle(*wazeroir.OperationV128Swizzle) error {
	indexVec := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(indexVec); err != nil {
		return err
	}

	baseVec := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(baseVec); err != nil {
		return err
	}

	tmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	err = c.assembler.CompileLoadStaticConstToRegister(amd64.MOVDQU, swizzleConst[:], tmp)
	if err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(amd64.PADDUSB, tmp, indexVec.register)
	c.assembler.CompileRegisterToRegister(amd64.PSHUFB, indexVec.register, baseVec.register)

	c.pushVectorRuntimeValueLocationOnRegister(baseVec.register)
	c.locationStack.markRegisterUnused(indexVec.register)
	return nil
}

// compileV128AnyTrue implements compiler.compileV128AnyTrue for amd64.
func (c *amd64Compiler) compileV128AnyTrue(*wazeroir.OperationV128AnyTrue) error {
	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(amd64.PTEST, v.register, v.register)

	c.locationStack.pushRuntimeValueLocationOnConditionalRegister(amd64.ConditionalRegisterStateNE)
	c.locationStack.markRegisterUnused(v.register)
	return nil
}

// compileV128AllTrue implements compiler.compileV128AllTrue for amd64.
func (c *amd64Compiler) compileV128AllTrue(o *wazeroir.OperationV128AllTrue) error {
	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	tmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	var cmpInst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		cmpInst = amd64.PCMPEQB
	case wazeroir.ShapeI16x8:
		cmpInst = amd64.PCMPEQW
	case wazeroir.ShapeI32x4:
		cmpInst = amd64.PCMPEQD
	case wazeroir.ShapeI64x2:
		cmpInst = amd64.PCMPEQQ
	}

	c.assembler.CompileRegisterToRegister(amd64.PXOR, tmp, tmp)
	c.assembler.CompileRegisterToRegister(cmpInst, v.register, tmp)
	c.assembler.CompileRegisterToRegister(amd64.PTEST, tmp, tmp)
	c.locationStack.markRegisterUnused(v.register, tmp)
	c.locationStack.pushRuntimeValueLocationOnConditionalRegister(amd64.ConditionalRegisterStateE)
	return nil
}

// compileV128BitMask implements compiler.compileV128BitMask for amd64.
func (c *amd64Compiler) compileV128BitMask(o *wazeroir.OperationV128BitMask) error {
	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	result, err := c.allocateRegister(registerTypeGeneralPurpose)
	if err != nil {
		return err
	}

	switch o.Shape {
	case wazeroir.ShapeI8x16:
		c.assembler.CompileRegisterToRegister(amd64.PMOVMSKB, v.register, result)
	case wazeroir.ShapeI16x8:
		// When we have:
		// 	R1 = [R1(w1), R1(w2), R1(w3), R1(w4), R1(w5), R1(w6), R1(w7), R1(v8)]
		// 	R2 = [R2(w1), R2(w2), R2(w3), R2(v4), R2(w5), R2(w6), R2(w7), R2(v8)]
		//	where RX(wn) is n-th signed word (16-bit) of RX register,
		//
		// "PACKSSWB R1, R2" produces
		//  R1 = [
		// 		byte_sat(R1(w1)), byte_sat(R1(w2)), byte_sat(R1(w3)), byte_sat(R1(w4)),
		// 		byte_sat(R1(w5)), byte_sat(R1(w6)), byte_sat(R1(w7)), byte_sat(R1(w8)),
		// 		byte_sat(R2(w1)), byte_sat(R2(w2)), byte_sat(R2(w3)), byte_sat(R2(w4)),
		// 		byte_sat(R2(w5)), byte_sat(R2(w6)), byte_sat(R2(w7)), byte_sat(R2(w8)),
		//  ]
		//  where R1 is the destination register, and
		// 	byte_sat(w) = int8(w) if w fits as signed 8-bit,
		//                0x80 if w is less than 0x80
		//                0x7F if w is greater than 0x7f
		//
		// See https://www.felixcloutier.com/x86/packsswb:packssdw for detail.
		//
		// Therefore, v.register ends up having i-th and (i+8)-th bit set if i-th lane is negative (for i in 0..8).
		c.assembler.CompileRegisterToRegister(amd64.PACKSSWB, v.register, v.register)
		c.assembler.CompileRegisterToRegister(amd64.PMOVMSKB, v.register, result)
		// Clear the higher bits than 8.
		c.assembler.CompileConstToRegister(amd64.SHRQ, 8, result)
	case wazeroir.ShapeI32x4:
		c.assembler.CompileRegisterToRegister(amd64.MOVMSKPS, v.register, result)
	case wazeroir.ShapeI64x2:
		c.assembler.CompileRegisterToRegister(amd64.MOVMSKPD, v.register, result)
	}

	c.locationStack.markRegisterUnused(v.register)
	c.pushRuntimeValueLocationOnRegister(result, runtimeValueTypeI32)
	return nil
}

// compileV128And implements compiler.compileV128And for amd64.
func (c *amd64Compiler) compileV128And(*wazeroir.OperationV128And) error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(amd64.PAND, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128Not implements compiler.compileV128Not for amd64.
func (c *amd64Compiler) compileV128Not(*wazeroir.OperationV128Not) error {
	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	tmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	// Set all bits on tmp register.
	c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, tmp, tmp)
	// Then XOR with tmp to reverse all bits on v.register.
	c.assembler.CompileRegisterToRegister(amd64.PXOR, tmp, v.register)
	c.pushVectorRuntimeValueLocationOnRegister(v.register)
	return nil
}

// compileV128Or implements compiler.compileV128Or for amd64.
func (c *amd64Compiler) compileV128Or(*wazeroir.OperationV128Or) error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(amd64.POR, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128Xor implements compiler.compileV128Xor for amd64.
func (c *amd64Compiler) compileV128Xor(*wazeroir.OperationV128Xor) error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(amd64.PXOR, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128Bitselect implements compiler.compileV128Bitselect for amd64.
func (c *amd64Compiler) compileV128Bitselect(*wazeroir.OperationV128Bitselect) error {
	selector := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(selector); err != nil {
		return err
	}

	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	// The following logic is equivalent to v128.or(v128.and(v1, selector), v128.and(v2, v128.not(selector)))
	// See https://github.com/WebAssembly/spec/blob/main/proposals/simd/SIMD.md#bitwise-select
	c.assembler.CompileRegisterToRegister(amd64.PAND, selector.register, x1.register)
	c.assembler.CompileRegisterToRegister(amd64.PANDN, x2.register, selector.register)
	c.assembler.CompileRegisterToRegister(amd64.POR, selector.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register, selector.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128AndNot implements compiler.compileV128AndNot for amd64.
func (c *amd64Compiler) compileV128AndNot(*wazeroir.OperationV128AndNot) error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(amd64.PANDN, x1.register, x2.register)

	c.locationStack.markRegisterUnused(x1.register)
	c.pushVectorRuntimeValueLocationOnRegister(x2.register)
	return nil
}

// compileV128Shr implements compiler.compileV128Shr for amd64.
func (c *amd64Compiler) compileV128Shr(o *wazeroir.OperationV128Shr) error {
	// https://stackoverflow.com/questions/35002937/sse-simd-shift-with-one-byte-element-size-granularity
	if o.Shape == wazeroir.ShapeI8x16 {
		return c.compileV128ShrI8x16Impl(o.Signed)
	} else if o.Shape == wazeroir.ShapeI64x2 && o.Signed {
		return c.compileV128ShrI64x2SignedImpl()
	} else {
		return c.compileV128ShrImpl(o)
	}
}

// compileV128ShrImpl implements shift right instructions except for i8x16 (logical/arithmetic) and i64x2 (arithmetic).
func (c *amd64Compiler) compileV128ShrImpl(o *wazeroir.OperationV128Shr) error {
	s := c.locationStack.pop()
	if err := c.compileEnsureOnGeneralPurposeRegister(s); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	vecTmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	var moduleConst int64
	var shift asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI16x8:
		moduleConst = 0xf // modulo 16.
		if o.Signed {
			shift = amd64.PSRAW
		} else {
			shift = amd64.PSRLW
		}
	case wazeroir.ShapeI32x4:
		moduleConst = 0x1f // modulo 32.
		if o.Signed {
			shift = amd64.PSRAD
		} else {
			shift = amd64.PSRLD
		}
	case wazeroir.ShapeI64x2:
		moduleConst = 0x3f // modulo 64.
		shift = amd64.PSRLQ
	}

	gpShiftAmount := s.register
	c.assembler.CompileConstToRegister(amd64.ANDQ, moduleConst, gpShiftAmount)
	c.assembler.CompileRegisterToRegister(amd64.MOVL, gpShiftAmount, vecTmp)
	c.assembler.CompileRegisterToRegister(shift, vecTmp, x1.register)

	c.locationStack.markRegisterUnused(gpShiftAmount)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128ShrI64x2SignedImpl implements compiler.compileV128Shr for i64x2 signed (arithmetic) shift.
// PSRAQ instruction requires AVX, so we emulate it without AVX instructions. https://www.felixcloutier.com/x86/psraw:psrad:psraq
func (c *amd64Compiler) compileV128ShrI64x2SignedImpl() error {
	const shiftCountRegister = amd64.RegCX
	// If another value lives on the CX register, we release it to the stack.
	c.onValueReleaseRegisterToStack(shiftCountRegister)

	s := c.locationStack.pop()
	if s.onStack() {
		s.setRegister(shiftCountRegister)
		c.compileLoadValueOnStackToRegister(s)
	} else if s.onConditionalRegister() {
		c.compileMoveConditionalToGeneralPurposeRegister(s, shiftCountRegister)
	} else { // already on register.
		old := s.register
		c.assembler.CompileRegisterToRegister(amd64.MOVL, old, shiftCountRegister)
		s.setRegister(shiftCountRegister)
		c.locationStack.markRegisterUnused(old)
	}
	c.locationStack.markRegisterUnused(shiftCountRegister)
	tmp, err := c.allocateRegister(registerTypeGeneralPurpose)
	if err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	// Extract each lane into tmp, execute SHR on tmp, and write it back to the lane.
	c.assembler.CompileRegisterToRegisterWithArg(amd64.PEXTRQ, x1.register, tmp, 0)
	c.assembler.CompileRegisterToRegister(amd64.SARQ, shiftCountRegister, tmp)
	c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRQ, tmp, x1.register, 0)
	c.assembler.CompileRegisterToRegisterWithArg(amd64.PEXTRQ, x1.register, tmp, 1)
	c.assembler.CompileRegisterToRegister(amd64.SARQ, shiftCountRegister, tmp)
	c.assembler.CompileRegisterToRegisterWithArg(amd64.PINSRQ, tmp, x1.register, 1)

	c.locationStack.markRegisterUnused(shiftCountRegister)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// i8x16LogicalSHRMaskTable is necessary for emulating non-existent packed bytes logical right shifts on amd64.
// The mask is applied after performing packed word shifts on the value to clear out the unnecessary bits.
var i8x16LogicalSHRMaskTable = [8 * 16]byte{ // (the number of possible shift amount 0, 1, ..., 7.) * 16 bytes.
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // for 0 shift
	0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, // for 1 shift
	0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, // for 2 shift
	0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, // for 3 shift
	0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, // for 4 shift
	0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, 0x07, // for 5 shift
	0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, // for 6 shift
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, // for 7 shift
}

// compileV128ShrI64x2SignedImpl implements compiler.compileV128Shr for i8x16 signed logical/arithmetic shifts.
// amd64 doesn't have packed byte shifts, so we need this special casing.
// See https://stackoverflow.com/questions/35002937/sse-simd-shift-with-one-byte-element-size-granularity
func (c *amd64Compiler) compileV128ShrI8x16Impl(signed bool) error {
	s := c.locationStack.pop()
	if err := c.compileEnsureOnGeneralPurposeRegister(s); err != nil {
		return err
	}

	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	vecTmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	gpShiftAmount := s.register
	c.assembler.CompileConstToRegister(amd64.ANDQ, 0x7, gpShiftAmount) // mod 8.

	if signed {
		c.locationStack.markRegisterUsed(vecTmp)
		vecTmp2, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}

		vreg := v.register

		// Copy the value from v.register to vecTmp.
		c.assembler.CompileRegisterToRegister(amd64.MOVDQA, vreg, vecTmp)

		// Assuming that we have
		//  vreg   = [b1, ..., b16]
		//  vecTmp = [b1, ..., b16]
		// at this point, then we use PUNPCKLBW and PUNPCKHBW to produce:
		//  vreg   = [b1, b1, b2, b2, ..., b8, b8]
		//  vecTmp = [b9, b9, b10, b10, ..., b16, b16]
		c.assembler.CompileRegisterToRegister(amd64.PUNPCKLBW, vreg, vreg)
		c.assembler.CompileRegisterToRegister(amd64.PUNPCKHBW, vecTmp, vecTmp)

		// Adding 8 to the shift amount, and then move the amount to vecTmp2.
		c.assembler.CompileConstToRegister(amd64.ADDQ, 0x8, gpShiftAmount)
		c.assembler.CompileRegisterToRegister(amd64.MOVL, gpShiftAmount, vecTmp2)

		// Perform the word packed arithmetic right shifts on vreg and vecTmp.
		// This changes these two registers as:
		//  vreg   = [xxx, b1 >> s, xxx, b2 >> s, ..., xxx, b8 >> s]
		//  vecTmp = [xxx, b9 >> s, xxx, b10 >> s, ..., xxx, b16 >> s]
		// where xxx is 1 or 0 depending on each byte's sign, and ">>" is the arithmetic shift on a byte.
		c.assembler.CompileRegisterToRegister(amd64.PSRAW, vecTmp2, vreg)
		c.assembler.CompileRegisterToRegister(amd64.PSRAW, vecTmp2, vecTmp)

		// Finally, we can get the result by packing these two word vectors.
		c.assembler.CompileRegisterToRegister(amd64.PACKSSWB, vecTmp, vreg)

		c.locationStack.markRegisterUnused(gpShiftAmount, vecTmp)
		c.pushVectorRuntimeValueLocationOnRegister(vreg)
	} else {
		c.assembler.CompileRegisterToRegister(amd64.MOVL, gpShiftAmount, vecTmp)
		// amd64 doesn't have packed byte shifts, so we packed word shift here, and then mark-out
		// the unnecessary bits below.
		c.assembler.CompileRegisterToRegister(amd64.PSRLW, vecTmp, v.register)

		gpTmp, err := c.allocateRegister(registerTypeGeneralPurpose)
		if err != nil {
			return err
		}

		// Read the initial address of the mask table into gpTmp register.
		err = c.assembler.CompileLoadStaticConstToRegister(amd64.LEAQ, i8x16LogicalSHRMaskTable[:], gpTmp)
		if err != nil {
			return err
		}

		// We have to get the mask according to the shift amount, so we first have to do
		// gpShiftAmount << 4 = gpShiftAmount*16 to get the initial offset of the mask (16 is the size of each mask in bytes).
		c.assembler.CompileConstToRegister(amd64.SHLQ, 4, gpShiftAmount)

		// Now ready to read the content of the mask into the vecTmp.
		c.assembler.CompileMemoryWithIndexToRegister(amd64.MOVDQU,
			gpTmp, 0, gpShiftAmount, 1,
			vecTmp,
		)

		// Finally, clear out the unnecessary
		c.assembler.CompileRegisterToRegister(amd64.PAND, vecTmp, v.register)

		c.locationStack.markRegisterUnused(gpShiftAmount)
		c.pushVectorRuntimeValueLocationOnRegister(v.register)
	}
	return nil
}

// i8x16SHLMaskTable is necessary for emulating non-existent packed bytes left shifts on amd64.
// The mask is applied after performing packed word shifts on the value to clear out the unnecessary bits.
var i8x16SHLMaskTable = [8 * 16]byte{ // (the number of possible shift amount 0, 1, ..., 7.) * 16 bytes.
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // for 0 shift
	0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, // for 1 shift
	0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, 0xfc, // for 2 shift
	0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, 0xf8, // for 3 shift
	0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, // for 4 shift
	0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, 0xe0, // for 5 shift
	0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, // for 6 shift
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, // for 7 shift
}

// compileV128Shl implements compiler.compileV128Shl for amd64.
func (c *amd64Compiler) compileV128Shl(o *wazeroir.OperationV128Shl) error {
	s := c.locationStack.pop()
	if err := c.compileEnsureOnGeneralPurposeRegister(s); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	vecTmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	var modulo int64
	var shift asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		modulo = 0x7 // modulo 8.
		// x86 doesn't have packed bytes shift, so we use PSLLW and mask-out the redundant bits.
		// See https://stackoverflow.com/questions/35002937/sse-simd-shift-with-one-byte-element-size-granularity
		shift = amd64.PSLLW
	case wazeroir.ShapeI16x8:
		modulo = 0xf // modulo 16.
		shift = amd64.PSLLW
	case wazeroir.ShapeI32x4:
		modulo = 0x1f // modulo 32.
		shift = amd64.PSLLD
	case wazeroir.ShapeI64x2:
		modulo = 0x3f // modulo 64.
		shift = amd64.PSLLQ
	}

	gpShiftAmount := s.register
	c.assembler.CompileConstToRegister(amd64.ANDQ, modulo, gpShiftAmount)
	c.assembler.CompileRegisterToRegister(amd64.MOVL, gpShiftAmount, vecTmp)
	c.assembler.CompileRegisterToRegister(shift, vecTmp, x1.register)

	if o.Shape == wazeroir.ShapeI8x16 {
		gpTmp, err := c.allocateRegister(registerTypeGeneralPurpose)
		if err != nil {
			return err
		}

		// Read the initial address of the mask table into gpTmp register.
		err = c.assembler.CompileLoadStaticConstToRegister(amd64.LEAQ, i8x16SHLMaskTable[:], gpTmp)
		if err != nil {
			return err
		}

		// We have to get the mask according to the shift amount, so we first have to do
		// gpShiftAmount << 4 = gpShiftAmount*16 to get the initial offset of the mask (16 is the size of each mask in bytes).
		c.assembler.CompileConstToRegister(amd64.SHLQ, 4, gpShiftAmount)

		// Now ready to read the content of the mask into the vecTmp.
		c.assembler.CompileMemoryWithIndexToRegister(amd64.MOVDQU,
			gpTmp, 0, gpShiftAmount, 1,
			vecTmp,
		)

		// Finally, clear out the unnecessary
		c.assembler.CompileRegisterToRegister(amd64.PAND, vecTmp, x1.register)
	}

	c.locationStack.markRegisterUnused(gpShiftAmount)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128Cmp implements compiler.compileV128Cmp for amd64.
func (c *amd64Compiler) compileV128Cmp(o *wazeroir.OperationV128Cmp) error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	const (
		// See https://www.felixcloutier.com/x86/cmppd and https://www.felixcloutier.com/x86/cmpps
		floatEqualArg           = 0
		floatLessThanArg        = 1
		floatLessThanOrEqualArg = 2
		floatNotEqualARg        = 4
	)

	x1Reg, x2Reg, result := x1.register, x2.register, asm.NilRegister
	switch o.Type {
	case wazeroir.V128CmpTypeF32x4Eq:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPS, x2Reg, x1Reg, floatEqualArg)
		result = x1Reg
	case wazeroir.V128CmpTypeF32x4Ne:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPS, x2Reg, x1Reg, floatNotEqualARg)
		result = x1Reg
	case wazeroir.V128CmpTypeF32x4Lt:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPS, x2Reg, x1Reg, floatLessThanArg)
		result = x1Reg
	case wazeroir.V128CmpTypeF32x4Gt:
		// Without AVX, there's no float Gt instruction, so we swap the register and use Lt instead.
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPS, x1Reg, x2Reg, floatLessThanArg)
		result = x2Reg
	case wazeroir.V128CmpTypeF32x4Le:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPS, x2Reg, x1Reg, floatLessThanOrEqualArg)
		result = x1Reg
	case wazeroir.V128CmpTypeF32x4Ge:
		// Without AVX, there's no float Ge instruction, so we swap the register and use Le instead.
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPS, x1Reg, x2Reg, floatLessThanOrEqualArg)
		result = x2Reg
	case wazeroir.V128CmpTypeF64x2Eq:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPD, x2Reg, x1Reg, floatEqualArg)
		result = x1Reg
	case wazeroir.V128CmpTypeF64x2Ne:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPD, x2Reg, x1Reg, floatNotEqualARg)
		result = x1Reg
	case wazeroir.V128CmpTypeF64x2Lt:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPD, x2Reg, x1Reg, floatLessThanArg)
		result = x1Reg
	case wazeroir.V128CmpTypeF64x2Gt:
		// Without AVX, there's no float Gt instruction, so we swap the register and use Lt instead.
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPD, x1Reg, x2Reg, floatLessThanArg)
		result = x2Reg
	case wazeroir.V128CmpTypeF64x2Le:
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPD, x2Reg, x1Reg, floatLessThanOrEqualArg)
		result = x1Reg
	case wazeroir.V128CmpTypeF64x2Ge:
		// Without AVX, there's no float Ge instruction, so we swap the register and use Le instead.
		c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPD, x1Reg, x2Reg, floatLessThanOrEqualArg)
		result = x2Reg
	case wazeroir.V128CmpTypeI8x16Eq:
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQB, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI8x16Ne:
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQB, x2Reg, x1Reg)
		// Set all bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x2Reg)
		// Swap the bits on x1Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PXOR, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI8x16LtS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTB, x1Reg, x2Reg)
		result = x2Reg
	case wazeroir.V128CmpTypeI8x16LtU, wazeroir.V128CmpTypeI8x16GtU:
		// Take the unsigned min/max values on each byte on x1 and x2 onto x1Reg.
		if o.Type == wazeroir.V128CmpTypeI8x16LtU {
			c.assembler.CompileRegisterToRegister(amd64.PMINUB, x2Reg, x1Reg)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.PMAXUB, x2Reg, x1Reg)
		}
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQB, x2Reg, x1Reg)
		// Set all bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x2Reg)
		// Swap the bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PXOR, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI8x16GtS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTB, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI8x16LeS, wazeroir.V128CmpTypeI8x16LeU:
		tmp, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		// Copy the value on the src to tmp.
		c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1Reg, tmp)
		if o.Type == wazeroir.V128CmpTypeI8x16LeS {
			c.assembler.CompileRegisterToRegister(amd64.PMINSB, x2Reg, tmp)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.PMINUB, x2Reg, tmp)
		}
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQB, tmp, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI8x16GeS, wazeroir.V128CmpTypeI8x16GeU:
		tmp, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1Reg, tmp)
		if o.Type == wazeroir.V128CmpTypeI8x16GeS {
			c.assembler.CompileRegisterToRegister(amd64.PMAXSB, x2Reg, tmp)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.PMAXUB, x2Reg, tmp)
		}
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQB, tmp, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI16x8Eq:
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQW, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI16x8Ne:
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQW, x2Reg, x1Reg)
		// Set all bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x2Reg)
		// Swap the bits on x1Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PXOR, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI16x8LtS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTW, x1Reg, x2Reg)
		result = x2Reg
	case wazeroir.V128CmpTypeI16x8LtU, wazeroir.V128CmpTypeI16x8GtU:
		// Take the unsigned min/max values on each byte on x1 and x2 onto x1Reg.
		if o.Type == wazeroir.V128CmpTypeI16x8LtU {
			c.assembler.CompileRegisterToRegister(amd64.PMINUW, x2Reg, x1Reg)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.PMAXUW, x2Reg, x1Reg)
		}
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQW, x2Reg, x1Reg)
		// Set all bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x2Reg)
		// Swap the bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PXOR, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI16x8GtS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTW, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI16x8LeS, wazeroir.V128CmpTypeI16x8LeU:
		tmp, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		// Copy the value on the src to tmp.
		c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1Reg, tmp)
		if o.Type == wazeroir.V128CmpTypeI16x8LeS {
			c.assembler.CompileRegisterToRegister(amd64.PMINSW, x2Reg, tmp)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.PMINUW, x2Reg, tmp)
		}
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQW, tmp, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI16x8GeS, wazeroir.V128CmpTypeI16x8GeU:
		tmp, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1Reg, tmp)
		if o.Type == wazeroir.V128CmpTypeI16x8GeS {
			c.assembler.CompileRegisterToRegister(amd64.PMAXSW, x2Reg, tmp)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.PMAXUW, x2Reg, tmp)
		}
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQW, tmp, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI32x4Eq:
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI32x4Ne:
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x1Reg)
		// Set all bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x2Reg)
		// Swap the bits on x1Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PXOR, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI32x4LtS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTD, x1Reg, x2Reg)
		result = x2Reg
	case wazeroir.V128CmpTypeI32x4LtU, wazeroir.V128CmpTypeI32x4GtU:
		// Take the unsigned min/max values on each byte on x1 and x2 onto x1Reg.
		if o.Type == wazeroir.V128CmpTypeI32x4LtU {
			c.assembler.CompileRegisterToRegister(amd64.PMINUD, x2Reg, x1Reg)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.PMAXUD, x2Reg, x1Reg)
		}
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x1Reg)
		// Set all bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x2Reg)
		// Swap the bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PXOR, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI32x4GtS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTD, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI32x4LeS, wazeroir.V128CmpTypeI32x4LeU:
		tmp, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		// Copy the value on the src to tmp.
		c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1Reg, tmp)
		if o.Type == wazeroir.V128CmpTypeI32x4LeS {
			c.assembler.CompileRegisterToRegister(amd64.PMINSD, x2Reg, tmp)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.PMINUD, x2Reg, tmp)
		}
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, tmp, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI32x4GeS, wazeroir.V128CmpTypeI32x4GeU:
		tmp, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1Reg, tmp)
		if o.Type == wazeroir.V128CmpTypeI32x4GeS {
			c.assembler.CompileRegisterToRegister(amd64.PMAXSD, x2Reg, tmp)
		} else {
			c.assembler.CompileRegisterToRegister(amd64.PMAXUD, x2Reg, tmp)
		}
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, tmp, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI64x2Eq:
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQQ, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI64x2Ne:
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQQ, x2Reg, x1Reg)
		// Set all bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x2Reg)
		// Swap the bits on x1Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PXOR, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI64x2LtS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTQ, x1Reg, x2Reg)
		result = x2Reg
	case wazeroir.V128CmpTypeI64x2GtS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTQ, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI64x2LeS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTQ, x2Reg, x1Reg)
		// Set all bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x2Reg, x2Reg)
		// Swap the bits on x1Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PXOR, x2Reg, x1Reg)
		result = x1Reg
	case wazeroir.V128CmpTypeI64x2GeS:
		c.assembler.CompileRegisterToRegister(amd64.PCMPGTQ, x1Reg, x2Reg)
		// Set all bits on x1Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, x1Reg, x1Reg)
		// Swap the bits on x2Reg register.
		c.assembler.CompileRegisterToRegister(amd64.PXOR, x1Reg, x2Reg)
		result = x2Reg
	}

	c.locationStack.markRegisterUnused(x1Reg, x2Reg)
	c.pushVectorRuntimeValueLocationOnRegister(result)
	return nil
}

// compileV128AddSat implements compiler.compileV128AddSat for amd64.
func (c *amd64Compiler) compileV128AddSat(o *wazeroir.OperationV128AddSat) error {
	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		if o.Signed {
			inst = amd64.PADDSB
		} else {
			inst = amd64.PADDUSB
		}
	case wazeroir.ShapeI16x8:
		if o.Signed {
			inst = amd64.PADDSW
		} else {
			inst = amd64.PADDUSW
		}
	}

	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(inst, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128SubSat implements compiler.compileV128SubSat for amd64.
func (c *amd64Compiler) compileV128SubSat(o *wazeroir.OperationV128SubSat) error {
	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		if o.Signed {
			inst = amd64.PSUBSB
		} else {
			inst = amd64.PSUBUSB
		}
	case wazeroir.ShapeI16x8:
		if o.Signed {
			inst = amd64.PSUBSW
		} else {
			inst = amd64.PSUBUSW
		}
	}

	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(inst, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128Mul implements compiler.compileV128Mul for amd64.
func (c *amd64Compiler) compileV128Mul(o *wazeroir.OperationV128Mul) error {
	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI16x8:
		inst = amd64.PMULLW
	case wazeroir.ShapeI32x4:
		inst = amd64.PMULLD
	case wazeroir.ShapeI64x2:
		return c.compileV128MulI64x2()
	case wazeroir.ShapeF32x4:
		inst = amd64.MULPS
	case wazeroir.ShapeF64x2:
		inst = amd64.MULPD
	}

	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(inst, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128MulI64x2 implements V128Mul for i64x2.
func (c *amd64Compiler) compileV128MulI64x2() error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	x1r, x2r := x1.register, x2.register

	tmp1, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	c.locationStack.markRegisterUsed(tmp1)

	tmp2, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	// Assuming that we have
	//	x1r = [p1, p2] = [p1_lo, p1_hi, p2_lo, p2_high]
	//  x2r = [q1, q2] = [q1_lo, q1_hi, q2_lo, q2_high]
	// where pN and qN are 64-bit (quad word) lane, and pN_lo, pN_hi, qN_lo and qN_hi are 32-bit (double word) lane.

	// Copy x1's value into tmp1.
	c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1r, tmp1)
	// And do the logical right shift by 32-bit on tmp1, which makes tmp1 = [0, p1_high, 0, p2_high]
	c.assembler.CompileConstToRegister(amd64.PSRLQ, 32, tmp1)

	// Execute "pmuludq x2r,tmp1", which makes tmp1 = [p1_high*q1_lo, p2_high*q2_lo] where each lane is 64-bit.
	c.assembler.CompileRegisterToRegister(amd64.PMULUDQ, x2r, tmp1)

	// Copy x2's value into tmp2.
	c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x2r, tmp2)
	// And do the logical right shift by 32-bit on tmp2, which makes tmp2 = [0, q1_high, 0, q2_high]
	c.assembler.CompileConstToRegister(amd64.PSRLQ, 32, tmp2)

	// Execute "pmuludq x1r,tmp2", which makes tmp2 = [p1_lo*q1_high, p2_lo*q2_high] where each lane is 64-bit.
	c.assembler.CompileRegisterToRegister(amd64.PMULUDQ, x1r, tmp2)

	// Adds tmp1 and tmp2 and do the logical left shift by 32-bit,
	// which makes tmp1 = [(p1_lo*q1_high+p1_high*q1_lo)<<32, (p2_lo*q2_high+p2_high*q2_lo)<<32]
	c.assembler.CompileRegisterToRegister(amd64.PADDQ, tmp2, tmp1)
	c.assembler.CompileConstToRegister(amd64.PSLLQ, 32, tmp1)

	// Execute "pmuludq x2r,x1r", which makes x1r = [p1_lo*q1_lo, p2_lo*q2_lo] where each lane is 64-bit.
	c.assembler.CompileRegisterToRegister(amd64.PMULUDQ, x2r, x1r)

	// Finally, we get the result by adding x1r and tmp1,
	// which makes x1r = [(p1_lo*q1_high+p1_high*q1_lo)<<32+p1_lo*q1_lo, (p2_lo*q2_high+p2_high*q2_lo)<<32+p2_lo*q2_lo]
	c.assembler.CompileRegisterToRegister(amd64.PADDQ, tmp1, x1r)

	c.locationStack.markRegisterUnused(x2r, tmp1)
	c.pushVectorRuntimeValueLocationOnRegister(x1r)
	return nil
}

// compileV128Div implements compiler.compileV128Div for amd64.
func (c *amd64Compiler) compileV128Div(o *wazeroir.OperationV128Div) error {

	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeF32x4:
		inst = amd64.DIVPS
	case wazeroir.ShapeF64x2:
		inst = amd64.DIVPD
	}

	c.assembler.CompileRegisterToRegister(inst, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128Neg implements compiler.compileV128Neg for amd64.
func (c *amd64Compiler) compileV128Neg(o *wazeroir.OperationV128Neg) error {
	if o.Shape <= wazeroir.ShapeI64x2 {
		return c.compileV128NegInt(o.Shape)
	} else {
		return c.compileV128NegFloat(o.Shape)
	}
}

// compileV128NegInt implements compiler.compileV128Neg for integer lanes.
func (c *amd64Compiler) compileV128NegInt(s wazeroir.Shape) error {
	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	result, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	var subInst asm.Instruction
	switch s {
	case wazeroir.ShapeI8x16:
		subInst = amd64.PSUBB
	case wazeroir.ShapeI16x8:
		subInst = amd64.PSUBW
	case wazeroir.ShapeI32x4:
		subInst = amd64.PSUBD
	case wazeroir.ShapeI64x2:
		subInst = amd64.PSUBQ
	}

	c.assembler.CompileRegisterToRegister(amd64.PXOR, result, result)
	c.assembler.CompileRegisterToRegister(subInst, v.register, result)

	c.locationStack.markRegisterUnused(v.register)
	c.pushVectorRuntimeValueLocationOnRegister(result)
	return nil
}

// compileV128NegInt implements compiler.compileV128Neg for float lanes.
func (c *amd64Compiler) compileV128NegFloat(s wazeroir.Shape) error {
	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	tmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	var leftShiftInst, xorInst asm.Instruction
	var leftShiftAmount asm.ConstantValue
	if s == wazeroir.ShapeF32x4 {
		leftShiftInst, leftShiftAmount, xorInst = amd64.PSLLD, 31, amd64.XORPS
	} else {
		leftShiftInst, leftShiftAmount, xorInst = amd64.PSLLQ, 63, amd64.XORPD
	}

	// Set all bits on tmp by CMPPD with arg=0 (== pseudo CMPEQPS instruction).
	// See https://www.felixcloutier.com/x86/cmpps
	c.assembler.CompileRegisterToRegisterWithArg(amd64.CMPPD, tmp, tmp, 0)
	// Do the left shift on each lane to set only the most significant bit in each.
	c.assembler.CompileConstToRegister(leftShiftInst, leftShiftAmount, tmp)
	// Get the negated result by XOR on each lane with tmp.
	c.assembler.CompileRegisterToRegister(xorInst, tmp, v.register)

	c.pushVectorRuntimeValueLocationOnRegister(v.register)
	return nil
}

// compileV128Sqrt implements compiler.compileV128Sqrt for amd64.
func (c *amd64Compiler) compileV128Sqrt(o *wazeroir.OperationV128Sqrt) error {
	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeF64x2:
		inst = amd64.SQRTPD
	case wazeroir.ShapeF32x4:
		inst = amd64.SQRTPS
	}

	c.assembler.CompileRegisterToRegister(inst, v.register, v.register)
	c.pushVectorRuntimeValueLocationOnRegister(v.register)
	return nil
}

// compileV128Abs implements compiler.compileV128Abs for amd64.
func (c *amd64Compiler) compileV128Abs(o *wazeroir.OperationV128Abs) error {
	if o.Shape == wazeroir.ShapeI64x2 {
		return c.compileV128AbsI64x2()
	}

	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}

	result := v.register
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		c.assembler.CompileRegisterToRegister(amd64.PABSB, result, result)
	case wazeroir.ShapeI16x8:
		c.assembler.CompileRegisterToRegister(amd64.PABSW, result, result)
	case wazeroir.ShapeI32x4:
		c.assembler.CompileRegisterToRegister(amd64.PABSD, result, result)
	case wazeroir.ShapeF32x4:
		tmp, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		// Set all bits on tmp.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, tmp, tmp)
		// Shift right packed single floats by 1 to clear the sign bits.
		c.assembler.CompileConstToRegister(amd64.PSRLD, 1, tmp)
		// Clear the sign bit of vr.
		c.assembler.CompileRegisterToRegister(amd64.ANDPS, tmp, result)
	case wazeroir.ShapeF64x2:
		tmp, err := c.allocateRegister(registerTypeVector)
		if err != nil {
			return err
		}
		// Set all bits on tmp.
		c.assembler.CompileRegisterToRegister(amd64.PCMPEQD, tmp, tmp)
		// Shift right packed single floats by 1 to clear the sign bits.
		c.assembler.CompileConstToRegister(amd64.PSRLQ, 1, tmp)
		// Clear the sign bit of vr.
		c.assembler.CompileRegisterToRegister(amd64.ANDPD, tmp, result)
	}

	c.pushVectorRuntimeValueLocationOnRegister(result)
	return nil
}

// compileV128AbsI64x2 implements compileV128Abs for i64x2 lanes.
func (c *amd64Compiler) compileV128AbsI64x2() error {
	// See https://www.felixcloutier.com/x86/blendvpd
	const blendMaskReg = amd64.RegX0
	c.onValueReleaseRegisterToStack(blendMaskReg)
	c.locationStack.markRegisterUsed(blendMaskReg)

	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}
	vr := v.register

	if vr == blendMaskReg {
		return errors.New("BUG: X0 must not be used")
	}

	tmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}
	c.locationStack.markRegisterUsed(tmp)

	// Copy the value to tmp.
	c.assembler.CompileRegisterToRegister(amd64.MOVDQA, vr, tmp)

	// Clear all bits on blendMaskReg.
	c.assembler.CompileRegisterToRegister(amd64.PXOR, blendMaskReg, blendMaskReg)
	// Subtract vr from blendMaskReg.
	c.assembler.CompileRegisterToRegister(amd64.PSUBQ, vr, blendMaskReg)
	// Copy the subtracted value ^^ back into vr.
	c.assembler.CompileRegisterToRegister(amd64.MOVDQA, blendMaskReg, vr)

	c.assembler.CompileRegisterToRegister(amd64.BLENDVPD, tmp, vr)

	c.locationStack.markRegisterUnused(blendMaskReg, tmp)
	c.pushVectorRuntimeValueLocationOnRegister(vr)
	return nil
}

var (
	popcntMask = [32]byte{
		0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f,
		0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f,
	}
	popcntTable = [32]byte{
		0x00, 0x01, 0x01, 0x02, 0x01, 0x02, 0x02, 0x03,
		0x01, 0x02, 0x02, 0x03, 0x02, 0x03, 0x03, 0x04,
	}
)

// compileV128Popcnt implements compiler.compileV128Popcnt for amd64.
func (c *amd64Compiler) compileV128Popcnt(*wazeroir.OperationV128Popcnt) error {
	v := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(v); err != nil {
		return err
	}
	vr := v.register

	tmp1, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	c.locationStack.markRegisterUsed(tmp1)

	tmp2, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	c.locationStack.markRegisterUsed(tmp2)

	tmp3, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	// Read the popcntMask into tmp1, and we have
	//  tmp1 = [0xf, ..., 0xf]
	if err := c.assembler.CompileLoadStaticConstToRegister(amd64.MOVDQU, popcntMask[:], tmp1); err != nil {
		return err
	}

	// Copy the original value into tmp2.
	c.assembler.CompileRegisterToRegister(amd64.MOVDQA, vr, tmp2)

	// Given that we have:
	//  v = [b1, ..., b16] where bn = hn:ln and hn and ln are higher and lower 4-bits of bn.
	//
	// Take PAND on tmp1 and tmp2, and we have
	//  tmp2 = [l1, ..., l16].
	c.assembler.CompileRegisterToRegister(amd64.PAND, tmp1, tmp2)

	// Do logical (packed word) right shift by 4 on vr and PAND with vr and tmp1, meaning that we have
	//  vr = [h1, ...., h16].
	c.assembler.CompileConstToRegister(amd64.PSRLW, 4, vr)
	c.assembler.CompileRegisterToRegister(amd64.PAND, tmp1, vr)

	// Read the popcntTable into tmp1, and we have
	//  tmp1 = [0x00, 0x01, 0x01, 0x02, 0x01, 0x02, 0x02, 0x03, 0x01, 0x02, 0x02, 0x03, 0x02, 0x03, 0x03, 0x04]
	if err := c.assembler.CompileLoadStaticConstToRegister(amd64.MOVDQU, popcntTable[:], tmp1); err != nil {
		return err
	}

	// Copy the tmp1 into tmp3, and we have
	//  tmp3 = [0x00, 0x01, 0x01, 0x02, 0x01, 0x02, 0x02, 0x03, 0x01, 0x02, 0x02, 0x03, 0x02, 0x03, 0x03, 0x04]
	c.assembler.CompileRegisterToRegister(amd64.MOVDQU, tmp1, tmp3)

	//  tmp3 = [popcnt(l1), ..., popcnt(l16)].
	c.assembler.CompileRegisterToRegister(amd64.PSHUFB, tmp2, tmp3)

	//  tmp1 = [popcnt(h1), ..., popcnt(h16)].
	c.assembler.CompileRegisterToRegister(amd64.PSHUFB, vr, tmp1)

	// vr = tmp1 = [popcnt(h1), ..., popcnt(h16)].
	c.assembler.CompileRegisterToRegister(amd64.MOVDQA, tmp1, vr)

	// vr += tmp3 = [popcnt(h1)+popcnt(l1), ..., popcnt(h16)+popcnt(l16)] = [popcnt(b1), ..., popcnt(b16)].
	c.assembler.CompileRegisterToRegister(amd64.PADDB, tmp3, vr)

	c.locationStack.markRegisterUnused(tmp1, tmp2)
	c.pushVectorRuntimeValueLocationOnRegister(vr)
	return nil
}

// compileV128Min implements compiler.compileV128Min for amd64.
func (c *amd64Compiler) compileV128Min(o *wazeroir.OperationV128Min) error {
	if o.Shape >= wazeroir.ShapeF32x4 {
		return c.compileV128MinOrMaxFloat(o.Shape, true)
	}

	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		if o.Signed {
			inst = amd64.PMINSB
		} else {
			inst = amd64.PMINUB
		}
	case wazeroir.ShapeI16x8:
		if o.Signed {
			inst = amd64.PMINSW
		} else {
			inst = amd64.PMINUW
		}
	case wazeroir.ShapeI32x4:
		if o.Signed {
			inst = amd64.PMINSD
		} else {
			inst = amd64.PMINUD
		}
	}

	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(inst, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128MinOrMaxFloat implements compiler.compileV128Min and compiler.compileV128Max for float lanes.
func (c *amd64Compiler) compileV128MinOrMaxFloat(o wazeroir.Shape, isMin bool) error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	x1r, x2r := x1.register, x2.register

	tmp, err := c.allocateRegister(registerTypeVector)
	if err != nil {
		return err
	}

	var minOrMaxInst, cmpInst, andnInst, orInst, logicalRightShiftInst asm.Instruction
	var shiftNumToInverseNaN asm.ConstantValue
	if o == wazeroir.ShapeF32x4 {
		cmpInst, andnInst, orInst, logicalRightShiftInst, shiftNumToInverseNaN =
			amd64.CMPPS, amd64.ANDNPS, amd64.ORPS, amd64.PSRLD, 0xa
		if isMin {
			minOrMaxInst = amd64.MINPS
		} else {
			minOrMaxInst = amd64.MAXPS
		}
	} else {
		cmpInst, andnInst, orInst, logicalRightShiftInst, shiftNumToInverseNaN =
			amd64.CMPPD, amd64.ANDNPD, amd64.ORPD, amd64.PSRLQ, 0xd
		if isMin {
			minOrMaxInst = amd64.MINPD
		} else {
			minOrMaxInst = amd64.MAXPD
		}
	}

	// Copy the value on x1 to tmp.
	c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1r, tmp)

	// Denote the original x1r and x2r 's vector as v1 and v2 below.
	//
	// Execute MINPS/MINPD/MAXPS/MAXPD with destination = tmp (holding v1), and we have
	//  tmp = [ if (v1[i] != NaN && v2[i] != NaN) {min_max(v1[i], v2[i])} else {v1[i]} for i in 0..LANE_NUM]
	c.assembler.CompileRegisterToRegister(minOrMaxInst, x2r, tmp)

	// Execute MINPS/MINPD/MAXPS/MAXPD with destination = x2r (holding v2), and we have
	//  x2r = [ if (v1[i] != NaN && v2[i] != NaN) {min_max(v1[i], v2[i])} else {v2[i]} for i in 0..LANE_NUM]
	c.assembler.CompileRegisterToRegister(minOrMaxInst, x1r, x2r)

	// Copy the current tmp into x1r.
	c.assembler.CompileRegisterToRegister(amd64.MOVDQA, tmp, x1r)

	// Set all bits on the lane where either v1[i] or v2[i] is NaN by via CMPPS/CMPPD (arg=3).
	// That means, we have:
	//  x1r =  [ if (v1[i] != NaN && v2[i] != NaN) {0} else {^0} for i in 0..4]
	//
	// See https://www.felixcloutier.com/x86/cmpps.
	c.assembler.CompileRegisterToRegisterWithArg(cmpInst, x2r, x1r, 3)

	// Mask all the lanes where either v1[i] or v2[i] is NaN, meaning that we have
	//  tmp = [ if (v1[i] != NaN && v2[i] != NaN) {min_max(v1[i], v2[i])} else {^0} for i in 0..LANE_NUM]
	c.assembler.CompileRegisterToRegister(orInst, x1r, tmp)

	// Put the inverse of NaN if either v1[i] or v2[i] is NaN on each lane, otherwise zero on x1r.
	// That means, we have:
	//  x1r =  [ if (v1[i] != NaN && v2[i] != NaN) {0} else {^NaN} for i in 0..LANE_NUM]
	//
	c.assembler.CompileConstToRegister(logicalRightShiftInst, shiftNumToInverseNaN, x1r)

	// Finally, we get the result but putting NaNs on each lane where either of v1[i] or v2[i] is NaN, otherwise min_max(v1[i], v2[i]).
	// That means, we have:
	//  x1r = [ if (v1[i] != NaN && v2[i] != NaN) {min_max(v1[i], v2[i])}  else {NaN} for i in 0..LANE_NUM]
	c.assembler.CompileRegisterToRegister(andnInst, tmp, x1r)

	c.locationStack.markRegisterUnused(x2r)
	c.pushVectorRuntimeValueLocationOnRegister(x1r)
	return nil
}

// compileV128Max implements compiler.compileV128Max for amd64.
func (c *amd64Compiler) compileV128Max(o *wazeroir.OperationV128Max) error {
	if o.Shape >= wazeroir.ShapeF32x4 {
		return c.compileV128MinOrMaxFloat(o.Shape, false)
	}

	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		if o.Signed {
			inst = amd64.PMAXSB
		} else {
			inst = amd64.PMAXUB
		}
	case wazeroir.ShapeI16x8:
		if o.Signed {
			inst = amd64.PMAXSW
		} else {
			inst = amd64.PMAXUW
		}
	case wazeroir.ShapeI32x4:
		if o.Signed {
			inst = amd64.PMAXSD
		} else {
			inst = amd64.PMAXUD
		}
	}

	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	c.assembler.CompileRegisterToRegister(inst, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}

// compileV128MaxFloat implements compiler.compileV128Max for float lanes.
func (c *amd64Compiler) compileV128MaxFloat(o wazeroir.Shape) error {
	//
	//x2 := c.locationStack.popV128()
	//if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
	//	return err
	//}
	//
	//x1 := c.locationStack.popV128()
	//if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
	//	return err
	//}
	//
	//x1r, x2r := x1.register, x2.register
	//
	//tmp, err := c.allocateRegister(registerTypeVector)
	//if err != nil {
	//	return err
	//}
	//
	//var maxInst, cmpInst, andnInst, xorInst, orInst, logicalRightShiftInst asm.Instruction
	//var shiftNumToInverseNaN asm.ConstantValue
	//if o == wazeroir.ShapeF32x4 {
	//	maxInst, cmpInst, andnInst, xorInst, orInst, logicalRightShiftInst, shiftNumToInverseNaN =
	//		amd64.MAXPS, amd64.CMPPS, amd64.ANDNPS, amd64.XORPS, amd64.ORPS, amd64.PSRLD, 0xa
	//} else {
	//	maxInst, cmpInst, andnInst, xorInst, orInst, logicalRightShiftInst, shiftNumToInverseNaN =
	//		amd64.MAXPD, amd64.CMPPD, amd64.ANDNPD, amd64.XORPD, amd64.ORPD, amd64.PSRLQ, 0xd
	//}
	//
	//// Copy the value on x1 to tmp.
	//c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1r, tmp)
	//
	//// Denote the original x1r and x2r 's vector as v1 and v2 below.
	////
	//// Execute MAXPS/MAXPD with source = x2r (holding v2), and we have
	////  x1r[i] = max(v1[i], v2[i]) if v1[i] != NaN && v2[i] != NaN, otherwise v1[i].
	//c.assembler.CompileRegisterToRegister(maxInst, x2r, x1r)
	//
	//// Execute MAXPS/MAXPD with source = tmp (holding v1), and we have
	////  x2r[i] = max(v1[i], v2[i]) if v1[i] != NaN && v2[i] != NaN, otherwise v2[i].
	//c.assembler.CompileRegisterToRegister(maxInst, tmp, x2r)
	//
	//// Copy the current x1r into tmp, meaning that:
	////  tmp[i] = max(v1[i], v2[i]) if v1[i] != NaN && v2[i] != NaN, otherwise v1[i].
	//c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1r, tmp)
	//
	//// XOR tmp and x2r, meaning that:
	////  tmp[i] = tmp[i]^x2r[i] = 0                   if v1[i] != NaN && v2[i] != NaN
	////                         = v1[i]^v2[i]         if v1[i] == NaN && v2[i] != NaN
	////                         = v1[i]^v2[i]         if v1[i] != NaN && v2[i] == NaN
	////                         = Any non nan         otherwise
	//c.assembler.CompileRegisterToRegister(xorInst, x2r, tmp)
	//
	//// OR tmp and x1r, meaning that:
	////  x1r[i] = x1r[i] | tmp[i] = max(v1[i], v2[i]) | 0 = max(v1[i], v2[i])    if v1[i] != NaN && v2[i] != NaN
	////                           = v1[i]|(v1[i]^v2[i])   = NaN                  if v1[i] == NaN && v2[i] != NaN
	////                           = v1[i]|(v1[i]^v2[i])   = NaN                  if v1[i] != NaN && v2[i] == NaN
	////                           = NaN                                          otherwise
	//c.assembler.CompileRegisterToRegister(orInst, tmp, x1r)
	//
	//// Copy the current x1r into x2r, meaning that:
	////  x2r[i] = NaN if one of operand is NaN, max(v1[0], v2[0]) otherwise.
	//c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1r, x2r)
	//
	//// Subtract tmp from x2r, and we have:
	////  x2r[i] = x2r[i] -
	//
	//// Copy the current x1r into x2r, therefore, we have:
	////  x2r[i] = NaN(in any form) if one of v0[i] or v1[i] is NaN, otherwise max(v1[i], v2[i]).
	//c.assembler.CompileRegisterToRegister(amd64.MOVDQA, x1r, x2r)
	//
	//c.locationStack.markRegisterUnused(x2r)
	//c.pushVectorRuntimeValueLocationOnRegister(x1r)
	return nil
}

// compileV128AvgrU implements compiler.compileV128AvgrU for amd64.
func (c *amd64Compiler) compileV128AvgrU(o *wazeroir.OperationV128AvgrU) error {
	x2 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x2); err != nil {
		return err
	}

	x1 := c.locationStack.popV128()
	if err := c.compileEnsureOnGeneralPurposeRegister(x1); err != nil {
		return err
	}

	var inst asm.Instruction
	switch o.Shape {
	case wazeroir.ShapeI8x16:
		inst = amd64.PAVGB
	case wazeroir.ShapeI16x8:
		inst = amd64.PAVGW
	}

	c.assembler.CompileRegisterToRegister(inst, x2.register, x1.register)

	c.locationStack.markRegisterUnused(x2.register)
	c.pushVectorRuntimeValueLocationOnRegister(x1.register)
	return nil
}
