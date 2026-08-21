package driver

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/corna/pvdriver/comune"
	"github.com/corna/pvdriver/reading"
)

// Driver nativo: decodifica in-process, senza wasm. La logica di decodifica e'
// la stessa dei vecchi guest wasm (pacchetto comune, Go puro), qui chiamata
// direttamente. Questo sblocca i protocolli che la ABI wasm non gestiva:
// seriale (Fronius) e, in futuro, MQTT.
type Driver interface {
	Meta() Meta
	Decode(in Input) (reading.Metric, error)
}

type Meta struct {
	Capability   string
	Vendor       string
	Protocol     string // modbus-tcp | rest-api | serial-fronius | mqtt-subscribe | yasdi-csv
	RestPath     string
	NeedsSession bool
	Blocks       []Block

	// MQTTTopic/MQTTWindow: solo per Protocol "mqtt-subscribe" (vedi il
	// package selta del repo pvdriver-vendor). MQTTWindow<=0 usa il default
	// del reader (20s).
	MQTTTopic  string
	MQTTWindow time.Duration

	// Validato: affermazione di chi ha scritto il driver su cosa e' stato
	// osservato su hardware reale, non derivabile dal codice -- per questo va
	// impostato a mano, non calcolato. "" = non verificato/non specificato,
	// "validato" = confermato su almeno un impianto reale (vedi
	// NotaValidazione per cosa esattamente), "non_validato" = costruito e
	// testato solo con dati sintetici, nessun impianto reale lo ha ancora
	// esercitato.
	Validato        string
	NotaValidazione string
}

type Block struct {
	Target       string
	Unit         int
	FromMap      string
	Ranges       [][]int
	FunctionCode int
}

// Input raccoglie tutto cio' che un driver puo' consumare. Il reader riempie
// solo il campo pertinente al protocollo: Registers per Modbus, Rest per REST,
// Serial per il bus seriale Fronius, MQTT per mqtt-subscribe, Text/FileModTime
// per yasdi-csv.
type Input struct {
	Unit      int
	Map       json.RawMessage
	Registers map[string]uint16
	Extra     map[string]map[string]uint16
	Rest      map[string]any
	Serial    SerialFunc

	// MQTT: ultimo payload visto per topic durante la finestra di ascolto.
	MQTT map[string][]byte

	// Text/FileModTime: contenuto grezzo e mtime di un file letto localmente
	// (yasdi-csv). FileModTime serve al driver per giudicare la freschezza
	// del dato: MeasuredAt e' sempre "adesso", non riflette da quando il
	// demone che scrive il file e' fermo.
	Text        string
	FileModTime time.Time
}

// SerialFunc esegue una transazione sul bus seriale: invia frame e ritorna la
// risposta grezza, uscendo appena trova `atteso` nel buffer (early-exit contro
// la contesa sul bus). La fornisce il reader (trasporto seriale, solo sul Pi).
type SerialFunc func(frame, atteso []byte) []byte

var registry = map[string]Driver{}

// Register aggiunge un driver al registro globale, indicizzato per
// capacita'. Le implementazioni vivono nel repo pvdriver-vendor (package
// huawei, sma, ...) e si registrano da un init(); vedi il package all di
// quel repo per il blank-import che le carica tutte.
func Register(d Driver) { registry[d.Meta().Capability] = d }

// Get ritorna il driver per una capacita', o (nil, false) se non c'e'.
func Get(capability string) (Driver, bool) {
	d, ok := registry[capability]
	return d, ok
}

// All ritorna i Meta di ogni driver registrato, ordinati per capacita'.
// Serve al generatore del manifest (vedi cmd/manifest in pvdriver-vendor):
// i percorsi di lettura normali usano sempre Get, mai All.
func All() []Meta {
	out := make([]Meta, 0, len(registry))
	for _, d := range registry {
		out = append(out, d.Meta())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Capability < out[j].Capability })
	return out
}

// Ingresso adatta Input al tipo che comune sa decodificare. Helper condiviso
// usato dai driver nei sottopackage per vendor.
func Ingresso(in Input) *comune.Ingresso {
	var mr comune.MappaReg
	if len(in.Map) > 0 {
		_ = json.Unmarshal(in.Map, &mr)
	}
	return &comune.Ingresso{
		Unit:          in.Unit,
		Mappa:         mr,
		Registri:      in.Registers,
		RegistriExtra: in.Extra,
		Rest:          in.Rest,
	}
}

// ConversionEfficiency calcola l'efficienza DC->AC (%) solo se la DC e'
// sopra soglia (100 W): sotto soglia il rapporto e' rumore, non un dato.
func ConversionEfficiency(powerAC, powerDC *float64) *float64 {
	if powerDC == nil || powerAC == nil || *powerDC <= 100 {
		return nil
	}
	v := comune.Round(*powerAC/(*powerDC)*100.0, 2)
	return &v
}
