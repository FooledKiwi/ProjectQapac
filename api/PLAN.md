# Plan de Implementación: MVP Transporte Público - Tareas Delegadas

---

## Estructura de Worktrees y Branches

```
qapac-api-workspace/
├── qapac-api/              → main worktree, rama main
├── wt-bootstrap/           → rama bootstrap (YO)
├── wt-db-model/            → rama db-model (YO)
├── wt-storage/             → rama storage (YO)
├── wt-routing/             → rama routing (YO)
├── wt-eta/                 → rama eta (TÚ)
├── wt-api/                 → rama api (TÚ)
├── wt-demo/                → rama demo (ambos)
├── wt-auth/                → rama auth (MVP v2-B: JWT)
└── wt-tracking/            → rama tracking (MVP v2-A: GPS + ETA real)
```

---

## ETAPA 1: Bootstrap del Proyecto (YO)
**Worktree:** `wt-bootstrap` | **Branch:** `bootstrap`

### Tareas:
1. Restructurar: mover `main.go` a `cmd/server/main.go`.
2. Crear directorios vacíos: `internal/{handler, service, storage, routing}`.
3. Crear archivo `internal/config/config.go`:
   - Leer env vars: `DB_DSN`, `GOOGLE_API_KEY`, `PORT` (default 8080).
   - Parsear y validar configuracion.
4. Crear archivo `internal/app/app.go`:
   - Inicializar pool de conexiones a PostGIS.
   - Configurar gin.Engine con middlewares basicos.
5. Actualizar `cmd/server/main.go`:
   - Cargar config.
   - Inicializar app.
   - Servir en `PORT`.
6. Actualizar `go.mod` con dependencias: `github.com/lib/pq` (driver PostGIS), `github.com/jackc/pgx/v5` (para pooling).
7. Actualizar `mise.toml`:
   - Apuntar `api:dev` a `cmd/server/main.go`.
   - Agregar vars de env requeridas si falta.

### Tests:
- Unit: ninguno (solo bootstrapping).
- Integration: validar que la API arranca sin errores con `mise run api:dev`.

### Consideraciones:
- Pooling de conexiones: `pgx.ConnPool` con `MaxConnLifetime: 30s`, `MaxConns: 20`.
- Manejo de errores: custom types para BD y config.
- No agregar rutas aun; solo servidor base.

### Merge a main:
Cuando bootstrap arranca sin errores.

---

## ETAPA 2: Modelo de Datos MVP (YO)
**Worktree:** `wt-db-model` | **Branch:** `db-model`

### Tareas:
1. Crear archivo `internal/migrations/001_initial_schema.sql`:
   - Usar `CREATE TABLE IF NOT EXISTS` y `CREATE INDEX IF NOT EXISTS` para idempotencia.
   - Habilitar extensión PostGIS con `CREATE EXTENSION IF NOT EXISTS postgis`.
   ```sql
   -- Tablas base
   CREATE TABLE IF NOT EXISTS stops (
     id SERIAL PRIMARY KEY,
     name VARCHAR(255) NOT NULL,
     geom GEOMETRY(POINT, 4326) NOT NULL,
     active BOOLEAN DEFAULT true,
     created_at TIMESTAMP DEFAULT NOW()
   );
   CREATE INDEX IF NOT EXISTS idx_stops_geom ON stops USING GIST(geom);
   
   CREATE TABLE IF NOT EXISTS routes (
     id SERIAL PRIMARY KEY,
     name VARCHAR(255) NOT NULL,
     active BOOLEAN DEFAULT true
   );
   
   CREATE TABLE IF NOT EXISTS route_stops (
     id SERIAL PRIMARY KEY,
     route_id INT NOT NULL REFERENCES routes(id),
     stop_id INT NOT NULL REFERENCES stops(id),
     sequence INT NOT NULL,
     UNIQUE(route_id, stop_id)
   );
   
   CREATE TABLE IF NOT EXISTS route_shapes (
     id SERIAL PRIMARY KEY,
     route_id INT NOT NULL REFERENCES routes(id),
     geom GEOMETRY(LINESTRING, 4326) NOT NULL,
     updated_at TIMESTAMP DEFAULT NOW()
   );
   CREATE INDEX IF NOT EXISTS idx_route_shapes_geom ON route_shapes USING GIST(geom);
   
   -- Tablas de cache (unlogged)
   CREATE UNLOGGED TABLE IF NOT EXISTS stop_eta_cache (
     id SERIAL PRIMARY KEY,
     stop_id INT NOT NULL REFERENCES stops(id),
     eta_seconds INT NOT NULL,
     calc_ts TIMESTAMP DEFAULT NOW(),
     expires_at TIMESTAMP NOT NULL,
     UNIQUE(stop_id)
   );
   CREATE INDEX IF NOT EXISTS idx_eta_cache_stop_id ON stop_eta_cache(stop_id);
   
   CREATE UNLOGGED TABLE IF NOT EXISTS route_to_stop_cache (
     id SERIAL PRIMARY KEY,
     origin_hash VARCHAR(50) NOT NULL,
     stop_id INT NOT NULL REFERENCES stops(id),
     polyline TEXT NOT NULL,
     distance_m INT NOT NULL,
     duration_s INT NOT NULL,
     calc_ts TIMESTAMP DEFAULT NOW(),
     expires_at TIMESTAMP NOT NULL,
     UNIQUE(origin_hash, stop_id)
   );
   CREATE INDEX IF NOT EXISTS idx_route_cache_hash_stop ON route_to_stop_cache(origin_hash, stop_id);
   ```

