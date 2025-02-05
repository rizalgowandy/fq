package macho

// https://github.com/aidansteele/osx-abi-macho-file-format-reference

import (
	"time"

	"github.com/wader/fq/format"
	"github.com/wader/fq/format/registry"
	"github.com/wader/fq/pkg/bitio"
	"github.com/wader/fq/pkg/decode"
	"github.com/wader/fq/pkg/scalar"
)

func init() {
	registry.MustRegister(decode.Format{
		Name:        format.MACHO,
		Description: "Mach-O macOS executable",
		Groups:      []string{format.PROBE},
		DecodeFn:    machoDecode,
	})
}

//nolint:revive
const (
	MH_MAGIC    = 0xfeed_face
	MH_CIGAM    = 0xcefa_edfe
	MH_MAGIC_64 = 0xfeed_facf
	MH_CIGAM_64 = 0xcffa_edfe
	FAT_MAGIC   = 0xcafe_babe
	FAT_CIGAM   = 0xbeba_feca
)

var magicSymMapper = scalar.UToScalar{
	MH_MAGIC:    scalar.S{Description: "32-bit little endian"},
	MH_CIGAM:    scalar.S{Description: "32-bit big endian"},
	MH_MAGIC_64: scalar.S{Description: "64-bit little endian"},
	MH_CIGAM_64: scalar.S{Description: "64-bit big endian"},
}

var endianNames = scalar.UToSymStr{
	MH_MAGIC:    "little_endian",
	MH_CIGAM:    "big_endian",
	MH_MAGIC_64: "little_endian",
	MH_CIGAM_64: "big_endian",
}

var cpuTypes = scalar.UToSymStr{
	0xff_ff_ff_ff: "CPU_TYPE_ANY",
	1:             "CPU_TYPE_VAX",
	2:             "CPU_TYPE_ROMP",
	4:             "CPU_TYPE_NS32032",
	5:             "CPU_TYPE_NS32332",
	6:             "CPU_TYPE_MC680x0",
	7:             "CPU_TYPE_X86",
	8:             "CPU_TYPE_MIPS",
	9:             "CPU_TYPE_NS32532",
	10:            "CPU_TYPE_MC98000",
	11:            "CPU_TYPE_HPPA",
	12:            "CPU_TYPE_ARM",
	13:            "CPU_TYPE_MC88000",
	14:            "CPU_TYPE_SPARC",
	15:            "CPU_TYPE_I860",
	16:            "CPU_TYPE_I860_LITTLE",
	17:            "CPU_TYPE_RS6000",
	18:            "CPU_TYPE_POWERPC",
	0x1000007:     "CPU_TYPE_X86_64",
	0x100000c:     "CPU_TYPE_ARM64",
	0x1000013:     "CPU_TYPE_POWERPC64",
	255:           "CPU_TYPE_VEO",
}

