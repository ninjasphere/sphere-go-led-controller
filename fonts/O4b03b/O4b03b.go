// XXXX X X      XXX XXX
// X  X X X X    X X   X X
// X  X XXX XXXX X X XXX XXXX
// X  X   X X  X X X   X X  X
// XXXX   X XXXX XXX XXX XXXX

package O4b03b

import "github.com/ninjasphere/pixfont"

var Font *pixfont.PixFont

func init() {
	charMap := map[int32]uint16{58: 0x7b, 50: 0xa9, 73: 0xd8, 62: 0xdb, 94: 0xf1, 88: 0x168, 38: 0x31, 115: 0x10a, 114: 0x153, 119: 0x1f9, 109: 0x61, 46: 0x32, 56: 0xc2, 45: 0xc3, 90: 0xd9, 86: 0x13b, 55: 0x1b1, 93: 0x19, 33: 0x63, 110: 0xc0, 112: 0x169, 78: 0x19b, 117: 0x1c9, 113: 0x62, 69: 0x91, 82: 0xa8, 60: 0xab, 48: 0x10b, 72: 0x183, 77: 0x1e2, 79: 0x0, 125: 0x7a, 95: 0x90, 74: 0x92, 103: 0x109, 75: 0x13a, 108: 0x1b, 101: 0x4b, 41: 0xf0, 57: 0x199, 123: 0x1b2, 42: 0x1fb, 126: 0x210, 83: 0x1a, 120: 0x30, 68: 0x150, 63: 0x18, 104: 0x60, 80: 0x93, 64: 0x1ca, 35: 0x3, 40: 0x33, 111: 0xda, 36: 0x1cb, 59: 0x1e0, 91: 0x1fa, 54: 0x2, 43: 0x123, 61: 0x79, 85: 0x49, 37: 0x78, 84: 0x121, 102: 0x122, 87: 0x152, 122: 0x180, 105: 0x1, 97: 0x4a, 52: 0xaa, 65: 0xf2, 49: 0x120, 34: 0x138, 53: 0x181, 106: 0x1e3, 81: 0x48, 107: 0x1f8, 121: 0x16b, 99: 0x108, 71: 0x139, 76: 0x151, 118: 0x16a, 70: 0x182, 67: 0x1e1, 51: 0xc1, 100: 0x198, 47: 0x19a, 116: 0x1b0, 89: 0x1b3, 98: 0x1c8, 66: 0xf3}
	data := []uint32{0xa03000f, 0x1f010109, 0xa070009, 0x1f050109, 0xa07010f, 0x0, 0xf030f, 0x1010208, 0x10f020e, 0x1080200, 0x10f0302, 0x0, 0x2000700, 0x1000100, 0x1001f05, 0x1000902, 0x2010f05, 0x0, 0x90f, 0x909, 0xf0e0909, 0x509090d, 0xe0f0f0f, 0x0, 0x1000000, 0x1000001, 0x10f1f0f, 0x91509, 0x10f1509, 0x80000, 0x30013, 0x102070b, 0x40004, 0x102071a, 0x30019, 0x0, 0xf0c0f00, 0x9080100, 0x9080f00, 0xf090100, 0x10f0f0f, 0x0, 0x405070f, 0x2050409, 0x107070f, 0x2040105, 0x404070d, 0x0, 0x70700, 0x50400, 0x707070f, 0x50409, 0x70709, 0x0, 0x2000f07, 0x4000802, 0x80f0602, 0x4090102, 0x20f0f07, 0x0, 0x70f0201, 0x5090502, 0xf0f0002, 0x9090002, 0xf090001, 0x0, 0x7000000, 0x5000000, 0x50f0f07, 0x5020901, 0x70f0f07, 0x400, 0x701, 0x20e0201, 0x7020201, 0x20f0201, 0x20201, 0x0, 0x9090f05, 0x9090105, 0x5070d00, 0x5090900, 0x2090f00, 0x0, 0x150107, 0x150109, 0x7150109, 0x1150109, 0x11f0f07, 0x0, 0x9, 0x9, 0x9090f06, 0x9090909, 0xf060f09, 0x8000100, 0x90f0700, 0x9010100, 0xf0f070f, 0x9010404, 0x901070f, 0x0, 0x9100700, 0xb080508, 0xd04070f, 0x9020409, 0x901060f, 0x0, 0x9060700, 0x9020402, 0xf010207, 0x8020202, 0xf060206, 0x0, 0xe1f0000, 0x5110001, 0xf1d090f, 0xa150909, 0x70f0f0f, 0x0, 0x1f0f00, 0x2150101, 0x150100, 0x2150101, 0x2150f01, 0x1000000, 0x5030000, 0x2010001, 0x5011509, 0x11507, 0x31f09, 0x0, 0xa, 0x5, 0x0, 0x0, 0x0, 0x0}
	Font = pixfont.NewPixFont(true, 5, 6, charMap, data)
}