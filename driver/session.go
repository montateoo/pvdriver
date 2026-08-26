package driver

// Aggiunta puramente additiva al contratto: nessuna firma esistente cambia,
// nessun driver esistente va toccato. Serve perche' alcuni bus di campo
// pretendono un handshake UNA VOLTA PER BUS, non per lettura -- e ne'
// Driver.Decode ne' Input possono esprimerlo: Input e' per-dispositivo,
// mentre l'handshake ha bisogno del roster completo del bus.
//
// Il caso che l'ha motivata: gli inverter AROS PVSER non hanno un indirizzo
// di bus. L'id numerico con cui si interrogano glielo ASSEGNA il master, in
// una sequenza che manda a ciascun inverter la propria matricola e il numero
// da assumere. Finche' quella sequenza non e' passata, l'inverter risponde al
// broadcast ma non risponde a nessuna lettura indirizzata: e' il guasto
// osservato in campo il 2026-08-25 (0/11 letture prima, 11/11 dopo).

import "time"

// SessionDevice descrive un dispositivo del bus come lo dichiara la
// configurazione: l'unit che deve assumere e la sua matricola.
type SessionDevice struct {
	Unit         int
	SerialNumber string
}

// SessionInput e' l'ingresso di OpenSession: la funzione di transazione sul
// bus e il roster COMPLETO dei dispositivi di quel bus, nell'ordine di
// configurazione.
//
// Repair, se non vuoto, limita la sequenza a quelle unit: serve a rimediare
// al singolo dispositivo che si e' perso (tipicamente un inverter che si e'
// riacceso dopo gli altri) senza rifare l'apertura del bus e senza disturbare
// chi sta gia' rispondendo.
type SessionInput struct {
	Serial  SerialFunc
	Devices []SessionDevice
	Repair  []int
}

// SessionDeviceResult e' l'esito per singolo dispositivo. Nameplate porta la
// targa che il dispositivo dichiara, quando la manda: e' il modo piu' solido
// di verificare che l'id sia finito sull'hardware giusto, perche' contiene la
// matricola.
type SessionDeviceResult struct {
	Unit         int
	SerialNumber string
	Registered   bool
	Nameplate    string
	Note         string
}

// SessionReport e' l'esito dell'handshake.
//
// BusAlive e' il campo che conta per il chiamante: distingue "bus muto"
// (notte, inverter spenti, cavo staccato: non c'e' niente da inizializzare e
// insistere e' solo tempo buttato) da "dispositivi vivi ma non registrati",
// l'unico caso in cui vale la pena spendere la sequenza intera. Chi implementa
// OpenSession deve poterlo dire dopo pochi frame, non dopo la sequenza
// completa.
type SessionReport struct {
	BusAlive    bool
	Devices     []SessionDeviceResult
	SerialsSeen []string
	Frames      int
	Duration    time.Duration
	Err         string

	// ResetNeeded: il bus ha risposto solo dopo che il driver ha mandato il
	// proprio comando di reset. Distingue due diagnosi che al chiamante
	// interessano in modo diverso: "i dispositivi erano senza indirizzo"
	// (normale dopo uno spegnimento) da "avevano un indirizzo ma non
	// rispondevano piu' alle letture" (qualcosa e' andato storto prima).
	ResetNeeded bool
}

// SessionOpener e' implementata dai driver che hanno bisogno di un handshake
// una volta per bus. La divisione del lavoro e' deliberata: i BYTE stanno nel
// driver (e' l'unico che conosce il protocollo), il CICLO DI VITA sta nel
// chiamante (e' l'unico che sa quando si apre una porta, quali dispositivi ci
// sono e quando le letture hanno smesso di funzionare).
//
// Interfaccia OPZIONALE: si risolve con una type assertion
//
//	if so, ok := drv.(driver.SessionOpener); ok { ... }
//
// quindi i driver che non la implementano continuano a funzionare senza
// modifiche, e un chiamante che non la conosce continua a compilare.
type SessionOpener interface {
	Driver
	OpenSession(in SessionInput) SessionReport
}