var cpuSubTypes = map[uint64]scalar.UToSymStr{
	0xff_ff_ff_ff: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
	},
	1: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_VAX_ALL",
		1:             "CPU_SUBTYPE_VAX780",
		2:             "CPU_SUBTYPE_VAX785",
		3:             "CPU_SUBTYPE_VAX750",
		4:             "CPU_SUBTYPE_VAX730",
		5:             "CPU_SUBTYPE_UVAXI",
		6:             "CPU_SUBTYPE_UVAXII",
		7:             "CPU_SUBTYPE_VAX8200",
		8:             "CPU_SUBTYPE_VAX8500",
		9:             "CPU_SUBTYPE_VAX8600",
		10:            "CPU_SUBTYPE_VAX8650",
		11:            "CPU_SUBTYPE_VAX8800",
		12:            "CPU_SUBTYPE_UVAXIII",
	},
	6: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		1:             "CPU_SUBTYPE_MC680X0_ALL", // 1: CPU_SUBTYPE_MC68030
		2:             "CPU_SUBTYPE_MC68040",
		3:             "CPU_SUBTYPE_MC68030_ONLY",
	},
	7: {
		0xff_ff_ff_ff:             "CPU_SUBTYPE_MULTIPLE",
		intelSubTypeHelper(3, 0):  "CPU_SUBTYPE_I386_ALL", // CPU_SUBTYPE_I386
		intelSubTypeHelper(4, 0):  "CPU_SUBTYPE_I486",
		intelSubTypeHelper(4, 8):  "CPU_SUBTYPE_486SX",
		intelSubTypeHelper(5, 0):  "CPU_SUBTYPE_PENT",
		intelSubTypeHelper(6, 1):  "CPU_SUBTYPE_PENTPRO",
		intelSubTypeHelper(6, 3):  "CPU_SUBTYPE_PENTII_M3",
		intelSubTypeHelper(6, 5):  "CPU_SUBTYPE_PENTII_M5",
		intelSubTypeHelper(7, 6):  "CPU_SUBTYPE_CELERON",
		intelSubTypeHelper(7, 7):  "CPU_SUBTYPE_CELERON_MOBILE",
		intelSubTypeHelper(8, 0):  "CPU_SUBTYPE_PENTIUM_3",
		intelSubTypeHelper(8, 1):  "CPU_SUBTYPE_PENTIUM_3_M",
		intelSubTypeHelper(8, 2):  "CPU_SUBTYPE_PENTIUM_3_XEON",
		intelSubTypeHelper(9, 0):  "CPU_SUBTYPE_PENTIUM_M",
		intelSubTypeHelper(10, 0): "CPU_SUBTYPE_PENTIUM_4",
		intelSubTypeHelper(10, 1): "CPU_SUBTYPE_PENTIUM_4_M",
		intelSubTypeHelper(11, 0): "CPU_SUBTYPE_ITANIUM",
		intelSubTypeHelper(11, 1): "CPU_SUBTYPE_ITANIUM_2",
		intelSubTypeHelper(12, 0): "CPU_SUBTYPE_XEON",
		intelSubTypeHelper(12, 1): "CPU_SUBTYPE_XEON_2",
	},
	8: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_MIPS_ALL",
		1:             "CPU_SUBTYPE_MIPS_R2300",
		2:             "CPU_SUBTYPE_MIPS_R2600",
		3:             "CPU_SUBTYPE_MIPS_R2800",
		4:             "CPU_SUBTYPE_MIPS_R2000A",
		5:             "CPU_SUBTYPE_MIPS_R2000",
		6:             "CPU_SUBTYPE_MIPS_R3000A",
		7:             "CPU_SUBTYPE_MIPS_R3000",
	},
	10: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_MC98000_ALL",
		1:             "CPU_SUBTYPE_MC98001",
	},
	11: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_HPPA_ALL",
		1:             "CPU_SUBTYPE_HPPA_7100",
		2:             "CPU_SUBTYPE_HPPA_7100_LC",
	},
	12: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_ARM_ALL",
		5:             "CPU_SUBTYPE_ARM_V4T",
		6:             "CPU_SUBTYPE_ARM_V6",
		7:             "CPU_SUBTYPE_ARM_V5TEJ",
		8:             "CPU_SUBTYPE_ARM_XSCALE",
		9:             "CPU_SUBTYPE_ARM_V7",
		10:            "CPU_SUBTYPE_ARM_V7F",
		11:            "CPU_SUBTYPE_ARM_V7S",
		12:            "CPU_SUBTYPE_ARM_V7K",
		13:            "CPU_SUBTYPE_ARM_V8",
		14:            "CPU_SUBTYPE_ARM_V6M",
		15:            "CPU_SUBTYPE_ARM_V7M",
		16:            "CPU_SUBTYPE_ARM_V7EM",
	},
	13: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_MC88000_ALL",
		1:             "CPU_SUBTYPE_MC88100",
		2:             "CPU_SUBTYPE_MC88110",
	},
	14: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_SPARC_ALL",
	},
	15: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_I860_ALL",
		1:             "CPU_SUBTYPE_I860_A860",
	},
	18: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_POWERPC_ALL",
		1:             "CPU_SUBTYPE_POWERPC_601",
		2:             "CPU_SUBTYPE_POWERPC_602",
		3:             "CPU_SUBTYPE_POWERPC_603",
		4:             "CPU_SUBTYPE_POWERPC_603E",
		5:             "CPU_SUBTYPE_POWERPC_603EV",
		6:             "CPU_SUBTYPE_POWERPC_604",
		7:             "CPU_SUBTYPE_POWERPC_604E",
		8:             "CPU_SUBTYPE_POWERPC_620",
		9:             "CPU_SUBTYPE_POWERPC_750",
		10:            "CPU_SUBTYPE_POWERPC_7400",
		11:            "CPU_SUBTYPE_POWERPC_7450",
		100:           "CPU_SUBTYPE_POWERPC_970",
	},
	0x1000012: {
		0xff_ff_ff_ff: "CPU_SUBTYPE_MULTIPLE",
		0:             "CPU_SUBTYPE_ARM64_ALL",
		1:             "CPU_SUBTYPE_ARM64_V8",
		2:             "CPU_SUBTYPE_ARM64_E",
	},
}

