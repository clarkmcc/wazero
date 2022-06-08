package spectest

import (
	"embed"
	"path"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/internal/engine/compiler"
	"github.com/tetratelabs/wazero/internal/engine/interpreter"
	"github.com/tetratelabs/wazero/internal/integration_test/spectest"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/internal/wasm"
)

//go:embed testdata/*.wasm
//go:embed testdata/*.json
var testcases embed.FS

const enabledFeatures = wasm.Features20220419

func TestCompiler(t *testing.T) {
	if !platform.CompilerSupported() {
		t.Skip()
	}

	spectest.Run(t, testcases, compiler.NewEngine, enabledFeatures, func(jsonname string) bool {
		// TODO: remove after SIMD proposal
		if strings.Contains(jsonname, "simd") {
			switch path.Base(jsonname) {
			case "simd_address.json", "simd_const.json", "simd_align.json", "simd_load16_lane.json", "simd_load32_lane.json",
				"simd_load64_lane.json", "simd_load8_lane.json", "simd_lane.json", "simd_load_extend.json",
				"simd_load_splat.json", "simd_load_zero.json", "simd_store.json", "simd_store16_lane.json",
				"simd_store32_lane.json", "simd_store64_lane.json", "simd_store8_lane.json":
				return false
			case "simd_bitwise.json", "simd_boolean.json", "simd_bit_shift.json",
				"simd_i8x16_cmp.json", "simd_i16x8_cmp.json", "simd_i32x4_cmp.json", "simd_i64x2_cmp.json",
				"simd_f32x4_cmp.json", "simd_f64x2_cmp.json", "simd_f32x4_arith.json", "simd_f64x2_arith.json", "simd_i16x8_arith.json", "simd_i64x2_arith.json",
				"simd_i32x4_arith.json", "simd_i8x16_arith.json", "simd_i16x8_sat_arith.json", "simd_i8x16_sat_arith.json",
				"simd_i16x8_arith2.json", "simd_i8x16_arith2.json", "simd_i32x4_arith2.json", "simd_i64x2_arith2.json",
				"simd_f64x2.json", "simd_f32x4.json", "simd_f32x4_rounding.json", "simd_f64x2_rounding.json",
				"simd_f64x2_pmin_pmax.json", "simd_f32x4_pmin_pmax.json":
				// TODO: implement on arm64.
				return false
			case "simd_int_to_int_extend.json":
				return true
			default:
				return false // others not supported, yet!
			}
		}
		return false
	})
}

func TestInterpreter(t *testing.T) {
	spectest.Run(t, testcases, interpreter.NewEngine, enabledFeatures, func(jsonname string) bool {
		// TODO: remove after SIMD proposal
		if strings.Contains(jsonname, "simd") {
			switch path.Base(jsonname) {
			case "simd_address.json", "simd_const.json", "simd_align.json", "simd_load16_lane.json",
				"simd_load32_lane.json", "simd_load64_lane.json", "simd_load8_lane.json", "simd_lane.json",
				"simd_load_extend.json", "simd_load_splat.json", "simd_load_zero.json", "simd_store.json",
				"simd_store16_lane.json", "simd_store32_lane.json", "simd_store64_lane.json", "simd_store8_lane.json",
				"simd_bitwise.json", "simd_boolean.json", "simd_bit_shift.json", "simd_i8x16_cmp.json", "simd_i16x8_cmp.json",
				"simd_i32x4_cmp.json", "simd_i64x2_cmp.json", "simd_f32x4_cmp.json", "simd_f64x2_cmp.json",
				"simd_f32x4_arith.json", "simd_f64x2_arith.json", "simd_i16x8_arith.json", "simd_i64x2_arith.json",
				"simd_i32x4_arith.json", "simd_i8x16_arith.json", "simd_i16x8_sat_arith.json", "simd_i8x16_sat_arith.json",
				"simd_i16x8_arith2.json", "simd_i8x16_arith2.json", "simd_i32x4_arith2.json", "simd_i64x2_arith2.json",
				"simd_f64x2.json", "simd_f32x4.json", "simd_f32x4_rounding.json", "simd_f64x2_rounding.json",
				"simd_f64x2_pmin_pmax.json", "simd_f32x4_pmin_pmax.json", "simd_i16x8_q15mulr_sat_s.json",
				"simd_int_to_int_extend.json", "simd_i64x2_extmul_i32x4.json", "simd_i32x4_extmul_i16x8.json",
				"simd_i16x8_extmul_i8x16.json":
				return true
			default:
				return false // others not supported, yet!
			}
		}
		return true
	})
}