2. Crear script de seed `internal/migrations/002_seed_data.sql` con fixtures sintéticos (5-10 stops + 1-2 routes en area de demo).
   - Usar `INSERT ... ON CONFLICT DO NOTHING` para idempotencia.

3. Crear tabla de control `schema_migrations` en `internal/migrations/000_migrations_table.sql`:
   ```sql
   CREATE TABLE IF NOT EXISTS schema_migrations (
     version     VARCHAR(255) PRIMARY KEY,
     applied_at  TIMESTAMP DEFAULT NOW()
   );
   ```
   - Esta tabla persiste el estado de las migraciones aplicadas.
   - Debe ser el primer archivo en ejecutarse (prefijo `000_`).

4. Crear runner en `internal/migrations/runner.go`:
   - Embeber los archivos `.sql` con `//go:embed *.sql`.
   - Función `Run(ctx, pool)`: iterar archivos en orden lexicográfico, saltar los que ya
     están registrados en `schema_migrations`, ejecutar el resto en transacción individual,
     registrar cada uno en `schema_migrations` al commitear.
   - Función `CheckSchema(ctx, pool)`: verificar que las 6 tablas de negocio existen.
   - **Nota:** el runner vive en `internal/migrations/` (no en `internal/storage/`) porque
     `//go:embed` no puede referenciar rutas con `../`; `internal/storage/` importará este
     paquete para invocar `migrations.Run`.

5. Crear `internal/storage/migrations.go` como punto de entrada del storage layer:
   - Reexportar o delegar a `migrations.Run` y `migrations.CheckSchema`.
   - Permite que `app.go` importe un único paquete (`storage`) para el arranque.

### Tests:
- SQL validation: script que verifica tablas, columnas, indices (bash/psql).
- Integracion: conectar a BD, asegurar schema existe.
- Idempotencia: ejecutar `Run` dos veces seguidas no debe producir error ni duplicados.

### Consideraciones:
- SRID 4326: WGS84 (lat/lon globales).
- Unlogged tables: datos perdidos en crash, pero rapido (sin WAL).
- PK simples para facilidad.
- `schema_migrations` es una tabla normal (logged) para sobrevivir reinicios.
- El runner NO re-ejecuta migraciones ya registradas; safe para múltiples arranques.

### Merge a main:
Cuando schema pasa validacion.

---

## ETAPA 3: Storage Layer con sqlc (YO)
**Worktree:** `wt-storage` | **Branch:** `storage`
**Estado:** Implementado. Integration tests pendientes (requieren BD real).