//nolint:revive
const (
	LC_REQ_DYLD                 = 0x80000000
	LC_SEGMENT                  = 0x1
	LC_SYMTAB                   = 0x2
	LC_SYMSEG                   = 0x3
	LC_THREAD                   = 0x4
	LC_UNIXTHREAD               = 0x5
	LC_LOADFVMLIB               = 0x6
	LC_IDFVMLIB                 = 0x7
	LC_IDENT                    = 0x8 // not implemented
	LC_FVMFILE                  = 0x9 // not implemented
	LC_PREPAGE                  = 0xa // not implemented
	LC_DYSYMTAB                 = 0xb
	LC_LOAD_DYLIB               = 0xc
	LC_ID_DYLIB                 = 0xd
	LC_LOAD_DYLINKER            = 0xe
	LC_ID_DYLINKER              = 0xf
	LC_PREBOUND_DYLIB           = 0x10
	LC_ROUTINES                 = 0x11
	LC_SUB_FRAMEWORK            = 0x12
	LC_SUB_UMBRELLA             = 0x13
	LC_SUB_CLIENT               = 0x14
	LC_SUB_LIBRARY              = 0x15
	LC_TWOLEVEL_HINTS           = 0x16
	LC_PREBIND_CKSUM            = 0x17 // not implemented
	LC_LOAD_WEAK_DYLIB          = 0x80000018
	LC_SEGMENT_64               = 0x19
	LC_ROUTINES_64              = 0x1a
	LC_UUID                     = 0x1b
	LC_RPATH                    = 0x8000001c
	LC_CODE_SIGNATURE           = 0x1d
	LC_SEGMENT_SPLIT_INFO       = 0x1e
	LC_REEXPORT_DYLIB           = 0x8000001f
	LC_LAZY_LOAD_DYLIB          = 0x20
	LC_ENCRYPTION_INFO          = 0x21
	LC_DYLD_INFO                = 0x22
	LC_DYLD_INFO_ONLY           = 0x80000022
	LC_LOAD_UPWARD_DYLIB        = 0x80000023
	LC_VERSION_MIN_MACOSX       = 0x24
	LC_VERSION_MIN_IPHONEOS     = 0x25
	LC_FUNCTION_STARTS          = 0x26
	LC_DYLD_ENVIRONMENT         = 0x27
	LC_MAIN                     = 0x80000028
	LC_DATA_IN_CODE             = 0x29
	LC_SOURCE_VERSION           = 0x2A
	LC_DYLIB_CODE_SIGN_DRS      = 0x2B
	LC_ENCRYPTION_INFO_64       = 0x2C
	LC_LINKER_OPTION            = 0x2D
	LC_LINKER_OPTIMIZATION_HINT = 0x2E
	LC_VERSION_MIN_TVOS         = 0x2F
	LC_VERSION_MIN_WATCHOS      = 0x30
	LC_NOTE                     = 0x31 // not implemented
	LC_BUILD_VERSION            = 0x32
)

var fileTypes = scalar.UToSymStr{
	0x1: "MH_OBJECT",
	0x2: "MH_EXECUTE",
	0x3: "MH_FVMLIB",
	0x4: "MH_CORE",
	0x5: "MH_PRELOAD",
	0x6: "MH_DYLIB",
	0x7: "MH_DYLINKER",
	0x8: "MH_BUNDLE",
	0x9: "MH_DYLIB_STUB",
	0xa: "MH_DSYM",
	0xb: "MH_KEXT_BUNDLE",
}

