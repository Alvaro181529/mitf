# MITFV2 - Backend en Go para Lectura de Logs vía SSH, Métricas de Hardware, Subsistemas TSM GE CT99 y Bitácora ATREC

Servicio backend en **Go (Golang)** diseñado como API REST pura para conectarse de forma remota mediante **SSH** a servidores **Linux** (con soporte multi-servidor dinámico), monitorear los indicadores de vida útil de hardware (Tubo de Rayos X, Gantry y Rotor), evaluar alertas en los **5 Subsistemas Físicos TSM**, parsear logs propietarios del sistema **GE CT99** (`gesys_ct99.log`), bus CAN (`jedi_can_at_error.log`), errores de DAS (`dataacq.stderr.log`), exponer el **Esquema Interactivo de Fallas (Mapa de Salud de Componentes)**, gestionar la **Bitácora de Mantenimiento ATREC (Atención Técnica y Registro de Calibración)** con control estricto de precondiciones de calidad post-reparación y gestionar la configuración de conexión de servidores en base de datos **PostgreSQL** con endpoints CRUD completos.

---

## 📌 1. Soporte Multi-Servidor Dinámico (`?server_id=<id>`)

Si agregas más servidores en PostgreSQL a través de los endpoints CRUD, puedes consultar cualquier endpoint operacional para un servidor específico simplemente agregando el parámetro **`?server_id=<id>`** a la URL:
- Si omites `server_id`, la API consultará el servidor marcado como **activo** (`is_active = true`) en la base de datos o en memoria.
- Si especificas `?server_id=2`, la API conectará por SSH a ese servidor específico (con su propio host, credenciales y ruta `remote_log_path`) y retornará el bloque de respuesta con metadatos `server_info` identificando qué máquina fue consultada.

---

## 📋 2. Módulo ATREC: Gestión de Bitácora y Calibración Post-Reparación

El módulo **ATREC** (*Atención Técnica y Registro de Calibración*) administra el ciclo de vida de los tickets de soporte técnico del tomógrafo y previene el cierre prematuro de incidentes:

### Estados del Ciclo de Vida:
- `OPEN`: Creado automáticamente por error crítico o manualmente por el técnico.
- `IN_PROGRESS`: Técnico asignado, reparación física o reemplazo en ejecución.
- `PENDING_CALIBRATION`: Reparación física concluida, pendiente de pruebas de calidad.
- `QA_VERIFIED`: Pruebas de calibración ejecutadas y aprobadas en su totalidad.
- `CLOSED`: Ticket cerrado y equipo reincorporado al servicio clínico.

### 🛡️ Regla de Negocio Crítica (Validación ATREC):
Al intentar cerrar un ticket (`PATCH /api/v1/atrec/tickets/:id/close`), el backend valida obligatoriamente que las 3 pruebas de `PostRepairQA` sean verdaderas (`true`):
1. **`ZAlignmentPassed`**: Alineación geométrica en eje Z.
2. **`PhantomIQPassed`**: Calidad de imagen con fantoma (ruido, resolución espacial y números CT/HU).
3. **`DetectorFlatfieldCalibrated`**: Calibración de Offset / Gain ADF de los detectores.

- Si falta cualquiera de estas pruebas, el backend **rechaza la petición con código HTTP 412 (Precondition Failed)** y enumera las pruebas pendientes.
- Si todas están aprobadas, responde con **200 OK** y actualiza el estado a `CLOSED`.

---

## 🗺️ 3. Módulo: Esquema Interactivo de Fallas (Mapa de Salud de Componentes)

Endpoint que expone el estado de salud, colores dinámicos y métricas operativas de cada nodo/subcomponente físico del tomógrafo para renderizar interfaces visuales interactivas:
- **`node_tube_xray`** (TSM 1): Tubo de Rayos X (Performix HD).
- **`node_gantry_rotor`** (TSM 2): Rotor y Anillos Deslizantes.
- **`node_couch_table`** (TSM 3): Mesa del Paciente (Couch Assembly).
- **`node_das_detector`** (TSM 4): Adquisición y Detectores (DAS).
- **`node_estop_console`** (TSM 5): Consola y Paros de Emergencia (E-STOP).

### Códigos de Colores Dinámicos:
- `OK` (`#10B981` / Verde): Sin alertas, funcionamiento normal.
- `WARNING` (`#F59E0B` / Amarillo): Desgaste elevado o desviaciones operativas.
- `CRITICAL` (`#EF4444` / Rojo): Falla crítica, sobrecalentamiento extremo o paro de emergencia activo.

---

## 📊 4. Métricas de Hardware y Límites de Vida Útil

1. **Tubo de Rayos X:**
   - Acumulación de mAs: `accumulated_mas`
   - Límite Nominal EOL: `100,000,000 mAs`
   - Porcentaje de uso consumido (`usage_percentage` ej: `173.52%`)
   - Temperatura Filamento / Ánodo (`anode_temperature_celsius` ej: `471.56 °C`)
   - Capacidad Térmica del Ánodo (`anode_thermal_capacity_mhu` ej: `1.36 MHU`)
   - Estado térmico: `"NORMAL"`, `"WARNING"` o `"CRITICAL"`
2. **Gantry y Rotor:**
   - Revoluciones Acumuladas: `accumulated_revolutions`
   - Límite Recomendado: `100,000 Revs`
   - Porcentaje de uso consumido (`usage_percentage` ej: `841.21%`)
   - Estado cinemático: `"OPERATIONAL"`, `"WARNING"` o `"CRITICAL"`