### Tareas:
1. Instalar `sqlc`: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`.
   - Versión instalada: `v1.30.0`.

2. Crear archivo `sqlc.yaml`:
   ```yaml
   version: "2"
   sql:
     - engine: "postgresql"
       queries: "internal/storage/queries"
       schema: "internal/migrations"
       gen:
         go:
           out: "internal/generated/db"
           package: "db"
           sql_package: "pgx/v5"   # usar pgx/v5 en lugar de database/sql
   ```
   - **Decisión:** `sql_package: "pgx/v5"` para que el `DBTX` generado use la interfaz
     nativa de pgx (`pgconn.CommandTag`, `pgx.Rows`, `pgx.Row`) y sea compatible con
     `pgxpool.Pool` sin adaptadores.

3. Crear queries en `internal/storage/queries/stops.sql`:
   ```sql
   -- name: FindStopsNear :many
   SELECT id, name, ST_AsText(geom) AS geom
   FROM stops
   WHERE ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326)::geography, sqlc.arg(radius_m)::float8)
     AND active = true
   ORDER BY ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326)::geography);

   -- name: GetStop :one
   SELECT id, name, ST_AsText(geom) AS geom
   FROM stops
   WHERE id = sqlc.arg(id)::int AND active = true;

   -- name: GetRouteShape :one
   SELECT id, route_id, ST_AsText(geom) AS geom
   FROM route_shapes
   WHERE route_id = sqlc.arg(route_id)::int;
   ```
   - **Decisión — `geography` cast:** `geom::geography` hace que `ST_DWithin` y
     `ST_Distance` operen en metros (esférico). Sin el cast, operan en grados, y un
     radio de `1000` significaría ~1000 grados, no 1000 metros.
   - **Decisión — `sqlc.arg()` con cast explícito:** Los parámetros posicionales puros
     (`$1`, `$2`) dentro de funciones PostGIS no permiten a sqlc inferir el tipo Go.
     Usando `sqlc.arg(nombre)::tipo` se obtienen nombres descriptivos y tipos concretos
     (`float64` para coordenadas/radio, `int32` para IDs) en el struct generado.
   - **Nota — parámetros repetidos:** `$1`/`$2` aparecen dos veces en `FindStopsNear`
     (en `WHERE` y en `ORDER BY`). PostgreSQL soporta parámetros repetidos; sqlc los
     mapea correctamente al mismo campo del struct.

4. Generar bindings: `sqlc generate`.
   - Código generado en `internal/generated/db/`: `db.go`, `models.go`, `stops.sql.go`.
   - Los campos `Geom` en los structs generados son `interface{}` porque sqlc no conoce
     los tipos PostGIS. En runtime reciben un `string` WKT (`ST_AsText` lo garantiza).

5. Crear `internal/storage/repository.go`:
   ```go
   type Stop struct {
       ID   int32
       Name string
       Lat  float64
       Lon  float64
   }

   type RouteShape struct {
       ID      int32
       RouteID int32
       GeomWKT string  // WKT del linestring, e.g. "LINESTRING(...)"
   }

   type StopsRepository interface {
       FindStopsNear(ctx context.Context, lat, lon, radiusMeters float64) ([]Stop, error)
       GetStop(ctx context.Context, id int32) (*Stop, error)
   }

   type RoutesRepository interface {
       GetRouteShape(ctx context.Context, routeID int32) (*RouteShape, error)
   }
   ```
   - `GetStop` y `GetRouteShape` devuelven `(nil, nil)` cuando no existe la fila
     (semántica limpia; el handler decide si es 404 o no).

6. Implementar repos en `internal/storage/postgres.go`:
   - Timeout de 5s por query via `context.WithTimeout`. Si el contexto entrante ya tiene
     un deadline más corto, ese gana (comportamiento estándar de Go).
   - Parseo WKT en `parsePointWKT`: `POINT(lon lat)` → `(lat, lon)`. PostGIS serializa
     en orden `(lon lat)` con SRID 4326; la función invierte el orden al devolver.
   - **Chequeo de geom nil:** `rowToStop` y `GetRouteShape` verifican explícitamente
     `geom == nil` antes del type assertion. Si la BD devuelve NULL en la columna geom
     (violación de integridad que el schema previene con `NOT NULL`, pero posible si la
     BD es alterada manualmente), el error incluye el ID afectado y la causa raíz para
     distinguirlo de un bug del storage layer.

### Tests implementados (`internal/storage/postgres_test.go`):
- `TestParsePointWKT`: tabla de 10 casos — coordenadas válidas, whitespace, cero,
  positivos, string vacío, prefijo incorrecto, paréntesis faltante, floats inválidos,
  demasiadas coordenadas.
- `TestRowToStop`: tabla de 4 casos — stop válido, geom de tipo incorrecto,
  **geom nil (NULL en BD)**, WKT inválido.

### Integration tests pendientes (requieren BD real con PostGIS):
- `FindStopsNear(lat, lon, 1000m)` devuelve stops en radio.
- `GetStop(id)` devuelve stop correcto.
- Medir latencia p95 para validar índices GiST.

### Consideraciones:
- `ST_DWithin` con `geography`: distancia en metros (PostGIS).
- Parametrización: sqlc previene SQL injection en las queries generadas.
- Timeout: 5s en queries. Cadena completa: handler 10s → storage 5s → Google API 5s.
- Tablas `UNLOGGED` (`stop_eta_cache`, `route_to_stop_cache`): sqlc las modela igual
  que tablas normales. Sin WAL → datos perdidos en crash de PG, pero escrituras más
  rápidas. Comportamiento esperado e intencional para cache.
- `WithTx` está disponible en el código generado para operaciones de escritura
  transaccionales en etapas futuras.

### Merge a main:
Cuando integration tests pasan.

---

## ETAPA 4: Routing Centralizado con Google Routes API v2 (YO)
**Worktree:** `wt-routing` | **Branch:** `routing`

### Tareas:
1. Crear interfaz `internal/routing/router.go`:
   ```go
   type RoutingRequest struct {
       OriginLat, OriginLon       float64
       DestinationLat, DestinationLon float64
   }
   
   type RoutingResponse struct {
       Polyline  string // encoded polyline
       DistanceM int
       DurationS int
   }
   
   type Router interface {
       Route(ctx context.Context, req RoutingRequest) (*RoutingResponse, error)
   }
   ```

2. Crear cliente de Google Routes API: `internal/routing/google.go`:
   - Usar librería `googlemaps/google-maps-services-go`.
   - Llamar Google Routes API v2 con origen/destino.
   - Parsear respuesta (polyline, distance, duration).
   - Manejo de errores/timeouts (5s máximo).

3. Crear wrapper de cache: `internal/routing/cache.go`:
   - Implementar Router interface.
   - Hash origen (ej: geohash + stop_id).
   - Antes de llamar Google, consultar `route_to_stop_cache`.
   - Si hit y no expirado (`expires_at > NOW()`), devolver cached.
   - Si miss:
     - Llamar `GoogleRouter.Route()`.
     - Insertar en cache con `expires_at = NOW() + 120s`.
     - Devolver respuesta.

4. Crear `internal/service/routing_service.go`:
   - Orquesta `CacheRouter`.
   - Expone metodo `GetRouteTo(ctx, userLat, userLon, stopID)`.

### Tests:
- Unit:
  - Mock de Google API (hardcode respuesta).
  - Valida cache hit/miss.
  - Valida expiry.
- Integration:
  - Contra BD real (tabla cache).
  - Primera llamada: consulta Google.
  - Segunda llamada (< 120s): devuelve cache sin Google.
  - Valida duracion < 1s total (incluyendo DB).

### Consideraciones:
- API key: env var `GOOGLE_API_KEY`, solo en backend.
- Rate limiting: Google tiene limites; cache reduce carga.
- Polyline: formato codificado de Google; devolver como-es para que app decodifique.
- Fallback: si Google falla, intentar fallback simple (linea recta + duracion estimada).

### Merge a main:
Cuando integration tests pasan.

---

## ETAPA 5: ETA Service (TÚ)
**Worktree:** `wt-eta` | **Branch:** `eta`
**Estado:** Implementado. Integration tests pendientes (requieren BD real).

### Contexto de dominio:
El ETA calculado es el tiempo de llegada del **bus al paradero**, no del usuario.
`SimpleETAProvider` es el placeholder para MVP v1; será reemplazado por
`GPSETAProvider` en MVP v2-A sin modificar `ETAService` ni los handlers.

### Tareas:
1. Crear interfaz en `internal/service/eta.go`:
   ```go
   type ETAProvider interface {
       GetETA(ctx context.Context, stopID int32) (seconds int, source string, err error)
   }

   // ErrNoVehicleData es retornado por GPSETAProvider cuando no hay posición
   // reciente. ETAService lo interpreta como señal para activar el fallback.
   var ErrNoVehicleData = ...
   ```

2. Implementar `SimpleETAProvider` (placeholder MVP v1):
   - ETA del bus simulado según hora del día:
     - Off-peak: `baseNormalS` (default 180s) + `stopID % 60` (offset por paradero).
     - Peak (7-9h, 17-19h): `basePeakS` (default 360s) + `stopID % 60`.
   - El offset `stopID % 60` simula buses en distintos puntos de la ruta.
   - Source: `"simple"`.
   - Configurable via functional options: `WithBaseETAs`, `WithPeakHours`.

3. Crear `ETACacheStore` + `ETAService` en `internal/service/eta_cache.go`:
   - `ETACacheStore` interface: `GetCachedETA` / `SetCachedETA`.
   - `ETAService` con soporte de fallback:
     - `NewETAService(primary, store)` — MVP v1 (sin fallback).
     - `NewETAServiceWithFallback(primary, fallback, store)` — MVP v2-A.
   - Resolución en cache miss:
     1. Llama primary provider.
     2. Si primary retorna `ErrNoVehicleData` y hay fallback → llama fallback,
        anota source como `"<source>_fallback"` (ej: `"simple_fallback"`).
     3. Escribe resultado en cache con TTL 60s.
   - Fallos de cache (get/set) son no-fatales.
   - `pgETACacheStore`: upsert sobre `stop_eta_cache` con `ON CONFLICT`.

4. Método principal: `ETAService.GetETAForStop(ctx, stopID int32)`.
   - Valida `stopID > 0`.
   - Retorna `(seconds, source, error)`.

### Tests implementados (`internal/service/eta_test.go`):
**ETAService (8 casos):**
- Cache hit → no llama al provider.
- Cache miss → llama primary, escribe cache.
- Cache miss → segunda llamada sirve desde cache.
- Cache expirado → llama primary.
- stopID inválido (0, -1) → error inmediato.
- Primary falla (error genérico, sin fallback) → error.
- Cache get falla → cae a primary (no-fatal).
- Cache set falla → retorna valor igualmente (no-fatal).

**ETAService con fallback (5 casos):**
- Primary retorna `ErrNoVehicleData` → llama fallback, source = `"simple_fallback"`.
- Primary tiene éxito → fallback no es llamado.
- Primary y fallback fallan → propaga error.
- `ErrNoVehicleData` sin fallback configurado → error.
- Resultado del fallback se cachea correctamente.

**SimpleETAProvider (6 casos):**
- Off-peak: `180 + offset` correcto.
- Peak: `360 + offset` correcto.
- Peak siempre > off-peak para mismo stopID.
- Distintos stopIDs → distintos ETAs.
- Offset wrappea en 60 (stopID=60 → offset=0, stopID=61 → offset=1).
- Opciones custom (`WithBaseETAs`, `WithPeakHours`).

### Integration tests pendientes (requieren BD real):
- Contra `stop_eta_cache` real: valida upsert, TTL, expiración.
- ETA se cachea en primera llamada; segunda llamada < 1ms (sin provider).

### Consideraciones:
- TTL corto (60s): ETA cambia frecuentemente.
- Source expuesto en respuesta JSON: útil para debugging y telemetría futura.
- `ErrNoVehicleData` como sentinel error: permite que ETAService distinga
  "no hay datos" (fallback graceful) de un error real del provider.
- En MVP v2-A, el único cambio en `app.go` es:
  ```go
  // Antes:
  etaSvc = service.NewETAService(simple, pgStore)
  // Después:
  etaSvc = service.NewETAServiceWithFallback(gps, simple, pgStore)
  ```

### Merge a main:
Cuando integration tests pasan.

---

## ETAPA 6: Handlers y Endpoints REST (TÚ)
**Worktree:** `wt-api` | **Branch:** `api`

### Tareas:
1. Crear `internal/handler/stops.go`:
   ```go
   // GET /stops/nearby?lat=X&lon=Y&radius=Z
   func (h *Handler) ListStopsNear(c *gin.Context) {
       lat := c.Query("lat")
       lon := c.Query("lon")
       radius := c.Query("radius") // metros, default 1000
       
       stops, err := h.stopsRepo.FindStopsNear(c, lat, lon, radius)
       if err != nil {
           c.JSON(500, gin.H{"error": err.Error()})
           return
       }
       c.JSON(200, stops)
   }
   
   // GET /stops/{id}
   func (h *Handler) GetStop(c *gin.Context) {
       id := c.Param("id")
       stop, err := h.stopsRepo.GetStop(c, id)
       if err != nil {
           c.JSON(404, gin.H{"error": "stop not found"})
           return
       }
       eta, _, _ := h.etaProvider.GetETA(c, id)
       c.JSON(200, gin.H{
           "stop": stop,
           "eta_seconds": eta,
       })
   }
   ```

2. Crear `internal/handler/routes.go`:
   ```go
   // GET /routes/to-stop?lat=X&lon=Y&stop_id=Z
   func (h *Handler) GetRouteToStop(c *gin.Context) {
       lat := c.Query("lat")
       lon := c.Query("lon")
       stopID := c.Query("stop_id")
       
       route, err := h.routingService.GetRouteTo(c, lat, lon, stopID)
       if err != nil {
           c.JSON(500, gin.H{"error": err.Error()})
           return
       }
       c.JSON(200, route)
   }
   ```

3. Registrar handlers en `cmd/server/main.go`:
   ```go
   api := r.Group("/api/v1")
   api.GET("/stops/nearby", handler.ListStopsNear)
   api.GET("/stops/:id", handler.GetStop)
   api.GET("/routes/to-stop", handler.GetRouteToStop)
   ```

4. Contratos JSON claros (documentacion para app):
   - Response `GET /stops/nearby`:
     ```json
     [
       {"id": 1, "name": "Paradero Centro", "lat": -12.123, "lon": -76.456}
     ]
     ```
   - Response `GET /stops/{id}`:
     ```json
     {"id": 1, "name": "...", "lat": ..., "lon": ..., "eta_seconds": 300}
     ```
   - Response `GET /routes/to-stop`:
     ```json
     {"polyline": "...", "distance_m": 500, "duration_s": 600}
     ```

### Tests:
- Unit: mock de repos/services, valida respuestas HTTP.
- Integration: contra API completa + BD mock.

### Consideraciones:
- Query params: validar tipos (float, int).
- Errores: respuestas consistentes (error codes).
- CORS: si app está en otro dominio, agregar middleware.

### Mock de storage/routing mientras esperas mi código:
- Crear mocks que devuelven datos hardcodeados.
- Permite pasar tests sin dependencias externas.
- Al final, reemplaza mocks con mis implementaciones.

---

## ETAPA 7: Integración y Demo (AMBOS)
**Worktree:** `wt-demo` | **Branch:** `demo`

### Tareas:
1. Mergear todas las branches a `demo`:
   - `git merge bootstrap db-model storage routing eta api`.
2. Validar que no hay conflictos.
3. Cargar datos reales:
   - Paraderos de tu ciudad (coords reales).
   - Rutas con shapes reales.
4. End-to-end test:
   - Arrange: usuario en lat/lon X, hay stop cercano con id 5.
   - Act: `GET /stops/nearby?lat=X&lon=Y&radius=1000`.
   - Assert: respuesta incluye stop 5 con ETA, tiempo < 500ms.
   - Act: `GET /routes/to-stop?lat=X&lon=Y&stop_id=5`.
   - Assert: respuesta incluye polyline válido, duracion ~ 600s.
5. Bench basico: medir p95 en ciclo de 100 req.
6. Documentar API con `swaggo/swag`:
   - Instalar: `go install github.com/swaggo/swag/cmd/swag@latest`.
   - Agregar anotaciones OpenAPI en los handlers de Etapa 6 (stops y routes).
   - Registrar ruta `/swagger/*any` en Gin con `ginSwagger.WrapHandler`.
   - Generar: `swag init -g cmd/server/main.go -o docs/`.
   - **Hacerlo aquí y no antes:** los handlers deben estar estables antes de anotar;
     anotar durante desarrollo implica actualizarlos dos veces.
7. Documentacion minima: README con endpoints y ejemplos curl.

### Tests:
- Integration: escenario completo de usuario (nearby + route + ETA).
- Performance: validar SLA < 500ms p95.

### Merge a main:
Cuando todo pasa y demo es estable.

---

## División de Trabajo Paralelo

| Yo | Tú |
|---|---|
| **Etapa 1:** Bootstrap (arquitectura base) | Revisar plan, preparar workspace |
| **Etapa 2:** Modelo datos (schema SQL) | Definir contratos JSON |
| **Etapa 3:** Storage layer sqlc (repos) | Crear mocks de repos para handlers |
| **Etapa 4:** Routing centralizado (Google) | Implementar ETA service |
| **Etapa 5-6:** (esperando resultados 3-4) | **Etapa 5:** ETA service completo |
| | **Etapa 6:** Handlers + endpoints + tests |
| **Ambos:** Etapa 7 - Integración y demo | **Ambos:** Validar end-to-end |

---

## Consideraciones Técnicas Generales

### Errores y Logging:
- Loggear a stdout (simple para MVP).
- Custom error types para BD, routing, config.

### Configuracion:
- Env vars: `DB_DSN`, `GOOGLE_API_KEY`, `PORT`.
- Validar al arrancar; fail fast si faltan.

### Timeouts:
- BD queries: 5s.
- Google API: 5s.
- HTTP handlers: 10s.

### Cache:
- ETA: TTL 60s.
- Rutas: TTL 120s.
- Limpieza: tareas periodicas (opcional para MVP).

### Testing:
- `go test ./...` debe pasar en cada etapa.
- Table-driven tests para logica SQL.
- Mocks de BD/API para unit tests.

### Git:
- Commits pequeños y claros por feature.
- PR (o squash) antes de merge a main.

---

## Paso Siguiente

Una vez que valides que VS Code funciona:
1. Voy a crear branches y worktrees.
2. Comienzo Etapa 1 (bootstrap) en `wt-bootstrap`.
3. Tú puedes comenzar en paralelo:
   - Etapa 5: `wt-eta` (define interfaz + implementa simple).
   - Crear mocks en `wt-api` para handlers.
   - Definir contratos JSON finales.

Te avisaré cuando Etapa 1 y 2 estén listos para que integres.

---

## Roadmap post-MVP v1

El MVP v1 cubre únicamente los 3 endpoints públicos definidos en Etapas 5-7.
No requieren autenticación: son de solo lectura y están pensados para usuarios comunes.

Una vez que el demo de MVP v1 sea estable, se procederá con **MVP v2**:

---

### MVP v2-A: Tracking de Conductores y ETA Real
**Worktree:** `wt-tracking` | **Branch:** `tracking`

El objetivo es reemplazar `SimpleETAProvider` (simulación) con `GPSETAProvider`
(posición real del vehículo), manteniendo la misma interfaz `ETAProvider` para que
`ETAService`, los handlers y los tests no cambien.

#### Nueva tabla en BD (migración `003_vehicle_positions.sql`):
```sql
CREATE TABLE IF NOT EXISTS vehicle_positions (
  id          SERIAL PRIMARY KEY,
  route_id    INT NOT NULL REFERENCES routes(id),
  driver_id   INT NOT NULL REFERENCES users(id),  -- requiere MVP v2-B (auth)
  geom        GEOMETRY(POINT, 4326) NOT NULL,
  reported_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_vehicle_positions_route ON vehicle_positions(route_id);
CREATE INDEX IF NOT EXISTS idx_vehicle_positions_geom  ON vehicle_positions USING GIST(geom);
CREATE INDEX IF NOT EXISTS idx_vehicle_positions_ts    ON vehicle_positions(reported_at DESC);
```
- Tabla **logged** (no UNLOGGED): las posiciones son datos operacionales, no cache.
- `reported_at` indexado descendente: la consulta siempre busca la posición más reciente.
- `driver_id` puede quedar como `INT` nullable hasta que MVP v2-B (auth) esté listo,
  para no bloquear el tracking.

#### Nuevo endpoint REST (protegido por JWT — requiere MVP v2-B):
```
POST /driver/position
Body: { "route_id": 3, "lat": -12.055, "lon": -77.053 }
```
- El conductor reporta su posición desde la app Android.
- Inserta o actualiza en `vehicle_positions`.
- Frecuencia sugerida: cada 10-15 segundos desde la app.

#### Implementar `GPSETAProvider` en `internal/service/eta_gps.go`:
```go
type GPSETAProvider struct {
    pool           *pgxpool.Pool
    routingService *RoutingService
    staleThreshold time.Duration  // posiciones más viejas que esto → ErrNoVehicleData
}
```
Algoritmo:
1. Consultar `route_stops` para obtener las rutas que sirven `stopID`.
2. Para cada ruta, obtener la última posición de `vehicle_positions`
   donde `reported_at > NOW() - staleThreshold` (default: 5 min).
3. Si no hay ninguna posición reciente → retornar `ErrNoVehicleData`
   (ETAService activará el fallback a `SimpleETAProvider`).
4. Para cada posición encontrada, llamar `RoutingService.GetRouteTo(vehicleLat, vehicleLon, stopID)`
   para obtener `DurationS` (usa el cache de rutas de Etapa 4).
5. Retornar el menor `DurationS` entre todas las rutas. Source: `"gps"`.

#### Wiring en `app.go` (único cambio al integrar GPS):
```go
// MVP v1 (actual):
etaProvider := service.NewSimpleETAProvider()
etaSvc      := service.NewETAService(etaProvider, pgETAStore)

// MVP v2-A (reemplazar las líneas anteriores):
gpsProvider    := service.NewGPSETAProvider(pool, routingService, 5*time.Minute)
simpleProvider := service.NewSimpleETAProvider()
etaSvc         := service.NewETAServiceWithFallback(gpsProvider, simpleProvider, pgETAStore)
```

#### Fuente (`source`) reportada al cliente:
| Valor | Significado |
|---|---|
| `"cache"` | Devuelto desde `stop_eta_cache` sin recalcular |
| `"gps"` | Calculado con posición real del conductor |
| `"simple_fallback"` | GPS intentado pero sin posición reciente → cayó a SimpleETAProvider |
| `"simple"` | SimpleETAProvider directo (MVP v1 sin GPS activo) |

#### Tests:
- Unit: mock de `vehicle_positions` (repo), valida selección del menor DurationS.
- Unit: valida que `ErrNoVehicleData` se retorna cuando no hay posición reciente.
- Integration: conductor reporta posición → `GET /stops/:id` devuelve ETA `"gps"`.
- Integration: sin posición reciente → ETA con source `"simple_fallback"`.

#### Consideraciones:
- `staleThreshold` configurable via env var `ETA_STALE_THRESHOLD` (default 5m).
- El cache de ETA (`stop_eta_cache`, TTL 60s) amortigua las llamadas a Google Routes
  aunque el conductor actualice posición cada 10s.
- Si un paradero es servido por múltiples rutas, se reporta el ETA del vehículo más
  cercano (menor DurationS), no el promedio.

---

### MVP v2-B: Autenticación JWT (conductores y administradores)
**Worktree:** `wt-auth` | **Branch:** `auth`

Prerrequisito de MVP v2-A para asociar posiciones a conductores reales.

#### Nueva tabla en BD (migración `004_users.sql`):
```sql
CREATE TYPE user_role AS ENUM ('driver', 'admin');

CREATE TABLE IF NOT EXISTS users (
  id            SERIAL PRIMARY KEY,
  username      VARCHAR(100) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role          user_role NOT NULL,
  active        BOOLEAN DEFAULT true,
  created_at    TIMESTAMP DEFAULT NOW()
);
```
- `role` como enum desde el inicio: evita migraciones costosas al agregar roles.
- `password_hash`: bcrypt, costo mínimo 12.

#### Nuevos endpoints:
```
POST /auth/register  — solo admin puede crear cuentas de conductor
POST /auth/login     — devuelve JWT firmado (claims: user_id, role, exp)
POST /auth/refresh   — renueva el token (requiere refresh token válido)
```

#### Middleware Gin:
- Validar JWT en rutas protegidas (`/driver/*`).
- Rutas públicas (`/api/v1/stops`, `/api/v1/routes`) sin cambios.
- Extraer `user_id` y `role` del token y colocarlos en el contexto Gin.

#### Librería sugerida: `golang-jwt/jwt/v5`.

#### Consideraciones:
- El flujo de auth debe estar en su propio worktree/branch (`wt-auth`) para no
  bloquear MVP v2-A.
- Si MVP v2-A llega antes que v2-B, `driver_id` en `vehicle_positions` puede ser
  nullable temporalmente y llenarse al integrar auth.
- Swagger (agregado en Etapa 7) debe actualizarse para incluir los endpoints de auth
  y los esquemas de seguridad JWT (`securityDefinitions`).

---

### Orden de implementación sugerido para MVP v2:

```
v2-B (auth: usuarios + JWT)  ──┐
                               ├──▶  v2-A (tracking GPS + ETA real)
v2-A sin driver_id (tracking) ─┘
```

Opción pragmática: implementar v2-A con `driver_id` nullable primero para tener
ETA real funcionando, y completar la asociación al terminar v2-B.