var loadCommands = scalar.UToSymStr{
	LC_REQ_DYLD:                 "req_dyld",
	LC_SEGMENT:                  "segment",
	LC_SYMTAB:                   "symtab",
	LC_SYMSEG:                   "symseg",
	LC_THREAD:                   "thread",
	LC_UNIXTHREAD:               "unixthread",
	LC_LOADFVMLIB:               "loadfvmlib",
	LC_IDFVMLIB:                 "idfvmlib",
	LC_IDENT:                    "ident",
	LC_FVMFILE:                  "fvmfile",
	LC_PREPAGE:                  "prepage",
	LC_DYSYMTAB:                 "dysymtab",
	LC_LOAD_DYLIB:               "load_dylib",
	LC_ID_DYLIB:                 "id_dylib",
	LC_LOAD_DYLINKER:            "load_dylinker",
	LC_ID_DYLINKER:              "id_dylinker",
	LC_PREBOUND_DYLIB:           "prebound_dylib",
	LC_ROUTINES:                 "routines",
	LC_SUB_FRAMEWORK:            "sub_framework",
	LC_SUB_UMBRELLA:             "sub_umbrella",
	LC_SUB_CLIENT:               "sub_client",
	LC_SUB_LIBRARY:              "sub_library",
	LC_TWOLEVEL_HINTS:           "twolevel_hints",
	LC_PREBIND_CKSUM:            "prebind_cksum",
	LC_LOAD_WEAK_DYLIB:          "load_weak_dylib",
	LC_SEGMENT_64:               "segment_64",
	LC_ROUTINES_64:              "routines_64",
	LC_UUID:                     "uuid",
	LC_RPATH:                    "rpath",
	LC_CODE_SIGNATURE:           "code_signature",
	LC_SEGMENT_SPLIT_INFO:       "segment_split_info",
	LC_REEXPORT_DYLIB:           "reexport_dylib",
	LC_LAZY_LOAD_DYLIB:          "lazy_load_dylib",
	LC_ENCRYPTION_INFO:          "encryption_info",
	LC_DYLD_INFO:                "dyld_info",
	LC_DYLD_INFO_ONLY:           "dyld_info_only",
	LC_LOAD_UPWARD_DYLIB:        "load_upward_dylib",
	LC_VERSION_MIN_MACOSX:       "version_min_macosx",
	LC_VERSION_MIN_IPHONEOS:     "version_min_iphoneos",
	LC_FUNCTION_STARTS:          "function_starts",
	LC_DYLD_ENVIRONMENT:         "dyld_environment",
	LC_MAIN:                     "main",
	LC_DATA_IN_CODE:             "data_in_code",
	LC_SOURCE_VERSION:           "source_version",
	LC_DYLIB_CODE_SIGN_DRS:      "dylib_code_sign_drs",
	LC_ENCRYPTION_INFO_64:       "encryption_info_64",
	LC_LINKER_OPTION:            "linker_option",
	LC_LINKER_OPTIMIZATION_HINT: "linker_optimization_hint",
	LC_VERSION_MIN_TVOS:         "version_min_tvos",
	LC_VERSION_MIN_WATCHOS:      "version_min_watchos",
	LC_NOTE:                     "note",
	LC_BUILD_VERSION:            "build_version",
}

var sectionTypes = scalar.UToSymStr{
	0x0:  "S_REGULAR",
	0x1:  "S_ZEROFILL",
	0x2:  "S_CSTRING_LITERALS",
	0x3:  "S_4BYTE_LITERALS",
	0x4:  "S_8BYTE_LITERALS",
	0x5:  "S_LITERAL_POINTERS",
	0x6:  "S_NON_LAZY_SYMBOL_POINTERS",
	0x7:  "S_LAZY_SYMBOL_POINTERS",
	0x8:  "S_SYMBOL_STUBS",
	0x9:  "S_MOD_INIT_FUNC_POINTERS",
	0xa:  "S_MOD_TERM_FUNC_POINTERS",
	0xb:  "S_COALESCED",
	0xc:  "S_GB_ZEROFILL",
	0xd:  "S_INTERPOSING",
	0xe:  "S_16BYTE_LITERALS",
	0xf:  "S_DTRACE_DOF",
	0x10: "S_LAZY_DYLIB_SYMBOL_POINTERS",
	0x11: "S_THREAD_LOCAL_REGULAR",
	0x12: "S_THREAD_LOCAL_ZEROFILL",
	0x13: "S_THREAD_LOCAL_VARIABLES",
	0x14: "S_THREAD_LOCAL_VARIABLE_POINTERS",
	0x15: "S_THREAD_LOCAL_INIT_FUNCTION_POINTERS",
}

func machoDecode(d *decode.D, in interface{}) interface{} {
	ofileDecode(d)
	return nil
}

