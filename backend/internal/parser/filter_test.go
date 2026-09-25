package parser

import (
	"net/http/httptest"
	"testing"
)

func TestParseFilterOptions(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/logs?severity=3&host=tgp&search=STOP%20SCAN&device=X-Ray&date=2026-09-03&page=2&limit=15", nil)
	opts := ParseFilterOptions(req)

	if opts.Severity != "3" {
		t.Errorf("Severity = %q, want '3'", opts.Severity)
	}
	if opts.Host != "tgp" {
		t.Errorf("Host = %q, want 'tgp'", opts.Host)
	}
	if opts.Search != "STOP SCAN" {
		t.Errorf("Search = %q, want 'STOP SCAN'", opts.Search)
	}
	if opts.Device != "X-Ray" {
		t.Errorf("Device = %q, want 'X-Ray'", opts.Device)
	}
	if opts.Date != "2026-09-03" {
		t.Errorf("Date = %q, want '2026-09-03'", opts.Date)
	}
	if opts.Page != 2 {
		t.Errorf("Page = %d, want 2", opts.Page)
	}
	if opts.Limit != 15 {
		t.Errorf("Limit = %d, want 15", opts.Limit)
	}
}

func TestApplyFiltersAndPagination(t *testing.T) {
	records := []GELogRecord{
		{
			SRID:         "SR 409",
			Date:         "Thu Sep 3 16:19:13 2026",
			Timestamp:    "2026-09-03 16:19:13",
			TimestampRaw: "Thu Sep 3 16:19:13 2026",
			ErmesCode:    "260114056",
			Host:         "Table Gantry Processor",
			Process:      "tgp",
			Severity:     "Pri/Soft",
			SeverityCode: 2,
			Device:       "Gantry Assembly",
			SourceFile:   "tgp_ctrl.cxx:112",
			Message:      "[STOP SCAN] pushbutton on Console Push Button (GSCB) was pressed.",
		},
		{
			SRID:         "SR 410",
			Date:         "Thu Sep 3 16:25:00 2026",
			Timestamp:    "2026-09-03 16:25:00",
			TimestampRaw: "Thu Sep 3 16:25:00 2026",
			ErmesCode:    "260199001",
			Host:         "Tube Manager subsystem",
			Process:      "tubemgr",
			Severity:     "Info",
			SeverityCode: 1,
			Device:       "X-Ray Tube Assembly",
			SourceFile:   "tube_diag.cxx:512",
			Message:      "Tube thermal check completed successfully.",
		},
		{
			SRID:         "SR 411",
			Date:         "Fri Sep 4 10:00:00 2026",
			Timestamp:    "2026-09-04 10:00:00",
			TimestampRaw: "Fri Sep 4 10:00:00 2026",
			ErmesCode:    "260199003",
			Host:         "Table Gantry Processor",
			Process:      "tgp",
			Severity:     "Pri/Hard",
			SeverityCode: 3,
			Device:       "High Voltage Generator",
			SourceFile:   "clutch_mgr.cxx:90",
			Message:      "Clutch slip detected during rotation.",
		},
	}

	// 1. Filtrar por Search "STOP SCAN"
	res := ApplyFilters(records, FilterOptions{Search: "STOP SCAN"})
	if len(res) != 1 || res[0].SRID != "SR 409" {
		t.Fatalf("Search STOP SCAN falló: obtenidos %d", len(res))
	}

	// 2. Filtrar por ErmesCode en Search "260114056"
	res = ApplyFilters(records, FilterOptions{Search: "260114056"})
	if len(res) != 1 || res[0].ErmesCode != "260114056" {
		t.Fatalf("Search por ErmesCode falló")
	}

	// 3. Filtrar por archivo .cxx en Search "clutch"
	res = ApplyFilters(records, FilterOptions{Search: "clutch"})
	if len(res) != 1 || res[0].SRID != "SR 411" {
		t.Fatalf("Search por .cxx falló")
	}

	// 4. Filtrar por Severity "3" o "Pri/Hard"
	res = ApplyFilters(records, FilterOptions{Severity: "3"})
	if len(res) != 1 || res[0].SRID != "SR 411" {
		t.Fatalf("Filtro por Severity '3' falló")
	}

	res = ApplyFilters(records, FilterOptions{Severity: "pri/soft"})
	if len(res) != 1 || res[0].SRID != "SR 409" {
		t.Fatalf("Filtro por Severity 'pri/soft' falló")
	}

	// 5. Filtrar por Host "tgp"
	res = ApplyFilters(records, FilterOptions{Host: "tgp"})
	if len(res) != 2 {
		t.Fatalf("Filtro por Host 'tgp' esperaba 2 registros, obtenidos %d", len(res))
	}

	// 6. Filtrar por Device "X-Ray Tube"
	res = ApplyFilters(records, FilterOptions{Device: "X-Ray Tube"})
	if len(res) != 1 || res[0].SRID != "SR 410" {
		t.Fatalf("Filtro por Device falló")
	}

	// 7. Filtrar por Fecha "2026-09-03"
	res = ApplyFilters(records, FilterOptions{Date: "2026-09-03"})
	if len(res) != 2 {
		t.Fatalf("Filtro por Date esperaba 2 registros, obtenidos %d", len(res))
	}

	// 8. Paginación
	paged, meta := PaginateRecords(records, 1, 2)
	if len(paged) != 2 || meta.TotalRecords != 3 || meta.TotalPages != 2 || !meta.HasNext || meta.HasPrev {
		t.Fatalf("Paginación página 1 falló: %+v", meta)
	}

	paged2, meta2 := PaginateRecords(records, 2, 2)
	if len(paged2) != 1 || meta2.HasNext || !meta2.HasPrev {
		t.Fatalf("Paginación página 2 falló: %+v", meta2)
	}
}
