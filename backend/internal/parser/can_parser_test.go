package parser

import (
	"testing"
)

func TestParseCANServerPayload(t *testing.T) {
	rawJSON := `{
  "content": " 5032265 T : 478h (0) \n 5032130 R : 479h (1) 00h \n 5030265 T : 478h (0) \n-----------------------\n 4940280 R : 208h (8) 01h 02h 03h 04h 05h 06h 07h 08h \n\n",
  "file": "jedi_can_at_error.log",
  "lines": 100,
  "server_info": {
    "host": "192.168.122.79",
    "port": 22,
    "user": "server"
  },
  "status": "success",
  "timestamp": "2026-09-25T01:57:47Z"
}`

	res, err := ParseCANServerPayload([]byte(rawJSON))
	if err != nil {
		t.Fatalf("Error inesperado en ParseCANServerPayload: %v", err)
	}

	if res.File != "jedi_can_at_error.log" {
		t.Errorf("File esperado 'jedi_can_at_error.log', obtenido %q", res.File)
	}

	if res.SubsystemID != 1 {
		t.Errorf("SubsystemID esperado 1, obtenido %d", res.SubsystemID)
	}

	if len(res.Frames) != 4 {
		t.Fatalf("Se esperaban 4 tramas válidas, obtenidas %d", len(res.Frames))
	}

	// Trama 0: 5032265 T : 478h (0)
	f0 := res.Frames[0]
	if f0.TimestampMS != 5032265 || f0.Direction != "T" || f0.CANID != "478h" || f0.DLC != 0 || !f0.IsHeartbeat {
		t.Errorf("Trama 0 incorrecta: %+v", f0)
	}
	if len(f0.Payload) != 0 {
		t.Errorf("Trama 0 payload esperado vacío, obtenido %+v", f0.Payload)
	}
	if f0.SubsystemID != 1 {
		t.Errorf("Trama 0 SubsystemID esperado 1, obtenido %d", f0.SubsystemID)
	}

	// Trama 1: 5032130 R : 479h (1) 00h
	f1 := res.Frames[1]
	if f1.TimestampMS != 5032130 || f1.Direction != "R" || f1.CANID != "479h" || f1.DLC != 1 || !f1.IsHeartbeat {
		t.Errorf("Trama 1 incorrecta: %+v", f1)
	}
	if len(f1.Payload) != 1 || f1.Payload[0] != "00h" {
		t.Errorf("Trama 1 payload esperado ['00h'], obtenido %+v", f1.Payload)
	}

	// Trama 3: 4940280 R : 208h (8) 01h 02h 03h 04h 05h 06h 07h 08h
	f3 := res.Frames[3]
	if f3.TimestampMS != 4940280 || f3.Direction != "R" || f3.CANID != "208h" || f3.DLC != 8 || f3.IsHeartbeat {
		t.Errorf("Trama 3 incorrecta: %+v", f3)
	}
	if len(f3.Payload) != 8 || f3.Payload[0] != "01h" || f3.Payload[7] != "08h" {
		t.Errorf("Trama 3 payload incorrecto: %+v", f3.Payload)
	}
}

func TestParseCANContentEmptyOrSeparators(t *testing.T) {
	content := "-----------------------\n\n   \n===================\n"
	frames := ParseCANContent(content)
	if len(frames) != 0 {
		t.Errorf("Se esperaba 0 tramas para separadores, obtenido %d", len(frames))
	}
}