func ofileDecode(d *decode.D) {
	var archBits int
	var cpuType uint64
	var ncmds uint64
	magicBuffer := d.U32LE()

	if magicBuffer == MH_MAGIC || magicBuffer == MH_MAGIC_64 {
		d.Endian = decode.LittleEndian
		if magicBuffer == MH_MAGIC {
			archBits = 32
		} else {
			archBits = 64
		}
	} else if magicBuffer == MH_CIGAM || magicBuffer == MH_CIGAM_64 {
		d.Endian = decode.BigEndian
		if magicBuffer == MH_CIGAM {
			archBits = 32
		} else {
			archBits = 64
		}
	} else if magicBuffer == FAT_MAGIC {
		d.Endian = decode.LittleEndian
		fatParse(d)
		return
	} else if magicBuffer == FAT_CIGAM {
		d.Endian = decode.BigEndian
		fatParse(d)
		return
	} else {
		// AR files are also valid OFiles but they should be parsed by `-d ar`
		d.Fatalf("Invalid magic field")
	}

	d.SeekRel(-4 * 8)
	d.FieldStruct("header", func(d *decode.D) {
		d.FieldValueS("arch_bits", int64(archBits))
		magic := d.FieldU32("magic", scalar.Hex, magicSymMapper)
		d.FieldValueU("bits", uint64(archBits))
		d.FieldValueStr("endian", endianNames[magic])
		cpuType = d.FieldU32("cputype", cpuTypes)
		d.FieldU32("cpusubtype", cpuSubTypes[cpuType])
		d.FieldU32("filetype", fileTypes)
		ncmds = d.FieldU32("ncdms")
		d.FieldU32("sizeofncdms")
		d.FieldStruct("flags", parseMachHeaderFlags)
		if archBits == 64 {
			d.FieldRawLen("reserved", 4*8, d.BitBufIsZero())
		}
	})
	ncmdsIdx := 0
	d.FieldStructArrayLoop("load_commands", "load_command", func() bool {
		return ncmdsIdx < int(ncmds)
	}, func(d *decode.D) {
		cmd := d.FieldU32("cmd", loadCommands, scalar.Hex)
		cmdsize := d.FieldU32("cmdsize")
		switch cmd {
		case LC_UUID:
			d.FieldStruct("uuid_command", func(d *decode.D) {
				d.FieldRawLen("uuid", 16*8)
			})
		case LC_SEGMENT, LC_SEGMENT_64:
			// nsect := (cmdsize - uint64(archBits)) / uint64(archBits)
			var nsects uint64
			d.FieldStruct("segment_command", func(d *decode.D) {
				d.FieldValueS("arch_bits", int64(archBits))
				d.FieldUTF8NullFixedLen("segname", 16) // OPCODE_DECODER segname==__TEXT
				if archBits == 32 {
					d.FieldU32("vmaddr", scalar.Hex)
					d.FieldU32("vmsize")
					d.FieldU32("fileoff")
					d.FieldU32("tfilesize")
				} else {
					d.FieldU64("vmaddr", scalar.Hex)
					d.FieldU64("vmsize")
					d.FieldU64("fileoff")
					d.FieldU64("tfilesize")
				}
				d.FieldS32("initprot")
				d.FieldS32("maxprot")
				nsects = d.FieldU32("nsects")
				d.FieldStruct("flags", parseSegmentFlags)
			})
			var nsectIdx uint64
			d.FieldStructArrayLoop("sections", "section", func() bool {
				return nsectIdx < nsects
			},
				func(d *decode.D) {
					// OPCODE_DECODER sectname==__text
					d.FieldUTF8NullFixedLen("sectname", 16)
					d.FieldUTF8NullFixedLen("segname", 16)
					if archBits == 32 {
						d.FieldU32("address", scalar.Hex)
						d.FieldU32("size")
					} else {
						d.FieldU64("address", scalar.Hex)
						d.FieldU64("size")
					}
					d.FieldU32("offset")
					d.FieldU32("align")
					d.FieldU32("reloff")
					d.FieldU32("nreloc")
					// get section type
					d.FieldStruct("flags", parseSectionFlags)
					d.FieldU8("type", sectionTypes)
					d.FieldU32("reserved1")
					d.FieldU32("reserved2")
					if archBits == 64 {
						d.FieldU32("reserved3")
					}
					nsectIdx++
				})
		case LC_TWOLEVEL_HINTS:
			d.FieldU32("offset")
			d.FieldU32("nhints")
		case LC_LOAD_DYLIB, LC_ID_DYLIB, LC_LOAD_UPWARD_DYLIB, LC_LOAD_WEAK_DYLIB, LC_LAZY_LOAD_DYLIB, LC_REEXPORT_DYLIB:
			d.FieldStruct("dylib_command", func(d *decode.D) {
				offset := d.FieldU32("offset")
				d.FieldU32("timestamp", timestampMapper)
				d.FieldU32("current_version")
				d.FieldU32("compatibility_version")
				d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
			})
		case LC_LOAD_DYLINKER, LC_ID_DYLINKER, LC_DYLD_ENVIRONMENT:
			offset := d.FieldU32("offset")
			d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
		case LC_RPATH:
			offset := d.FieldU32("offset")
			d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
		case LC_PREBOUND_DYLIB:
			// https://github.com/aidansteele/osx-abi-macho-file-format-reference#prebound_dylib_command
			d.U32() // name_offset
			nmodules := d.FieldU32("nmodules")
			d.U32() // linked_modules_offset
			d.FieldUTF8Null("name")
			d.FieldBitBufFn("linked_modules", func(d *decode.D) bitio.ReaderAtSeeker {
				return d.RawLen(int64((nmodules / 8) + (nmodules % 8)))
			})
		case LC_THREAD, LC_UNIXTHREAD:
			d.FieldU32("flavor")
			count := d.FieldU32("count")
			d.FieldStruct("state", func(d *decode.D) {
				switch cpuType {
				case 0x7:
					threadStateI386Decode(d)
				case 0xC:
					threadStateARM32Decode(d)
				case 0x13:
					threadStatePPC32Decode(d)
				case 0x1000007:
					threadStateX8664Decode(d)
				case 0x100000C:
					threadStateARM64Decode(d)
				case 0x1000013:
					threadStatePPC64Decode(d)
				default:
					d.FieldRawLen("state", int64(count*32))
				}
			})
		case LC_ROUTINES, LC_ROUTINES_64:
			if archBits == 32 {
				d.FieldU32("init_address", scalar.Hex)
				d.FieldU32("init_module")
				d.FieldU32("reserved1")
				d.FieldU32("reserved2")
				d.FieldU32("reserved3")
				d.FieldU32("reserved4")
				d.FieldU32("reserved5")
				d.FieldU32("reserved6")
			} else {
				d.FieldU64("init_address", scalar.Hex)
				d.FieldU64("init_module")
				d.FieldU64("reserved1")
				d.FieldU64("reserved2")
				d.FieldU64("reserved3")
				d.FieldU64("reserved4")
				d.FieldU64("reserved5")
				d.FieldU64("reserved6")
			}
		case LC_SUB_UMBRELLA, LC_SUB_LIBRARY, LC_SUB_CLIENT, LC_SUB_FRAMEWORK:
			offset := d.FieldU32("offset")
			d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
		case LC_SYMTAB:
			d.FieldU32("symoff")
			d.FieldU32("nsyms")
			d.FieldU32("stroff")
			d.FieldU32("strsize")
		case LC_DYSYMTAB:
			d.FieldU32("ilocalsym")
			d.FieldU32("nlocalsym")
			d.FieldU32("iextdefsym")
			d.FieldU32("nextdefsym")
			d.FieldU32("iundefsym")
			d.FieldU32("nundefsym")
			d.FieldU32("tocoff")
			d.FieldU32("ntoc")
			d.FieldU32("modtaboff")
			d.FieldU32("nmodtab")
			d.FieldU32("extrefsymoff")
			d.FieldU32("nextrefsyms")
			d.FieldU32("indirectsymoff")
			d.FieldU32("nindirectsyms")

			d.FieldU32("extreloff")
			d.FieldU32("nextrel")
			d.FieldU32("locreloff")
			d.FieldU32("nlocrel")
		case LC_BUILD_VERSION:
			d.FieldU32("platform")
			d.FieldU32("minos")
			d.FieldU32("sdk")
			ntools := d.FieldU32("ntools")
			var ntoolsIdx uint64
			d.FieldStructArrayLoop("tools", "tool", func() bool {
				return ntoolsIdx < ntools
			}, func(d *decode.D) {
				d.FieldU32("tool")
				d.FieldU32("version")
				ntoolsIdx++
			})
		case LC_CODE_SIGNATURE, LC_SEGMENT_SPLIT_INFO, LC_FUNCTION_STARTS, LC_DATA_IN_CODE, LC_DYLIB_CODE_SIGN_DRS, LC_LINKER_OPTIMIZATION_HINT:
			d.FieldStruct("linkedit_data", func(d *decode.D) {
				d.FieldU32("off")
				d.FieldU32("size")
			})
		case LC_VERSION_MIN_IPHONEOS, LC_VERSION_MIN_MACOSX, LC_VERSION_MIN_TVOS, LC_VERSION_MIN_WATCHOS:
			d.FieldU32("version")
			d.FieldU32("sdk")
		case LC_DYLD_INFO, LC_DYLD_INFO_ONLY:
			d.FieldStruct("dyld_info", func(d *decode.D) {
				d.FieldU32("rebase_off")
				d.FieldU32("rebase_size")
				d.FieldU32("bind_off")
				d.FieldU32("bind_size")
				d.FieldU32("weak_bind_off")
				d.FieldU32("weak_bind_size")
				d.FieldU32("lazy_bind_off")
				d.FieldU32("lazy_bind_size")
				d.FieldU32("export_off")
				d.FieldU32("export_size")
			})
		case LC_MAIN:
			d.FieldStruct("entrypoint", func(d *decode.D) {
				d.FieldU64("entryoff")
				d.FieldU64("stacksize")
			})
		case LC_SOURCE_VERSION:
			d.FieldStruct("source_version_tag", func(d *decode.D) {
				d.FieldU64("tag")
			})
		case LC_LINKER_OPTION:
			d.FieldStruct("linker_option", func(d *decode.D) {
				count := d.FieldU32("count")
				d.FieldUTF8NullFixedLen("option", int(count))
			})
		case LC_ENCRYPTION_INFO, LC_ENCRYPTION_INFO_64:
			d.FieldStruct("encryption_info", func(d *decode.D) {
				d.FieldU32("offset")
				d.FieldU32("size")
				d.FieldU32("id")
			})
		case LC_IDFVMLIB, LC_LOADFVMLIB:
			d.FieldStruct("fvmlib", func(d *decode.D) {
				offset := d.FieldU32("offset")
				d.FieldU32("minor_version")
				d.FieldU32("header_addr", scalar.Hex)
				d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
			})
		default:
			if _, ok := loadCommands[cmd]; !ok {
				d.SeekRel(int64((cmdsize - 8) * 8))
				// Seek Rel so the parts are marked unknown
			}
		}
		ncmdsIdx++
	})
}

