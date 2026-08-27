package reading

import "time"

type Outcome string

const (
	OK              Outcome = "ok"
	Partial         Outcome = "partial"
	Timeout         Outcome = "timeout"
	ProtocolError   Outcome = "protocol_error"
	ConnectionError Outcome = "connection_error"
	StaleData       Outcome = "stale_data"
)

type Identity struct {
	Vendor       string  `json:"vendor"`
	Model        *string `json:"model"`
	SerialNumber *string `json:"serial_number"`
	Firmware     *string `json:"firmware,omitempty"`
}

type MPPT struct {
	Index     int      `json:"index"`
	CurrentDC *float64 `json:"current_dc"`
	VoltageDC *float64 `json:"voltage_dc"`
	PowerDC   *float64 `json:"power_dc"`
}

type Device struct {
	Plant string `json:"plant"`
	ID    string `json:"id"`
	Unit  *int   `json:"unit"`
	Host  string `json:"host"`
}

type Metric struct {
	Device     Device    `json:"device"`
	MeasuredAt time.Time `json:"measured_at"`

	// Le grandezze di potenza -- PowerAC, PowerDC, ReactivePowerVAR,
	// ApparentPowerVA -- sono SEMPRE in unita' di base: W, VAR, VA. Mai in
	// kW/kVAR/kVA, anche quando la mappa registri del dispositivo produce
	// direttamente i kilo- (scale 0.001): in quel caso il driver riporta il
	// valore in unita' di base, come fanno sma/webbox, huawei e sigenergy.
	//
	// Il motivo non e' estetico: il consumatore converte. L'agente Helios
	// divide per 1000 in infra/http/helios/ingest.kilo() subito prima di
	// spedire, perche' la sua API vuole kW sul filo. Un driver che consegna
	// gia' kW produce quindi valori 1000 volte piu' piccoli a database, e
	// non se ne accorge nessuno: nessun allarme scatta, il grafico resta
	// della forma giusta e solo la scala e' sbagliata.
	//
	// Tensioni, correnti, frequenza, temperatura e fattore di potenza
	// passano invece grezzi (V, A, Hz, gradi C, adimensionale).
	PowerAC          *float64 `json:"power_ac"`
	PowerDC          *float64 `json:"power_dc"`
	VoltageDC        *float64 `json:"voltage_dc"`
	CurrentDC        *float64 `json:"current_dc"`
	VoltageAC        *float64 `json:"voltage_ac"`
	CurrentAC        *float64 `json:"current_ac"`
	FrequencyHz      *float64 `json:"frequency_hz"`
	ReactivePowerVAR *float64 `json:"reactive_power_var"`
	ApparentPowerVA  *float64 `json:"apparent_power_va"`
	PowerFactor      *float64 `json:"power_factor"`
	Temperature      *float64 `json:"temperature"`
	// EnergyAC is energy produced during THIS reading's interval (kWh), not
	// a running total. Only set it if the device itself reports a
	// per-interval or per-poll delta -- Decode has no memory between polls,
	// so it cannot compute one from a cumulative counter. Most drivers
	// leave this nil and rely on EnergyACCumulative instead: the server
	// derives interval energy from consecutive cumulative readings, and
	// that delta is preferred over EnergyAC whenever both are present (see
	// ingest/rollup/inverter.go). Putting a daily/cumulative counter here
	// by mistake inflates the interval sum by however many times that
	// bucket samples it.
	EnergyAC *float64 `json:"energy_ac"`
	// EnergyACCumulative is the device's lifetime (or at least
	// long-running, monotonic-within-a-day) energy counter, e.g. an
	// inverter's "total energy" register -- NOT the daily counter that
	// resets at midnight. The server computes interval energy as the delta
	// between consecutive readings of this field.
	//
	// UNIT: kWh. This is the one field where getting the unit wrong is
	// invisible in code review and catastrophic at the DB: ingest converts
	// only the POWER fields to kilo (see kilo() in the agent's ingest
	// client) and takes energies as-is, so a driver emitting Wh here lands
	// values 1000x too large in a kWh column. It happened: AROS on site
	// solar_db_anna, 2026-08-26, caught only by comparing cumulative
	// energies across sites (a 10 kW inverter cannot out-produce a 432 kW
	// one by 15x).
	EnergyACCumulative   *float64 `json:"energy_ac_cumulative"`
	ConversionEfficiency *float64 `json:"conversion_efficiency"`
	OperatingState       *string  `json:"operating_state"`
	FaultCode            *string  `json:"fault_code"`

	MPPT     []MPPT         `json:"mppt"`
	Identity *Identity      `json:"identity"`
	Raw      map[string]any `json:"raw,omitempty"`

	Outcome   Outcome  `json:"outcome"`
	Error     string   `json:"error,omitempty"`
	Source    string   `json:"source"`
	Verdict   string   `json:"verdict,omitempty"`
	Anomalies []string `json:"anomalies,omitempty"`
}

func (m Metric) Usable() bool {
	return m.Outcome == OK || m.Outcome == Partial
}

type PlantReading struct {
	Plant     string    `json:"plant"`
	Name      string    `json:"name,omitempty"`
	Latitude  *float64  `json:"latitude,omitempty"`
	Longitude *float64  `json:"longitude,omitempty"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Metrics   []Metric  `json:"metrics"`
}

func (r PlantReading) Failed() int {
	n := 0
	for _, m := range r.Metrics {
		if !m.Usable() {
			n++
		}
	}
	return n
}
