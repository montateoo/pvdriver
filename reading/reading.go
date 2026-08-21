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

	PowerAC              *float64 `json:"power_ac"`
	PowerDC              *float64 `json:"power_dc"`
	VoltageDC            *float64 `json:"voltage_dc"`
	CurrentDC            *float64 `json:"current_dc"`
	VoltageAC            *float64 `json:"voltage_ac"`
	CurrentAC            *float64 `json:"current_ac"`
	FrequencyHz          *float64 `json:"frequency_hz"`
	ReactivePowerVAR     *float64 `json:"reactive_power_var"`
	ApparentPowerVA      *float64 `json:"apparent_power_va"`
	PowerFactor          *float64 `json:"power_factor"`
	Temperature          *float64 `json:"temperature"`
	EnergyAC             *float64 `json:"energy_ac"`
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