func fatParse(d *decode.D) {
	// Go to start of the file again
	d.SeekAbs(0)
	var narchs uint64
	var ofileOffsets []uint64
	d.FieldStruct("fat_header", func(d *decode.D) {
		d.FieldRawLen("magic", 4*8)
		narchs = d.FieldU32("narchs")
		narchsIdx := 0

		d.FieldStructArrayLoop("archs", "arch", func() bool {
			return narchsIdx < int(narchs)
		}, func(d *decode.D) {
			// parse FatArch
			d.FieldStruct("fat_arch", func(d *decode.D) {
				// beware cputype and cpusubtype changes from ofile header to fat header
				cpuType := d.FieldU32("cputype", cpuTypes)
				d.FieldU32("cpusubtype", cpuSubTypes[cpuType])
				ofileOffsets = append(ofileOffsets, d.FieldU32("offset"))
				d.FieldU32("size")
				d.FieldU32("align")
			})
			narchsIdx++
		})
	})
	nfilesIdx := 0
	d.FieldStructArrayLoop("files", "file", func() bool {
		return nfilesIdx < int(narchs)
	}, func(d *decode.D) {
		d.SeekAbs(int64(ofileOffsets[nfilesIdx]) * 8)
		ofileDecode(d)
		nfilesIdx++
	})
}

