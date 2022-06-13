package compiler

import (
	"github.com/tetratelabs/wazero/internal/wazeroir"
)

// compiler is the interface of architecture-specific native code compiler,
// and this is responsible for compiling native code for all wazeroir operations.
type compiler interface {
	// String is for debugging purpose.
	String() string
	// compilePreamble is called before compiling any wazeroir operation.
	// This is used, for example, to initialize the reserved registers, etc.
	compilePreamble() error
	// compile generates the byte slice of native code.
	// stackPointerCeil is the max stack pointer that the target function would reach.
	// staticData is codeStaticData for the resulting native code.
	compile() (code []byte, staticData codeStaticData, stackPointerCeil uint64, err error)
	// compileHostFunction emits the trampoline code from which native code can jump into the host function.
	// TODO: maybe we wouldn't need to have trampoline for host functions.
	compileHostFunction() error
	// compileLabel notify compilers of the beginning of a label.
	// Return true if the compiler decided to skip the entire label.
	// See wazeroir.OperationLabel
	compileLabel(o *wazeroir.OperationLabel) (skipThisLabel bool)
	// compileUnreachable adds instructions to return from the native code with nativeCallStatusCodeUnreachable status.
	// See wazeroir.OperationUnreachable.
	compileUnreachable() error
	// compileSwap adds instruction to perform wazeroir.OperationSwap.
	compileSwap(o *wazeroir.OperationSwap) error
	// compileGlobalGet adds instructions to perform wazeroir.OperationGlobalGet.
	compileGlobalGet(o *wazeroir.OperationGlobalGet) error
	// compileGlobalSet adds instructions to perform wazeroir.OperationGlobalSet.
	compileGlobalSet(o *wazeroir.OperationGlobalSet) error
	// compileBr adds instructions to perform wazeroir.OperationBr.
	compileBr(o *wazeroir.OperationBr) error
	// compileBrIf adds instructions to perform wazeroir.OperationBrIf.
	compileBrIf(o *wazeroir.OperationBrIf) error
	// compileBrTable adds instructions to perform wazeroir.OperationBrTable.
	compileBrTable(o *wazeroir.OperationBrTable) error
	// compileCall adds instructions to perform wazeroir.OperationCall.
	compileCall(o *wazeroir.OperationCall) error
	// compileCallIndirect adds instructions to perform wazeroir.OperationCallIndirect.
	compileCallIndirect(o *wazeroir.OperationCallIndirect) error
	// compileDrop adds instructions to perform wazeroir.OperationDrop.
	compileDrop(o *wazeroir.OperationDrop) error
	// compileSelect uses top three values on the stack. For example, if we have stack as [..., x1, x2, c]
	// and the value "c" equals zero, then the stack results in [..., x1], otherwise, [..., x2].
	// See wasm.OpcodeSelect
	compileSelect() error
	// compilePick adds instructions to perform wazeroir.OperationPick.
	compilePick(o *wazeroir.OperationPick) error
	// compileAdd adds instructions to pop two values from the stack, add these two values, and push
	// back the result onto the stack.
	// See wasm.OpcodeI32Add wasm.OpcodeI64Add wasm.OpcodeF32Add wasm.OpcodeF64Add
	compileAdd(o *wazeroir.OperationAdd) error
	// compileSub adds instructions to pop two values from the stack, subtract the top from the second one, and push
	// back the result onto the stack.
	// See wasm.OpcodeI32Sub wasm.OpcodeI64Sub wasm.OpcodeF32Sub wasm.OpcodeF64Sub
	compileSub(o *wazeroir.OperationSub) error
	// compileMul adds instructions to pop two values from the stack, multiply these two values, and push
	// back the result onto the stack.
	// See wasm.OpcodeI32Mul wasm.OpcodeI64Mul wasm.OpcodeF32Mul wasm.OpcodeF64Mul
	compileMul(o *wazeroir.OperationMul) error
	// compileClz emits instructions to count up the leading zeros in the
	// current top of the stack, and push the count result.
	// For example, stack of [..., 0x00_ff_ff_ff] results in [..., 8].
	// See wasm.OpcodeI32Clz wasm.OpcodeI64Clz
	compileClz(o *wazeroir.OperationClz) error
	// compileCtz emits instructions to count up the trailing zeros in the
	// current top of the stack, and push the count result.
	// For example, stack of [..., 0xff_ff_ff_00] results in [..., 8].
	// See wasm.OpcodeI32Ctz wasm.OpcodeI64Ctz
	compileCtz(o *wazeroir.OperationCtz) error
	// compilePopcnt emits instructions to count up the number of set bits in the
	// current top of the stack, and push the count result.
	// For example, stack of [..., 0b00_00_00_11] results in [..., 2].
	// See wasm.OpcodeI32Popcnt wasm.OpcodeI64Popcnt
	compilePopcnt(o *wazeroir.OperationPopcnt) error
	// compileDiv emits the instructions to perform division on the top two values on the stack.
	// See wasm.OpcodeI32DivS wasm.OpcodeI32DivU wasm.OpcodeI64DivS wasm.OpcodeI64DivU wasm.OpcodeF32Div wasm.OpcodeF64Div
	compileDiv(o *wazeroir.OperationDiv) error
	// compileRem emits the instructions to perform division on the top
	// two values of integer type on the stack and puts the remainder of the result
	// onto the stack. For example, stack [..., 10, 3] results in [..., 1] where
	// the quotient is discarded.
	// See wasm.OpcodeI32RemS wasm.OpcodeI32RemU wasm.OpcodeI64RemS wasm.OpcodeI64RemU
	compileRem(o *wazeroir.OperationRem) error
	// compileAnd emits instructions to perform logical "and" operation on
	// top two values on the stack, and push the result.
	// See wasm.OpcodeI32And wasm.OpcodeI64And
	compileAnd(o *wazeroir.OperationAnd) error
	// compileOr emits instructions to perform logical "or" operation on
	// top two values on the stack, and pushes the result.
	// See wasm.OpcodeI32Or wasm.OpcodeI64Or
	compileOr(o *wazeroir.OperationOr) error
	// compileXor emits instructions to perform logical "xor" operation on
	// top two values on the stack, and pushes the result.
	// See wasm.OpcodeI32Xor wasm.OpcodeI64Xor
	compileXor(o *wazeroir.OperationXor) error
	// compileShl emits instructions to perform a shift-left operation on
	// top two values on the stack, and pushes the result.
	// See wasm.OpcodeI32Shl wasm.OpcodeI64Shl
	compileShl(o *wazeroir.OperationShl) error
	// compileShr emits instructions to perform a shift-right operation on
	// top two values on the stack, and pushes the result.
	// See wasm.OpcodeI32Shr wasm.OpcodeI64Shr
	compileShr(o *wazeroir.OperationShr) error
	// compileRotl emits instructions to perform a rotate-left operation on
	// top two values on the stack, and pushes the result.
	// See wasm.OpcodeI32Rotl wasm.OpcodeI64Rotl
	compileRotl(o *wazeroir.OperationRotl) error
	// compileRotr emits instructions to perform a rotate-right operation on
	// top two values on the stack, and pushes the result.
	// See wasm.OpcodeI32Rotr wasm.OpcodeI64Rotr
	compileRotr(o *wazeroir.OperationRotr) error
	// compileAbs adds instructions to replace the top value of float type on the stack with its absolute value.
	// For example, stack [..., -1.123] results in [..., 1.123].
	// See wasm.OpcodeF32Abs wasm.OpcodeF64Abs
	compileAbs(o *wazeroir.OperationAbs) error
	// compileNeg adds instructions to replace the top value of float type on the stack with its negated value.
	// For example, stack [..., -1.123] results in [..., 1.123].
	// See wasm.OpcodeF32Neg wasm.OpcodeF64Neg
	compileNeg(o *wazeroir.OperationNeg) error
	// compileCeil adds instructions to replace the top value of float type on the stack with its ceiling value.
	// For example, stack [..., 1.123] results in [..., 2.0]. This is equivalent to math.Ceil.
	// See wasm.OpcodeF32Ceil wasm.OpcodeF64Ceil
	compileCeil(o *wazeroir.OperationCeil) error
	// compileFloor adds instructions to replace the top value of float type on the stack with its floor value.
	// For example, stack [..., 1.123] results in [..., 1.0]. This is equivalent to math.Floor.
	// See wasm.OpcodeF32Floor wasm.OpcodeF64Floor
	compileFloor(o *wazeroir.OperationFloor) error
	// compileTrunc adds instructions to replace the top value of float type on the stack with its truncated value.
	// For example, stack [..., 1.9] results in [..., 1.0]. This is equivalent to math.Trunc.
	// See wasm.OpcodeF32Trunc wasm.OpcodeF64Trunc
	compileTrunc(o *wazeroir.OperationTrunc) error
	// compileNearest adds instructions to replace the top value of float type on the stack with its nearest integer value.
	// For example, stack [..., 1.9] results in [..., 2.0]. This is *not* equivalent to math.Round and instead has the same
	// the semantics of LLVM's rint intrinsic. See https://llvm.org/docs/LangRef.html#llvm-rint-intrinsic.
	// For example, math.Round(-4.5) produces -5 while we want to produce -4.
	// See wasm.OpcodeF32Nearest wasm.OpcodeF64Nearest
	compileNearest(o *wazeroir.OperationNearest) error
	// compileSqrt adds instructions to replace the top value of float type on the stack with its square root.
	// For example, stack [..., 9.0] results in [..., 3.0]. This is equivalent to "math.Sqrt".
	// See wasm.OpcodeF32Sqrt wasm.OpcodeF64Sqrt
	compileSqrt(o *wazeroir.OperationSqrt) error
	// compileMin adds instructions to pop two values from the stack, and push back the maximum of
	// these two values onto the stack. For example, stack [..., 100.1, 1.9] results in [..., 1.9].
	// Note: WebAssembly specifies that min/max must always return NaN if one of values is NaN,
	// which is a different behavior different from math.Min.
	// See wasm.OpcodeF32Min wasm.OpcodeF64Min
	compileMin(o *wazeroir.OperationMin) error
	// compileMax adds instructions to pop two values from the stack, and push back the maximum of
	// these two values onto the stack. For example, stack [..., 100.1, 1.9] results in [..., 100.1].
	// Note: WebAssembly specifies that min/max must always return NaN if one of values is NaN,
	// which is a different behavior different from math.Max.
	// See wasm.OpcodeF32Max wasm.OpcodeF64Max
	compileMax(o *wazeroir.OperationMax) error
	// compileCopysign adds instructions to pop two float values from the stack, and copy the signbit of
	// the first-popped value to the last one.
	// For example, stack [..., 1.213, -5.0] results in [..., -1.213].
	// See wasm.OpcodeF32Copysign wasm.OpcodeF64Copysign
	compileCopysign(o *wazeroir.OperationCopysign) error
	// compileI32WrapFromI64 adds instructions to replace the 64-bit int on top of the stack
	// with the corresponding 32-bit integer. This is equivalent to uint64(uint32(v)) in Go.
	// See wasm.OpcodeI32WrapI64.
	compileI32WrapFromI64() error
	// compileITruncFromF adds instructions to replace the top value of float type on the stack with
	// the corresponding int value. This is equivalent to int32(math.Trunc(float32(x))), uint32(math.Trunc(float64(x))), etc in Go.
	//
	// Please refer to [1] and [2] for when we encounter undefined behavior in the WebAssembly specification.
	// To summarize, if the source float value is NaN or doesn't fit in the destination range of integers (incl. +=Inf),
	// then the runtime behavior is undefined. In wazero, we exit the function in these undefined cases with
	// nativeCallStatusCodeInvalidFloatToIntConversion or nativeCallStatusIntegerOverflow status code.
	// [1] https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#-hrefop-trunc-umathrmtruncmathsfu_m-n-z for unsigned integers.
	// [2] https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#-hrefop-trunc-smathrmtruncmathsfs_m-n-z for signed integers.
	// See OpcodeI32TruncF32S OpcodeI32TruncF32U OpcodeI32TruncF64S OpcodeI32TruncF64U
	// See OpcodeI64TruncF32S OpcodeI64TruncF32U OpcodeI64TruncF64S OpcodeI64TruncF64U
	compileITruncFromF(o *wazeroir.OperationITruncFromF) error
	// compileFConvertFromI adds instructions to replace the top value of int type on the stack with
	// the corresponding float value. This is equivalent to float32(uint32(x)), float32(int32(x)), etc in Go.
	// See OpcodeI32ConvertF32S OpcodeI32ConvertF32U OpcodeI32ConvertF64S OpcodeI32ConvertF64U
	// See OpcodeI64ConvertF32S OpcodeI64ConvertF32U OpcodeI64ConvertF64S OpcodeI64ConvertF64U
	compileFConvertFromI(o *wazeroir.OperationFConvertFromI) error
	// compileF32DemoteFromF64 adds instructions to replace the 64-bit float on top of the stack
	// with the corresponding 32-bit float. This is equivalent to float32(float64(v)) in Go.
	// See wasm.OpcodeF32DemoteF64
	compileF32DemoteFromF64() error
	// compileF64PromoteFromF32 adds instructions to replace the 32-bit float on top of the stack
	// with the corresponding 64-bit float. This is equivalent to float64(float32(v)) in Go.
	// See wasm.OpcodeF64PromoteF32
	compileF64PromoteFromF32() error
	// compileI32ReinterpretFromF32 adds instructions to reinterpret the 32-bit float on top of the stack
	// as a 32-bit integer by preserving the bit representation. If the value is on the stack,
	// this is no-op as there is nothing to do for converting type.
	// See wasm.OpcodeI32ReinterpretF32.
	compileI32ReinterpretFromF32() error
	// compileI64ReinterpretFromF64 adds instructions to reinterpret the 64-bit float on top of the stack
	// as a 64-bit integer by preserving the bit representation.
	// See wasm.OpcodeI64ReinterpretF64.
	compileI64ReinterpretFromF64() error
	// compileF32ReinterpretFromI32 adds instructions to reinterpret the 32-bit int on top of the stack
	// as a 32-bit float by preserving the bit representation.
	// See wasm.OpcodeF32ReinterpretI32.
	compileF32ReinterpretFromI32() error
	// compileF64ReinterpretFromI64 adds instructions to reinterpret the 64-bit int on top of the stack
	// as a 64-bit float by preserving the bit representation.
	// See wasm.OpcodeF64ReinterpretI64.
	compileF64ReinterpretFromI64() error
	// compileExtend adds instructions to extend the 32-bit signed or unsigned int on top of the stack
	// as a 64-bit integer of corresponding signedness. For unsigned case, this is just reinterpreting the
	// underlying bit pattern as 64-bit integer. For signed case, this is sign-extension which preserves the
	// original integer's sign.
	// See wasm.OpcodeI64ExtendI32S wasm.OpcodeI64ExtendI32U
	compileExtend(o *wazeroir.OperationExtend) error
	// compileEq adds instructions to pop two values from the stack and push 1 if they equal otherwise 0.
	// See wasm.OpcodeI32Eq wasm.OpcodeI64Eq
	compileEq(o *wazeroir.OperationEq) error
	// compileEq adds instructions to pop two values from the stack and push 0 if they equal otherwise 1.
	// See wasm.OpcodeI32Ne wasm.OpcodeI64Ne
	compileNe(o *wazeroir.OperationNe) error
	// compileEq adds instructions to pop a value from the stack and push 1 if it equals zero, 0.
	// See wasm.OpcodeI32Eqz wasm.OpcodeI64Eqz
	compileEqz(o *wazeroir.OperationEqz) error
	// compileLt adds instructions to pop two values from the stack and push 1 if the second is less than the top one. Otherwise 0.
	// See wasm.OpcodeI32Lt wasm.OpcodeI64Lt
	compileLt(o *wazeroir.OperationLt) error
	// compileGt adds instructions to pop two values from the stack and push 1 if the second is greater than the top one. Otherwise 0.
	// See wasm.OpcodeI32Gt wasm.OpcodeI64Gt
	compileGt(o *wazeroir.OperationGt) error
	// compileLe adds instructions to pop two values from the stack and push 1 if the second is less than or equals the top one. Otherwise 0.
	// See wasm.OpcodeI32Le wasm.OpcodeI64Le
	compileLe(o *wazeroir.OperationLe) error
	// compileLe adds instructions to pop two values from the stack and push 1 if the second is greater than or equals the top one. Otherwise 0.
	// See wasm.OpcodeI32Ge wasm.OpcodeI64Ge
	compileGe(o *wazeroir.OperationGe) error
	// compileLoad adds instructions to perform load instruction in WebAssembly.
	// See wasm.OpcodeI32Load wasm.OpcodeI64Load wasm.OpcodeF32Load wasm.OpcodeF64Load
	compileLoad(o *wazeroir.OperationLoad) error
	// compileLoad8 adds instructions to perform load8 instruction in WebAssembly.
	// The resulting code checks the memory boundary at runtime, and exit the function with nativeCallStatusCodeMemoryOutOfBounds if out-of-bounds access happens.
	// See wasm.OpcodeI32Load8S wasm.OpcodeI32Load8U wasm.OpcodeI64Load8S wasm.OpcodeI64Load8U
	compileLoad8(o *wazeroir.OperationLoad8) error
	// compileLoad16 adds instructions to perform load16 instruction in WebAssembly.
	// The resulting code checks the memory boundary at runtime, and exit the function with nativeCallStatusCodeMemoryOutOfBounds if out-of-bounds access happens.
	// See wasm.OpcodeI32Load16S wasm.OpcodeI32Load16U wasm.OpcodeI64Load16S wasm.OpcodeI64Load16U
	compileLoad16(o *wazeroir.OperationLoad16) error
	// compileLoad32 adds instructions to perform load32 instruction in WebAssembly.
	// The resulting code checks the memory boundary at runtime, and exit the function with nativeCallStatusCodeMemoryOutOfBounds
	// if out-of-bounds access happens.
	// See wasm.OpcodeI64Load32S wasm.OpcodeI64Load32U
	compileLoad32(o *wazeroir.OperationLoad32) error
	// compileStore adds instructions to perform store instruction in WebAssembly.
	// The resulting code checks the memory boundary at runtime, and exit the function with nativeCallStatusCodeMemoryOutOfBounds
	// if out-of-bounds access happens.
	// See wasm.OpcodeI32Store wasm.OpcodeI64Store wasm.OpcodeF32Store wasm.OpcodeF64Store
	compileStore(o *wazeroir.OperationStore) error
	// compileStore8 adds instructions to perform store8 instruction in WebAssembly.
	// The resulting code checks the memory boundary at runtime, and exit the function with nativeCallStatusCodeMemoryOutOfBounds
	// if out-of-bounds access happens.
	// See wasm.OpcodeI32Store8S wasm.OpcodeI32Store8U wasm.OpcodeI64Store8S wasm.OpcodeI64Store8U
	compileStore8(o *wazeroir.OperationStore8) error
	// compileStore16 adds instructions to perform store16 instruction in WebAssembly.
	// The resulting code checks the memory boundary at runtime, and exit the function with nativeCallStatusCodeMemoryOutOfBounds
	// if out-of-bounds access happens.
	// See wasm.OpcodeI32Store16S wasm.OpcodeI32Store16U wasm.OpcodeI64Store16S wasm.OpcodeI64Store16U
	compileStore16(o *wazeroir.OperationStore16) error
	// compileStore32 adds instructions to perform store32 instruction in WebAssembly.
	// The resulting code checks the memory boundary at runtime, and exit the function with nativeCallStatusCodeMemoryOutOfBounds
	// if out-of-bounds access happens.
	// See wasm.OpcodeI64Store32S wasm.OpcodeI64Store32U
	compileStore32(o *wazeroir.OperationStore32) error
	// compileMemorySize adds instruction to pop a value from the stack, grow the memory buffer according to the value,
	// and push the previous page size onto the stack.
	// See wasm.OpcodeMemoryGrow
	compileMemoryGrow() error
	// compileMemorySize adds instruction to read the current page size of memory instance and push it onto the stack.
	// See wasm.OpcodeMemorySize
	compileMemorySize() error
	// compileConstI32 adds instruction to push the given constant i32 value onto the stack.
	// See wasm.OpcodeI32Const
	compileConstI32(o *wazeroir.OperationConstI32) error
	// compileConstI32 adds instruction to push the given constant i64 value onto the stack.
	// See wasm.OpcodeI64Const
	compileConstI64(o *wazeroir.OperationConstI64) error
	// compileConstI32 adds instruction to push the given constant f32 value onto the stack.
	// See wasm.OpcodeF32Const
	compileConstF32(o *wazeroir.OperationConstF32) error
	// compileConstI32 adds instruction to push the given constant f64 value onto the stack.
	// See wasm.OpcodeF64Const
	compileConstF64(o *wazeroir.OperationConstF64) error
	// compileSignExtend32From8 adds instruction to sign-extends the first 8-bits of 32-bit in as signed 32-bit int.
	// See wasm.OpcodeI32Extend8S
	compileSignExtend32From8() error
	// compileSignExtend32From16 adds instruction to sign-extends the first 16-bits of 32-bit in as signed 32-bit int.
	// See wasm.OpcodeI32Extend16S
	compileSignExtend32From16() error
	// compileSignExtend64From8 adds instruction to sign-extends the first 8-bits of 64-bit in as signed 64-bit int.
	// See wasm.OpcodeI64Extend8S
	compileSignExtend64From8() error
	// compileSignExtend64From16 adds instruction to sign-extends the first 16-bits of 64-bit in as signed 64-bit int.
	// See wasm.OpcodeI64Extend16S
	compileSignExtend64From16() error
	// compileSignExtend64From32 adds instruction to sign-extends the first 32-bits of 64-bit in as signed 64-bit int.
	// See wasm.OpcodeI64Extend32S
	compileSignExtend64From32() error
	// compileMemoryInit adds instructions to perform operations corresponding to the wasm.OpcodeMemoryInitName instruction in
	// wasm.FeatureBulkMemoryOperations.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/appendix/changes.html#bulk-memory-and-table-instructions
	compileMemoryInit(*wazeroir.OperationMemoryInit) error
	// compileDataDrop adds instructions to perform operations corresponding to the wasm.OpcodeDataDropName instruction in
	// wasm.FeatureBulkMemoryOperations.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/appendix/changes.html#bulk-memory-and-table-instructions
	compileDataDrop(*wazeroir.OperationDataDrop) error
	// compileMemoryCopy adds instructions to perform operations corresponding to the wasm.OpcodeMemoryCopylName instruction in
	// wasm.FeatureBulkMemoryOperations.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/appendix/changes.html#bulk-memory-and-table-instructions
	compileMemoryCopy() error
	// compileMemoryCopy adds instructions to perform operations corresponding to the wasm.OpcodeMemoryFillName instruction in
	// wasm.FeatureBulkMemoryOperations.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/appendix/changes.html#bulk-memory-and-table-instructions
	compileMemoryFill() error
	// compileTableInit adds instructions to perform operations corresponding to the wasm.OpcodeTableInit instruction in
	// wasm.FeatureBulkMemoryOperations.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/appendix/changes.html#bulk-memory-and-table-instructions
	compileTableInit(*wazeroir.OperationTableInit) error
	// compileTableCopy adds instructions to perform operations corresponding to the wasm.OpcodeTableCopy instruction in
	// wasm.FeatureBulkMemoryOperations.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/appendix/changes.html#bulk-memory-and-table-instructions
	compileTableCopy(*wazeroir.OperationTableCopy) error
	// compileElemDrop adds instructions to perform operations corresponding to the wasm.OpcodeElemDrop instruction in
	// wasm.FeatureBulkMemoryOperations.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/appendix/changes.html#bulk-memory-and-table-instructions
	compileElemDrop(*wazeroir.OperationElemDrop) error
	// compileRefFunc adds instructions to perform operations corresponding to wasm.OpcodeRefFunc instruction introduced in
	// wasm.FeatureReferenceTypes.
	//
	// Note: in wazero, we express any reference types (funcref or externref) as opaque pointers which is uint64.
	// Therefore, the compilers implementations emit instructions to push the address of *function onto the stack.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/valid/instructions.html#xref-syntax-instructions-syntax-instr-ref-mathsf-ref-func-x
	compileRefFunc(*wazeroir.OperationRefFunc) error
	// compileTableGet adds instructions to perform operations corresponding to wasm.OpcodeTableGet instruction introduced in
	// wasm.FeatureReferenceTypes.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/valid/instructions.html#xref-syntax-instructions-syntax-instr-table-mathsf-table-get-x
	compileTableGet(*wazeroir.OperationTableGet) error
	// compileTableSet adds instructions to perform operations corresponding to wasm.OpcodeTableSet instruction introduced in
	// wasm.FeatureReferenceTypes.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/valid/instructions.html#xref-syntax-instructions-syntax-instr-table-mathsf-table-set-x
	compileTableSet(*wazeroir.OperationTableSet) error
	// compileTableGrow adds instructions to perform operations corresponding to wasm.OpcodeMiscTableGrow instruction introduced in
	// wasm.FeatureReferenceTypes.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/valid/instructions.html#xref-syntax-instructions-syntax-instr-table-mathsf-table-grow-x
	compileTableGrow(*wazeroir.OperationTableGrow) error
	// compileTableSize adds instructions to perform operations corresponding to wasm.OpcodeMiscTableSize instruction introduced in
	// wasm.FeatureReferenceTypes.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/valid/instructions.html#xref-syntax-instructions-syntax-instr-table-mathsf-table-size-x
	compileTableSize(*wazeroir.OperationTableSize) error
	// compileTableFill adds instructions to perform operations corresponding to wasm.OpcodeMiscTableFill instruction introduced in
	// wasm.FeatureReferenceTypes.
	//
	// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/valid/instructions.html#xref-syntax-instructions-syntax-instr-table-mathsf-table-fill-x
	compileTableFill(*wazeroir.OperationTableFill) error
	// compileV128Const adds instructions to push a constant V128 value onto the stack.
	// See wasm.OpcodeVecV128Const
	compileV128Const(*wazeroir.OperationV128Const) error
	// compileV128Add adds instruction to add two vector values whose shape is specified as `o.Shape`.
	// See wasm.OpcodeVecI8x16Add wasm.OpcodeVecI16x8Add wasm.OpcodeVecI32x4Add wasm.OpcodeVecI64x2Add wasm.OpcodeVecF32x4Add wasm.OpcodeVecF64x2Add
	compileV128Add(o *wazeroir.OperationV128Add) error
	// compileV128Sub adds instruction to subtract two vector values whose shape is specified as `o.Shape`.
	// See wasm.OpcodeVecI8x16Sub wasm.OpcodeVecI16x8Sub wasm.OpcodeVecI32x4Sub wasm.OpcodeVecI64x2Sub wasm.OpcodeVecF32x4Sub wasm.OpcodeVecF64x2Sub
	compileV128Sub(o *wazeroir.OperationV128Sub) error
	// compileV128Load adds instruction to perform vector load kind instructions.
	// See wasm.OpcodeVecV128Load* instructions.
	compileV128Load(o *wazeroir.OperationV128Load) error
	// compileV128LoadLane adds instructions which are equivalent to wasm.OpcodeVecV128LoadXXLane instructions.
	// See wasm.OpcodeVecV128Load8LaneName wasm.OpcodeVecV128Load16LaneName wasm.OpcodeVecV128Load32LaneName wasm.OpcodeVecV128Load64LaneName
	compileV128LoadLane(o *wazeroir.OperationV128LoadLane) error
	// compileV128Store adds instructions which are equivalent to wasm.OpcodeVecV128StoreName.
	compileV128Store(o *wazeroir.OperationV128Store) error
	// compileV128StoreLane adds instructions which are equivalent to wasm.OpcodeVecV128StoreXXLane instructions.
	// See wasm.OpcodeVecV128Load8LaneName wasm.OpcodeVecV128Load16LaneName wasm.OpcodeVecV128Load32LaneName wasm.OpcodeVecV128Load64LaneName.
	compileV128StoreLane(o *wazeroir.OperationV128StoreLane) error
	// compileV128ExtractLane adds instructions which are equivalent to wasm.OpcodeVecXXXXExtractLane instructions.
	// See wasm.OpcodeVecI8x16ExtractLaneSName wasm.OpcodeVecI8x16ExtractLaneUName wasm.OpcodeVecI16x8ExtractLaneSName wasm.OpcodeVecI16x8ExtractLaneUName
	// wasm.OpcodeVecI32x4ExtractLaneName wasm.OpcodeVecI64x2ExtractLaneName wasm.OpcodeVecF32x4ExtractLaneName wasm.OpcodeVecF64x2ExtractLaneName.
	compileV128ExtractLane(o *wazeroir.OperationV128ExtractLane) error
	// compileV128ReplaceLane adds instructions which are equivalent to wasm.OpcodeVecXXXXReplaceLane instructions.
	// See wasm.OpcodeVecI8x16ReplaceLaneName wasm.OpcodeVecI16x8ReplaceLaneName wasm.OpcodeVecI32x4ReplaceLaneName wasm.OpcodeVecI64x2ReplaceLaneName
	// wasm.OpcodeVecF32x4ReplaceLaneName wasm.OpcodeVecF64x2ReplaceLaneName.
	compileV128ReplaceLane(o *wazeroir.OperationV128ReplaceLane) error
	// compileV128Splat adds instructions which are equivalent to wasm.OpcodeVecXXXSplat instructions.
	// See wasm.OpcodeVecI8x16SplatName wasm.OpcodeVecI16x8SplatName wasm.OpcodeVecI32x4SplatName wasm.OpcodeVecI64x2SplatName
	// wasm.OpcodeVecF32x4SplatName wasm.OpcodeVecF64x2SplatName.
	compileV128Splat(o *wazeroir.OperationV128Splat) error
	// compileV128Shuffle adds instructions which are equivalent to wasm.OpcodeVecV128i8x16ShuffleName instruction.
	compileV128Shuffle(o *wazeroir.OperationV128Shuffle) error
	// compileV128Swizzle adds instructions which are equivalent to wasm.OpcodeVecI8x16SwizzleName instruction.
	compileV128Swizzle(o *wazeroir.OperationV128Swizzle) error
	// compileV128Swizzle adds instructions which are equivalent to wasm.OpcodeVecV128AnyTrueName instruction.
	compileV128AnyTrue(o *wazeroir.OperationV128AnyTrue) error
	// compileV128AllTrue adds instructions which are equivalent to wasm.OpcodeVecXXXAllTrue instructions.
	// See wasm.OpcodeVecI8x16AllTrueName wasm.OpcodeVecI16x8AllTrueName wasm.OpcodeVecI32x4AllTrueName wasm.OpcodeVecI64x2AllTrueName.
	compileV128AllTrue(o *wazeroir.OperationV128AllTrue) error
	// compileV128BitMask adds instructions which are equivalent to wasm.OpcodeVecV128XXXBitMask instruction.
	// See wasm.OpcodeVecI8x16BitMaskName wasm.OpcodeVecI16x8BitMaskName wasm.OpcodeVecI32x4BitMaskName wasm.OpcodeVecI64x2BitMaskName.
	compileV128BitMask(*wazeroir.OperationV128BitMask) error
	// compileV128And adds instructions which are equivalent to wasm.OpcodeVecV128AndName instruction.
	// See wasm.OpcodeVecV128AndName.
	compileV128And(*wazeroir.OperationV128And) error
	// compileV128Not adds instructions which are equivalent to wasm.OpcodeVecV128NotName instruction.
	// See wasm.OpcodeVecV128NotName.
	compileV128Not(*wazeroir.OperationV128Not) error
	// compileV128Or adds instructions which are equivalent to wasm.OpcodeVecV128OrName instruction.
	// See wasm.OpcodeVecV128OrName.
	compileV128Or(*wazeroir.OperationV128Or) error
	// compileV128Xor adds instructions which are equivalent to wasm.OpcodeVecV128XorName instruction.
	// See wasm.OpcodeVecV128XorName.
	compileV128Xor(*wazeroir.OperationV128Xor) error
	// compileV128Bitselect adds instructions which are equivalent to wasm.OpcodeVecV128BitselectName instruction.
	// See wasm.OpcodeVecV128BitselectName.
	compileV128Bitselect(*wazeroir.OperationV128Bitselect) error
	// compileV128AndNot adds instructions which are equivalent to wasm.OpcodeVecV128AndNotName instruction.
	// See wasm.OpcodeVecV128AndNotName.
	compileV128AndNot(*wazeroir.OperationV128AndNot) error
	// compileV128Shr adds instructions which are equivalent to wasm.OpcodeVecXXXShrYYYY instructions.
	// See wasm.OpcodeVecI8x16ShrSName wasm.OpcodeVecI8x16ShrUName wasm.OpcodeVecI16x8ShrSName
	// wasm.OpcodeVecI16x8ShrUName wasm.OpcodeVecI32x4ShrSName wasm.OpcodeVecI32x4ShrUName.
	// wasm.OpcodeVecI64x2ShrSName wasm.OpcodeVecI64x2ShrUName.
	compileV128Shr(*wazeroir.OperationV128Shr) error
	// compileV128Shl adds instructions which are equivalent to wasm.OpcodeVecXXXShl instructions.
	// See wasm.OpcodeVecI8x16ShlName wasm.OpcodeVecI16x8ShlName wasm.OpcodeVecI32x4ShlName wasm.OpcodeVecI64x2ShlName
	compileV128Shl(*wazeroir.OperationV128Shl) error
	// compileV128Cmp adds instructions which are equivalent to various vector comparison instructions.
	// See wasm.OpcodeVecI8x16EqName, wasm.OpcodeVecI8x16NeName, wasm.OpcodeVecI8x16LtSName, wasm.OpcodeVecI8x16LtUName, wasm.OpcodeVecI8x16GtSName,
	//	wasm.OpcodeVecI8x16GtUName, wasm.OpcodeVecI8x16LeSName, wasm.OpcodeVecI8x16LeUName, wasm.OpcodeVecI8x16GeSName, wasm.OpcodeVecI8x16GeUName,
	//	wasm.OpcodeVecI16x8EqName, wasm.OpcodeVecI16x8NeName, wasm.OpcodeVecI16x8LtSName, wasm.OpcodeVecI16x8LtUName, wasm.OpcodeVecI16x8GtSName,
	//	wasm.OpcodeVecI16x8GtUName, wasm.OpcodeVecI16x8LeSName, wasm.OpcodeVecI16x8LeUName, wasm.OpcodeVecI16x8GeSName, wasm.OpcodeVecI16x8GeUName,
	//	wasm.OpcodeVecI32x4EqName, wasm.OpcodeVecI32x4NeName, wasm.OpcodeVecI32x4LtSName, wasm.OpcodeVecI32x4LtUName, wasm.OpcodeVecI32x4GtSName,
	//	wasm.OpcodeVecI32x4GtUName, wasm.OpcodeVecI32x4LeSName, wasm.OpcodeVecI32x4LeUName, wasm.OpcodeVecI32x4GeSName, wasm.OpcodeVecI32x4GeUName,
	//	wasm.OpcodeVecI64x2EqName, wasm.OpcodeVecI64x2NeName, wasm.OpcodeVecI64x2LtSName, wasm.OpcodeVecI64x2GtSName, wasm.OpcodeVecI64x2LeSName,
	//	wasm.OpcodeVecI64x2GeSName, wasm.OpcodeVecF32x4EqName, wasm.OpcodeVecF32x4NeName, wasm.OpcodeVecF32x4LtName, wasm.OpcodeVecF32x4GtName,
	//	wasm.OpcodeVecF32x4LeName, wasm.OpcodeVecF32x4GeName, wasm.OpcodeVecF64x2EqName, wasm.OpcodeVecF64x2NeName, wasm.OpcodeVecF64x2LtName,
	//	wasm.OpcodeVecF64x2GtName, wasm.OpcodeVecF64x2LeName, wasm.OpcodeVecF64x2GeName
	compileV128Cmp(*wazeroir.OperationV128Cmp) error
	// compileV128AddSat adds instructions which are equivalent to wasm.OpcodeVecXXXAddSatY.
	// See wasm.OpcodeVecI8x16AddSatUName wasm.OpcodeVecI8x16AddSatSName wasm.OpcodeVecI16x8AddSatUName wasm.OpcodeVecI16x8AddSatSName
	compileV128AddSat(*wazeroir.OperationV128AddSat) error
	// compileV128SubSat adds instructions which are equivalent to wasm.OpcodeVecXXXSubSatY.
	// See wasm.OpcodeVecI8x16SubSatUName wasm.OpcodeVecI8x16SubSatSName wasm.OpcodeVecI16x8SubSatUName wasm.OpcodeVecI16x8SubSatSName
	compileV128SubSat(*wazeroir.OperationV128SubSat) error
	// compileV128Mul adds instructions which are equivalent to wasm.OpcodeVecXXXMul.
	// See wasm.OpcodeVecF32x4MulName wasm.OpcodeVecF64x2MulName wasm.OpcodeVecI16x8MulName wasm.OpcodeVecI32x4MulName wasm.OpcodeVecI64x2MulName.
	compileV128Mul(*wazeroir.OperationV128Mul) error
	// compileV128Div adds instructions which are equivalent to wasm.OpcodeVecXXXDiv.
	// See wasm.OpcodeVecF32x4DivName wasm.OpcodeVecF64x2DivName.
	compileV128Div(*wazeroir.OperationV128Div) error
	// compileV128Neg adds instructions which are equivalent to wasm.OpcodeVecXXXXNeg instructions.
	// See wasm.OpcodeVecI8x16NegName wasm.OpcodeVecI16x8NegName wasm.OpcodeVecI32x4NegName
	// 	wasm.OpcodeVecI64x2NegName wasm.OpcodeVecF32x4NegName wasm.OpcodeVecF64x2NegName.
	compileV128Neg(*wazeroir.OperationV128Neg) error
	// compileV128Sqrt adds instructions which are equivalent to wasm.OpcodeVecXXXXSqrt instructions.
	// See wasm.OpcodeVecF32x4SqrtName wasm.OpcodeVecF64x2SqrtName.
	compileV128Sqrt(*wazeroir.OperationV128Sqrt) error
	// compileV128Abs adds instructions which are equivalent to wasm.OpcodeVecXXXXAbs instructions.
	// See wasm.OpcodeVecI8x16AbsName wasm.OpcodeVecI16x8AbsName wasm.OpcodeVecI32x4AbsName
	// 	wasm.OpcodeVecI64x2AbsName wasm.OpcodeVecF32x4AbsName wasm.OpcodeVecF64x2AbsName.
	compileV128Abs(*wazeroir.OperationV128Abs) error
	// compileV128Popcnt adds instructions which are equivalent to wasm.OpcodeVecI8x16PopcntName.
	compileV128Popcnt(*wazeroir.OperationV128Popcnt) error
	// compileV128Min adds instructions which are equivalent to wasm.OpcodeVecXXXXMinY instructions.
	// See wasm.OpcodeVecI8x16MinSName wasm.OpcodeVecI8x16MinUName　wasm.OpcodeVecI16x8MinSName wasm.OpcodeVecI16x8MinUName
	//	wasm.OpcodeVecI32x4MinSName wasm.OpcodeVecI32x4MinUName　wasm.OpcodeVecI16x8MinSName wasm.OpcodeVecI16x8MinUName
	//	wasm.OpcodeVecF32x4MinName wasm.OpcodeVecF64x2MinName
	compileV128Min(*wazeroir.OperationV128Min) error
	// compileV128Max adds instructions which are equivalent to wasm.OpcodeVecXXXXMaxY instructions.
	// See wasm.OpcodeVecI8x16MaxSName wasm.OpcodeVecI8x16MaxUName　wasm.OpcodeVecI16x8MaxSName wasm.OpcodeVecI16x8MaxUName
	//	wasm.OpcodeVecI32x4MaxSName wasm.OpcodeVecI32x4MaxUName　wasm.OpcodeVecI16x8MaxSName wasm.OpcodeVecI16x8MaxUName
	//	wasm.OpcodeVecF32x4MaxName wasm.OpcodeVecF64x2MaxName
	compileV128Max(*wazeroir.OperationV128Max) error
	// compileV128AvgrU adds instructions which are equivalent to wasm.OpcodeVecI8x16AvgrUName.
	compileV128AvgrU(*wazeroir.OperationV128AvgrU) error
	// compileV128Pmin adds instructions which are equivalent to wasm.OpcodeVecXXXPmin.
	// See wasm.OpcodeVecF32x4PminName wasm.OpcodeVecF64x2PminName
	compileV128Pmin(*wazeroir.OperationV128Pmin) error
	// compileV128Pmax adds instructions which are equivalent to wasm.OpcodeVecXXXPmax.
	// See wasm.OpcodeVecF32x4PmaxName wasm.OpcodeVecF64x2PmaxName
	compileV128Pmax(*wazeroir.OperationV128Pmax) error
	// compileV128Ceil adds instructions which are equivalent to wasm.OpcodeVecXXXCeil.
	// See wasm.OpcodeVecF32x4CeilName wasm.OpcodeVecF64x2CeilName
	compileV128Ceil(*wazeroir.OperationV128Ceil) error
	// compileV128Floor adds instructions which are equivalent to wasm.OpcodeVecXXXFloor.
	// See wasm.OpcodeVecF32x4FloorName wasm.OpcodeVecF64x2Floor
	compileV128Floor(*wazeroir.OperationV128Floor) error
	// compileV128Trunc adds instructions which are equivalent to wasm.OpcodeVecXXXTrunc.
	// See wasm.OpcodeVecF32x4TruncName wasm.OpcodeVecF64x2TruncName
	compileV128Trunc(*wazeroir.OperationV128Trunc) error
	// compileV128Nearest adds instructions which are equivalent to wasm.OpcodeVecXXXNearest.
	// See wasm.OpcodeVecF32x4NearestName wasm.OpcodeVecF64x2NearestName
	compileV128Nearest(*wazeroir.OperationV128Nearest) error
	// compileV128Extend adds instructions which are equivalent to wasm.OpcodeVec
	// See wasm.OpcodeVecI16x8ExtendLowI8x16SName wasm.OpcodeVecI16x8ExtendHighI8x16SName
	// 	wasm.OpcodeVecI16x8ExtendLowI8x16UName wasm.OpcodeVecI16x8ExtendHighI8x16UName
	// 	wasm.OpcodeVecI32x4ExtendLowI16x8SName wasm.OpcodeVecI32x4ExtendHighI16x8SName
	// 	wasm.OpcodeVecI32x4ExtendLowI16x8UName wasm.OpcodeVecI32x4ExtendHighI16x8UName
	// 	wasm.OpcodeVecI64x2ExtendLowI32x4SName wasm.OpcodeVecI64x2ExtendHighI32x4SName
	// 	wasm.OpcodeVecI64x2ExtendLowI32x4UName wasm.OpcodeVecI64x2ExtendHighI32x4UName
	compileV128Extend(*wazeroir.OperationV128Extend) error
	// compileV128ExtMul adds instructions which are equivalent to wasm.OpcodeVecXXXXExtMulYYY.
	// See wasm.OpcodeVecI16x8ExtMulLowI8x16SName wasm.OpcodeVecI16x8ExtMulLowI8x16UName
	// 	wasm.OpcodeVecI16x8ExtMulHighI8x16SName wasm.OpcodeVecI16x8ExtMulHighI8x16UName
	//  wasm.OpcodeVecI32x4ExtMulLowI16x8SName wasm.OpcodeVecI32x4ExtMulLowI16x8UName
	// 	wasm.OpcodeVecI32x4ExtMulHighI16x8SName wasm.OpcodeVecI32x4ExtMulHighI16x8UName
	//  wasm.OpcodeVecI64x2ExtMulLowI32x4SName wasm.OpcodeVecI64x2ExtMulLowI32x4UName
	// 	wasm.OpcodeVecI64x2ExtMulHighI32x4SName wasm.OpcodeVecI64x2ExtMulHighI32x4UName.
	compileV128ExtMul(*wazeroir.OperationV128ExtMul) error
	// compileV128Q15mulrSatS adds instructions which are equivalent to wasm.OpcodeVecI16x8Q15mulrSatSName.
	compileV128Q15mulrSatS(*wazeroir.OperationV128Q15mulrSatS) error
	// compileV128ExtAddPairwise adds instructions which are equivalent to wasm.OpcodeVecXXXXExtaddPairwiseYYYY.
	// See wasm.OpcodeVecI16x8ExtaddPairwiseI8x16SName wasm.OpcodeVecI16x8ExtaddPairwiseI8x16UName
	// 	wasm.OpcodeVecI32x4ExtaddPairwiseI16x8SName wasm.OpcodeVecI32x4ExtaddPairwiseI16x8UName.
	compileV128ExtAddPairwise(o *wazeroir.OperationV128ExtAddPairwise) error
	// compileV128FloatPromote adds instructions which are equivalent to wasm.OpcodeVecF64x2PromoteLowF32x4ZeroName.
	compileV128FloatPromote(o *wazeroir.OperationV128FloatPromote) error
	// compileV128FloatDemote adds instructions which are equivalent to wasm.OpcodeVecF32x4DemoteF64x2ZeroName.
	compileV128FloatDemote(o *wazeroir.OperationV128FloatDemote) error
	// compileV128FConvertFromI adds instructions which are equivalent to wasm.OpcodeVecXXXXConvertYYYY.
	// See wasm.OpcodeVecF32x4ConvertI32x4SName wasm.OpcodeVecF32x4ConvertI32x4UName
	// 	wasm.OpcodeVecF64x2ConvertLowI32x4SName wasm.OpcodeVecF64x2ConvertLowI32x4UName.
	compileV128FConvertFromI(o *wazeroir.OperationV128FConvertFromI) error
	// compileV128Dot adds instructions which are equivalent to wasm.OpcodeVecI32x4DotI16x8SName.
	compileV128Dot(o *wazeroir.OperationV128Dot) error
	// compileV128Narrow adds instructions which are equivalent to wasm.OpcodeVecXXXNarrowYYY.
	// See wasm.OpcodeVecI8x16NarrowI16x8SName wasm.OpcodeVecI8x16NarrowI16x8UName
	// 	wasm.OpcodeVecI16x8NarrowI32x4SName wasm.OpcodeVecI16x8NarrowI32x4UName.
	compileV128Narrow(o *wazeroir.OperationV128Narrow) error
	// compileV128ITruncSatFromF adds instructions which are equivalent to wasm.OpcodeVecXXXTruncSatYYYName
	// See wasm.OpcodeVecI32x4TruncSatF64x2UZeroName wasm.OpcodeVecI32x4TruncSatF64x2SZeroName
	// 	wasm.OpcodeVecI32x4TruncSatF32x4UName wasm.OpcodeVecI32x4TruncSatF32x4SName.
	compileV128ITruncSatFromF(o *wazeroir.OperationV128ITruncSatFromF) error
}
