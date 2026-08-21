// Package comune raccoglie i tipi e gli helper di decodifica condivisi da
// tutti i driver vendor (mappa registri Modbus -> valori tipizzati). Copiato
// da RemoteConnections/libraries-wasm/guest/comune -- dove viveva come
// libreria condivisa dei vecchi guest wasm, prima che Hermod passasse a
// driver Go nativi -- tagliando cio' che serviva solo al vecchio protocollo
// guest/host (Risultato, Parse, Errore, Timeout, MustJSON): qui resta solo
// la logica di decodifica, pura, senza alcuna dipendenza da wasm.
package comune

import (
	"math"
	"strconv"

	"github.com/corna/pvdriver/decode"
)

// --------------------------------------------------------------------------- //
//  Ingresso / mappa registri                                                  //
// --------------------------------------------------------------------------- //

type Ingresso struct {
	Unit     int               `json:"unit"`
	Mappa    MappaReg          `json:"mappa"`
	Registri map[string]uint16 `json:"registri"`
	// RegistriExtra: set di registri letti ad ALTRI unit (driver multi-livello,
	// es. sigenergy: "device" = slave 1 mentre Registri = Plant slave 247).
	RegistriExtra map[string]map[string]uint16 `json:"registri_extra"`
	// Rest: corpo JSON di un'API REST, per i driver non-Modbus (sonnen). L'host
	// fa la GET e lo passa qui; il guest non fa I/O.
	Rest map[string]interface{} `json:"rest"`
}

type MappaReg struct {
	FunctionCode int      `json:"function_code"`
	WordOrder    string   `json:"word_order"`
	UnitID       int      `json:"unit_id"`
	Registers    []RegDef `json:"registers"`
	// DeviceRegisters: sotto-mappa a livello device (sigenergy).
	DeviceRegisters *SottoMappa `json:"device_registers"`
}

type SottoMappa struct {
	UnitID int      `json:"unit_id"`
	List   []RegDef `json:"list"`
}

type RegDef struct {
	Name      string   `json:"name"`
	Addr      int      `json:"addr"`
	Count     int      `json:"count"`
	Type      string   `json:"type"`
	Scale     *float64 `json:"scale"`
	WordOrder string   `json:"word_order"`
}

// --------------------------------------------------------------------------- //
//  Helper decodifica guidata da mappa (porta di leggi_mappa, scala fissa)     //
// --------------------------------------------------------------------------- //

// LeggiMappa: registri grezzi -> {nome: valore} usando i registri principali
// della mappa. Per i driver a scala fissa. I driver a scala dinamica
// (SolarEdge) NON la usano.
func LeggiMappa(in *Ingresso) map[string]interface{} {
	return LeggiRegistri(in.Mappa.Registers, in.Registri, in.Mappa.WordOrder)
}

// LeggiRegistri decodifica una lista di RegDef da un set di registri grezzi.
// Riusabile per sotto-mappe (device_registers) e blocchi extra.
func LeggiRegistri(registers []RegDef, registri map[string]uint16, ordineDefault string) map[string]interface{} {
	if ordineDefault == "" {
		ordineDefault = "big"
	}
	g := map[string]interface{}{}
	for _, reg := range registers {
		count := reg.Count
		if count == 0 {
			count = 1
		}
		ordine := reg.WordOrder
		if ordine == "" {
			ordine = ordineDefault
		}
		parole := Parole(registri, reg.Addr, count)
		if parole == nil {
			g[reg.Name] = nil
			continue
		}
		val, str, isStr, presente := decode.Decodifica(parole, reg.Type, ordine)
		if !presente {
			g[reg.Name] = nil
			continue
		}
		if isStr {
			g[reg.Name] = str
			continue
		}
		scala := 1.0
		if reg.Scale != nil {
			scala = *reg.Scale
		}
		if scala != 1 {
			val = decode.Round4(val * scala)
		}
		g[reg.Name] = val
	}
	return g
}

// Parole raccoglie i `count` registri a partire da addr; nil se ne manca uno.
func Parole(registri map[string]uint16, addr, count int) []uint16 {
	out := make([]uint16, 0, count)
	for i := 0; i < count; i++ {
		w, ok := registri[strconv.Itoa(addr+i)]
		if !ok {
			return nil
		}
		out = append(out, w)
	}
	return out
}

// --------------------------------------------------------------------------- //
//  Utility                                                                    //
// --------------------------------------------------------------------------- //

func Fptr(x float64) *float64 { return &x }
func Sptr(s string) *string   { return &s }

func Round(x float64, decimali int) float64 {
	m := math.Pow(10, float64(decimali))
	return math.Round(x*m) / m
}

func TuttiNil(g map[string]interface{}) bool {
	for _, v := range g {
		if v != nil {
			return false
		}
	}
	return true
}

func AsFloat(v interface{}) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}

// FdaG estrae un campo float dalla mappa {nome: valore} di LeggiMappa; nil se
// assente o non numerico.
func FdaG(g map[string]interface{}, nome string) *float64 {
	if v, ok := g[nome].(float64); ok {
		return &v
	}
	return nil
}

// SdaG estrae un campo stringa non vuoto; nil altrimenti.
func SdaG(g map[string]interface{}, nome string) *string {
	if s, ok := g[nome].(string); ok && s != "" {
		return &s
	}
	return nil
}

// FdaGCandidati ritorna il primo campo float presente tra i nomi candidati.
func FdaGCandidati(g map[string]interface{}, nomi ...string) *float64 {
	for _, n := range nomi {
		if v, ok := g[n].(float64); ok {
			return &v
		}
	}
	return nil
}