func intelSubTypeHelper(f, m uint64) uint64 {
	return f + (m << 4)
}

func parseMachHeaderFlags(d *decode.D) {
	d.FieldRawLen("reserved", 6)
	d.FieldBool("MH_APP_EXTENSION_SAFE")
	d.FieldBool("MH_NO_HEAP_EXECUTION")

	d.FieldBool("MH_HAS_TLV_DESCRIPTORS")
	d.FieldBool("MH_DEAD_STRIPPABLE_DYLIB")
	d.FieldBool("MH_PIE")
	d.FieldBool("MH_NO_REEXPORTED_DYLIBS")

	d.FieldBool("MH_SETUID_SAFE")
	d.FieldBool("MH_ROOT_SAFE")
	d.FieldBool("MH_ALLOW_STACK_EXECUTION")
	d.FieldBool("MH_BINDS_TO_WEAK")

	d.FieldBool("MH_WEAK_DEFINES")
	d.FieldBool("MH_CANONICAL")
	d.FieldBool("MH_SUBSECTIONS_VIA_SYMBOLS")
	d.FieldBool("MH_ALLMODSBOUND")

	d.FieldBool("MH_PREBINDABLE")
	d.FieldBool("MH_NOFIXPREBINDING")
	d.FieldBool("MH_NOMULTIDEFS")
	d.FieldBool("MH_FORCE_FLAT")

	d.FieldBool("MH_TWOLEVEL")
	d.FieldBool("MH_LAZY_INIT")
	d.FieldBool("MH_SPLIT_SEGS")
	d.FieldBool("MH_PREBOUND")

	d.FieldBool("MH_BINDATLOAD")
	d.FieldBool("MH_DYLDLINK")
	d.FieldBool("MH_INCRLINK")
	d.FieldBool("MH_NOUNDEFS")
}

func parseSegmentFlags(d *decode.D) {
	d.FieldRawLen("reserved", 28)
	d.FieldBool("SG_PROTECTED_VERSION_1")
	d.FieldBool("SG_NORELOC")
	d.FieldBool("SG_FVMLIB")
	d.FieldBool("SG_HIGHVM")
}

