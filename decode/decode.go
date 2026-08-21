// Package decode contiene la logica di decodifica condivisa dai driver vendor
// (tipizzazione dei registri U16, byte-order, sentinel NaN). Pura, senza I/O.
// Copiato da RemoteConnections/libraries-wasm/guest/decode, dove era gia'
// indipendente da wasm.
package decode

import (
	"encoding/binary"
	"math"
	"strings"
)

// Decodifica: registri U16 -> valore tipizzato; i sentinel NaN diventano
// "assente" (presente=false ⇒ null). tipo: u16 i16 u32 i32 u64 f32 string.
// ordineParole "swap" = parola bassa prima. La scala la applica il chiamante.
func Decodifica(parole []uint16, tipo, ordineParole string) (val float64, str string, isStr, presente bool) {
	if len(parole) == 0 {
		return 0, "", false, false
	}
	switch tipo {
	case "string":
		b := make([]byte, 0, len(parole)*2)
		for _, w := range parole {
			var tmp [2]byte
			binary.BigEndian.PutUint16(tmp[:], w)
			b = append(b, tmp[0], tmp[1])
		}
		if i := indexZero(b); i >= 0 {
			b = b[:i]
		}
		return 0, strings.TrimSpace(string(b)), true, true

	case "u16":
		v := parole[0]
		if v == 0xFFFF {
			return 0, "", false, false
		}
		return float64(v), "", false, true

	case "i16":
		v := parole[0]
		if v == 0x8000 {
			return 0, "", false, false
		}
		if v&0x8000 != 0 {
			return float64(int32(v) - 0x10000), "", false, true
		}
		return float64(v), "", false, true

	case "u64":
		if len(parole) < 4 {
			return 0, "", false, false
		}
		v := uint64(parole[0])<<48 | uint64(parole[1])<<32 | uint64(parole[2])<<16 | uint64(parole[3])
		if v == 0xFFFFFFFFFFFFFFFF {
			return 0, "", false, false
		}
		return float64(v), "", false, true
	}

	// tipi a 32 bit (2 registri)
	if len(parole) < 2 {
		return 0, "", false, false
	}
	var alta, bassa uint16
	if ordineParole == "swap" {
		alta, bassa = parole[1], parole[0]
	} else {
		alta, bassa = parole[0], parole[1]
	}
	v := uint32(alta)<<16 | uint32(bassa)

	switch tipo {
	case "u32":
		if v == 0xFFFFFFFF {
			return 0, "", false, false
		}
		return float64(v), "", false, true
	case "i32":
		if v == 0x80000000 {
			return 0, "", false, false
		}
		return float64(int32(v)), "", false, true
	case "f32":
		f64 := float64(math.Float32frombits(v))
		if math.IsNaN(f64) || math.IsInf(f64, 0) || f64 <= -1e12 || f64 >= 1e12 {
			return 0, "", false, false
		}
		return f64, "", false, true
	}
	return 0, "", false, false
}

func indexZero(b []byte) int {
	for i, c := range b {
		if c == 0 {
			return i
		}
	}
	return -1
}

// Round4 replica round(x, 4). Differisce dall'arrotondamento bancario di Python
// solo su pareggi esatti a 5 decimali, che coi dati reali non si presentano.
func Round4(x float64) float64 { return math.Round(x*1e4) / 1e4 }
