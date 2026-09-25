package database

import (
	"testing"

	"mitfv2/config"
)

func init() {
	config.LoadConfig()
}

func TestDBRepositoryCRUD(t *testing.T) {
	repo, err := InitDB()
	if err != nil {
		t.Skipf("Omitiendo prueba de BD si PostgreSQL no está disponible: %v", err)
	}

	// 1. Obtener todas las configuraciones
	all, err := repo.GetAll()
	if err != nil {
		t.Fatalf("Error en GetAll: %v", err)
	}
	if len(all) == 0 {
		t.Fatalf("Se esperaba al menos una configuración inicial sembrada, obtenido 0")
	}

	// 2. Obtener la activa
	active, err := repo.GetActive()
	if err != nil {
		t.Fatalf("Error en GetActive: %v", err)
	}
	if active == nil {
		t.Fatalf("Se esperaba una configuración activa")
	}
	if active.SSHHost == "" {
		t.Errorf("SSHHost no debería estar vacío")
	}

	// 3. Crear una nueva configuración de prueba
	testCfg := &ServerConfig{
		Name:          "Test Server QA",
		SSHHost:       "192.168.122.80",
		SSHPort:       2222,
		SSHUser:       "qa_user",
		SSHPassword:   "qa_pass",
		RemoteLogPath: "/var/log/qa",
		ServerPort:    ":8081",
		IsActive:      false,
	}
	created, err := repo.Create(testCfg)
	if err != nil {
		t.Fatalf("Error creando configuración: %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("ID debe ser positivo, obtenido %d", created.ID)
	}

	// 4. Buscar por ID
	byID, err := repo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("Error en GetByID: %v", err)
	}
	if byID.Name != "Test Server QA" {
		t.Errorf("Nombre esperado 'Test Server QA', obtenido %q", byID.Name)
	}

	// 5. Actualizar
	byID.Name = "Test Server Updated"
	updated, err := repo.Update(created.ID, byID)
	if err != nil {
		t.Fatalf("Error en Update: %v", err)
	}
	if updated.Name != "Test Server Updated" {
		t.Errorf("Nombre actualizado esperado 'Test Server Updated', obtenido %q", updated.Name)
	}

	// 6. SetActive
	activated, err := repo.SetActive(created.ID)
	if err != nil {
		t.Fatalf("Error en SetActive: %v", err)
	}
	if !activated.IsActive {
		t.Errorf("Debería estar marcada como activa")
	}

	// Restaurar la activa original
	_, _ = repo.SetActive(active.ID)

	// 7. Eliminar la de prueba
	if err := repo.Delete(created.ID); err != nil {
		t.Fatalf("Error en Delete: %v", err)
	}

	// Verificar eliminación
	_, err = repo.GetByID(created.ID)
	if err == nil {
		t.Errorf("Se esperaba error al buscar ID eliminado")
	}
}