func parseSectionFlags(d *decode.D) {
	d.FieldBool("S_ATTR_PURE_INSTRUCTIONS")
	d.FieldBool("S_ATTR_NO_TOC")
	d.FieldBool("S_ATTR_STRIP_STATIC_SYMS")
	d.FieldBool("S_ATTR_NO_DEAD_STRIP")

	d.FieldBool("S_ATTR_LIVE_SUPPORT")
	d.FieldBool("S_ATTR_SELF_MODIFYING_CODE")
	d.FieldBool("S_ATTR_DEBUG")
	d.FieldRawLen("reserved", 14)

	d.FieldBool("S_ATTR_SOME_INSTRUCTIONS")
	d.FieldBool("S_ATTR_EXT_RELOC")
	d.FieldBool("S_ATTR_LOC_RELOC")
}

var timestampMapper = scalar.Fn(func(s scalar.S) (scalar.S, error) {
	ts, ok := s.Actual.(uint64)
	if !ok {
		return s, nil
	}
	s.Sym = time.UnixMilli(int64(ts)).UTC().String()
	return s, nil
})

func threadStateI386Decode(d *decode.D) {
	d.FieldU32("eax")
	d.FieldU32("ebx")
	d.FieldU32("ecx")
	d.FieldU32("edx")
	d.FieldU32("edi")
	d.FieldU32("esi")
	d.FieldU32("ebp")
	d.FieldU32("esp")
	d.FieldU32("ss")
	d.FieldU32("eflags")
	d.FieldU32("eip")
	d.FieldU32("cs")
	d.FieldU32("ds")
	d.FieldU32("es")
	d.FieldU32("fs")
	d.FieldU32("gs")
}

func threadStateX8664Decode(d *decode.D) {
	d.FieldU64("rax")
	d.FieldU64("rbx")
	d.FieldU64("rcx")
	d.FieldU64("rdx")
	d.FieldU64("rdi")
	d.FieldU64("rsi")
	d.FieldU64("rbp")
	d.FieldU64("rsp")
	d.FieldU64("r8")
	d.FieldU64("r9")
	d.FieldU64("r10")
	d.FieldU64("r11")
	d.FieldU64("r12")
	d.FieldU64("r13")
	d.FieldU64("r14")
	d.FieldU64("r15")
	d.FieldU64("rip")
	d.FieldU64("rflags")
	d.FieldU64("cs")
	d.FieldU64("fs")
	d.FieldU64("gs")
}

func threadStateARM32Decode(d *decode.D) {
	rIdx := 0
	d.FieldStructArrayLoop("r", "r", func() bool {
		return rIdx < 13
	}, func(d *decode.D) {
		d.FieldU32("value")
		rIdx++
	})
	d.FieldU32("sp")
	d.FieldU32("lr")
	d.FieldU32("pc")
	d.FieldU32("cpsr")
}

func threadStateARM64Decode(d *decode.D) {
	rIdx := 0
	d.FieldStructArrayLoop("r", "r", func() bool {
		return rIdx < 29
	}, func(d *decode.D) {
		d.FieldU64("value")
		rIdx++
	})
	d.FieldU64("fp")
	d.FieldU64("lr")
	d.FieldU64("sp")
	d.FieldU64("pc")
	d.FieldU32("cpsr")
	d.FieldU32("pad")
}

func threadStatePPC32Decode(d *decode.D) {
	srrIdx := 0
	d.FieldStructArrayLoop("srr", "srr", func() bool {
		return srrIdx < 2
	}, func(d *decode.D) {
		d.FieldU32("value")
		srrIdx++
	})
	rIdx := 0
	d.FieldStructArrayLoop("r", "r", func() bool {
		return rIdx < 32
	}, func(d *decode.D) {
		d.FieldU32("value")
		rIdx++
	})
	d.FieldU32("ct")
	d.FieldU32("xer")
	d.FieldU32("lr")
	d.FieldU32("ctr")
	d.FieldU32("mq")
	d.FieldU32("vrsave")
}

func threadStatePPC64Decode(d *decode.D) {
	srrIdx := 0
	d.FieldStructArrayLoop("srr", "srr", func() bool {
		return srrIdx < 2
	}, func(d *decode.D) {
		d.FieldU64("value")
		srrIdx++
	})
	rIdx := 0
	d.FieldStructArrayLoop("r", "r", func() bool {
		return rIdx < 32
	}, func(d *decode.D) {
		d.FieldU64("value")
		rIdx++
	})
	d.FieldU32("ct")
	d.FieldU64("xer")
	d.FieldU64("lr")
	d.FieldU64("ctr")
	d.FieldU32("vrsave")
}