---

## 🏥 5. Subsistemas Físicos TSM y Mapeo de Logs (`gesys_ct99.log`)

| ID | Subsistema Físico TSM | Procesos / Hosts en `gesys_ct99.log` | Eventos y Alertas Críticas |
| :-: | :--- | :--- | :--- |
| **1** | **Generador y Tubo RX** | `tubemgr`, `hvg`, `generator`, `xray` | Arcos en el tubo, fallos de filamento, sobrecalentamiento de ánodo, fallos HVG. |
| **2** | **Gantry y Rotor** | `rotmgr`, `gantry`, `rotor`, `drive` | Desviación de velocidad del rotor, fallos de freno, errores en anillos deslizantes (*slip ring*). |
| **3** | **Mesa del Paciente** | `tablemgr`, `tgp`, `couch` | Bloqueos H/V, colisiones mecánicas, fallos en sensores de posición (*encoders*). |
| **4** | **Adquisición y Detectores (DAS)** | `dasmgr`, `das`, `pdu` | Canales fuera de tolerancia, sobrecalentamiento de matriz DAS, pérdidas en fibra óptica. |
| **5** | **Consola y Emergencia** | `gscb`, `scanmgr` | Presión del botón de paro (*E-STOP / STOP SCAN*), desconexión de consola. |

---

## 🗄️ 6. Persistencia en PostgreSQL y CRUD de Configuración

Las credenciales SSH y rutas de logs se almacenan en la tabla `server_configs` de PostgreSQL:
- Auto-migración al iniciar (`CREATE TABLE IF NOT EXISTS server_configs`).
- Auto-sembrado (seed) automático con los valores iniciales de `.env` si la tabla está vacía.
- Conmutación en caliente (*hot-reload*): activar una configuración cambia inmediatamente el servidor por defecto sin reiniciar el backend.

### Endpoints CRUD de Servidores:
- `GET /api/v1/configs` : Listar todas las configuraciones almacenadas.
- `GET /api/v1/configs/active` : Obtener la configuración activa actual.
- `GET /api/v1/configs/:id` : Obtener una configuración por su ID.
- `POST /api/v1/configs` : Crear una nueva configuración de servidor SSH.
- `PUT /api/v1/configs/:id` : Actualizar una configuración existente.
- `DELETE /api/v1/configs/:id` : Eliminar una configuración.
- `POST /api/v1/configs/:id/activate` : Activar una configuración específica y conmutar la conexión SSH en caliente.

---

## 🌐 7. Endpoints de la API REST

### A. Diagnóstico y Conectividad
- `GET /api/health?server_id=1` : Estado de salud y parámetros del servidor.
- `GET /api/ssh/test?server_id=1` : Test en vivo de conexión SSH (`uname -a && uptime`).
- `GET /api/ls?server_id=1&path=<subpath>` : Explorador de directorios del servidor remoto.

### B. Módulo de Métricas e Indicadores de Hardware (KPIs)
- `GET /api/v1/hardware/interactive-map?server_id=1` : Esquema Interactivo de Fallas (Mapa de Salud de los 5 Nodos Físicos).
- `GET /api/v1/hardware/summary?server_id=1` : Resumen completo de salud (Tubo RX, Gantry y los 5 Subsistemas TSM).
- `GET /api/v1/hardware/tube-health?server_id=1` : Métricas y porcentaje consumido del tubo de rayos X.
- `GET /api/v1/hardware/gantry-stats?server_id=1` : Estadísticas de rotación y vida útil del rotor.

### C. Módulo ATREC (Bitácora de Mantenimiento y Calibración)
- `GET /api/v1/atrec/tickets?status=OPEN` : Listar tickets (con filtro opcional por estado).
- `POST /api/v1/atrec/tickets` : Crear un nuevo ticket de falla técnica.
- `GET /api/v1/atrec/tickets/:id` : Obtener detalles de un ticket.
- `PUT /api/v1/atrec/tickets/:id/calibration` : Registrar pruebas de calibración y QA.
- `PATCH /api/v1/atrec/tickets/:id/close` : Cerrar ticket (aplica validación HTTP 412).

### D. Módulo de Alertas Filtradas por Subsistema TSM
- `GET /api/v1/alerts/events?server_id=1&severity=2&tsm_subsystem=1&limit=50` : Alertas filtradas por severidad y subsistema TSM.
- `GET /api/v1/alerts/tsm/:subsystem_id?server_id=1` : Alertas del subsistema solicitado (ej: `/api/v1/alerts/tsm/1`).

### E. Módulo de Telemetría Específica
- `GET /api/v1/logs/jedi-can?server_id=1&lines=100` : Eventos del bus CAN (`jedi_can_at_error.log`).
- `GET /api/v1/logs/das-errors?server_id=1&lines=100` : Errores de detectores y fibra óptica (`dataacq.stderr.log`).

### F. Módulo de Logs Generales con Filtros y Paginación
- `GET /api/logs?server_id=1&severity=2&host=tgp&search=STOP%20SCAN&page=1&limit=20` : Logs parseados con búsqueda y paginación.
- `POST /api/parser/ct99` : Parseo de logs desde payload de texto crudo.

---

## 🚀 Ejecución y Pruebas

```bash
cd /home/alvaro/Documents/dev/Ing_Med/MITFV2/backend

# Iniciar servidor
go run main.go

# Ejecutar tests unitarios e integraciones
go test -v ./...
```
