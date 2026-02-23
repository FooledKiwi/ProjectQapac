# Deuda Técnica

Registro de problemas conocidos y decisiones diferidas identificados durante el
desarrollo. Cada ítem incluye contexto, impacto real y la solución propuesta.

---

## [TD-01] Escritura al cache ETA usa el contexto del request

**Archivo:** `internal/service/eta_cache.go` — `ETAService.GetETAForStop`  
**Severidad:** Alta  
**Detectado en:** Etapa 5 (ETA Service)  
**Bloquea MVP v2:** Sí — se vuelve frecuente cuando `GPSETAProvider` entra en juego.

### Problema
`SetCachedETA` recibe el mismo `ctx` del request HTTP. Si el cliente cierra la
conexión justo después de recibir la respuesta, el contexto se cancela y la
escritura al cache falla silenciosamente. El ETA recién computado (potencialmente
tras una llamada costosa a Google Routes) se descarta.

El routing layer (`internal/routing/cache.go`) ya resolvió esto correctamente con
una goroutine + `context.Background()`.

### Solución
Mover la escritura al cache a una goroutine con contexto independiente, igual que
`CachedRouter.Route`:

```go
go func() {
    storeCtx, cancel := context.WithTimeout(context.Background(), etaCacheQueryTimeout)
    defer cancel()
    if err := s.store.SetCachedETA(storeCtx, stopID, secs); err != nil {
        // log non-fatal error
    }
}()
```

---

## [TD-02] `stop_eta_cache` tiene un único slot por paradero (sin distinción de ruta)

**Archivo:** `internal/migrations/001_initial_schema.sql`  
**Severidad:** Alta  
**Detectado en:** Etapa 5 (ETA Service)  
**Bloquea MVP v2:** Sí — `GPSETAProvider` elige el menor ETA entre varias rutas;
el cache actual puede sobrescribir ese resultado con el de la ruta equivocada.

### Problema
El schema define `UNIQUE(stop_id)` en `stop_eta_cache`, lo que significa un único
valor de ETA por paradero. Si un paradero es servido por dos rutas (bus A llega
en 2 min, bus B en 8 min) y dos requests concurrentes escriben el cache, el
resultado correcto (2 min) puede quedar desplazado por el incorrecto (8 min).

Para MVP v1 con `SimpleETAProvider` no tiene impacto porque el ETA es determinista.

### Solución (elegir una al diseñar migración `003`):

**Opción A — ETA mínimo explícito (recomendada para MVP v2):**  
Mantener `UNIQUE(stop_id)` pero documentar que el valor almacenado es siempre el
ETA mínimo entre todas las rutas activas. `GPSETAProvider` calcula el mínimo antes
de escribir; el upsert siempre sobrescribe con el valor más reciente.

**Opción B — Cache por (stop_id, route_id):**  
Cambiar la clave a `UNIQUE(stop_id, route_id)`. Permite cachear ETA por ruta y
que el handler devuelva la lista completa o el mínimo según el contrato JSON.
Requiere cambio de schema y de `ETACacheStore`.

---

## [TD-03] `GetETAForStop` no aplica timeout propio a BD ni al provider

**Archivo:** `internal/service/eta_cache.go` — `ETAService.GetETAForStop`  
**Severidad:** Media  
**Detectado en:** Etapa 5 (ETA Service)

### Problema
El service asume que el contexto entrante ya tiene un deadline configurado por el
handler. Si el handler de Etapa 6 no aplica `context.WithTimeout`, la query de
cache y la llamada al provider se ejecutan sin límite de tiempo propio.

El plan especifica "HTTP handlers: 10s" pero depende de que cada handler lo
configure correctamente, lo cual es fácil de olvidar.

### Solución
Aplicar un timeout interno en `GetETAForStop` como defensa en profundidad:

```go
const etaServiceTimeout = 8 * time.Second // < 10s del handler

func (s *ETAService) GetETAForStop(ctx context.Context, stopID int32) (...) {
    ctx, cancel := context.WithTimeout(ctx, etaServiceTimeout)
    defer cancel()
    ...
}
```

Alternativamente, centralizar el timeout en un middleware Gin que lo aplique a
todos los handlers, lo cual elimina el riesgo de olvidarlo por endpoint.

---

## [TD-04] Anotación `_fallback` en `source` puede ser semánticamente incorrecta

**Archivo:** `internal/service/eta_cache.go` — `ETAService.GetETAForStop`  
**Severidad:** Baja  
**Detectado en:** Etapa 5 (ETA Service)  
**Bloquea MVP v2:** No — solo afecta telemetría.

### Problema
El sufijo `_fallback` se concatena al `source` devuelto por cualquier fallback
provider:

```go
src = src + "_fallback"
```

Si el fallback devolviera `source = "cache"` (por ejemplo, si en el futuro el
fallback fuera otro service con su propio cache), el resultado sería
`"cache_fallback"`, que es semánticamente ambiguo.

Hoy no ocurre porque `SimpleETAProvider` siempre retorna `"simple"`, pero es
una trampa al agregar providers compuestos.

### Solución
Definir los valores de `source` como constantes en el package y construir el
string anotado de forma explícita en lugar de concatenar ciegamente:

```go
const (
    SourceCache          = "cache"
    SourceSimple         = "simple"
    SourceGPS            = "gps"
    SourceSimpleFallback = "simple_fallback"
)
```

`ETAService` asignaría la constante correcta en lugar de depender del string
devuelto por el fallback.

---

## [TD-05] TTL del cache ETA no considera la antigüedad del dato de origen (MVP v2)

**Archivo:** `internal/service/eta_cache.go`  
**Severidad:** Baja  
**Detectado en:** Etapa 5 (ETA Service)  
**Bloquea MVP v2:** No — degradación de precisión aceptable para MVP.

### Problema
El TTL de 60s es fijo independientemente de la frescura de la posición GPS usada
para calcularlo. Un ETA calculado a partir de una posición de hace 4 min 55s se
cachea con los mismos 60s que uno calculado con posición de hace 5s. El ETA
podría llegar al cliente con hasta ~5 min de desactualización acumulada.

### Solución (para MVP v2-A)
`GPSETAProvider` devuelve el timestamp de la posición usada. `SetCachedETA`
recibe ese timestamp y calcula:

```
expires_at = min(positionTimestamp + staleThreshold, now + etaCacheTTL)
```

Esto acorta el TTL del cache cuando el dato de origen ya es viejo, evitando
servir ETAs basados en posiciones casi stale.

---

## [TD-06] `ErrNoVehicleData` es una variable mutable exportada

**Archivo:** `internal/service/eta.go`  
**Severidad:** Baja  
**Detectado en:** Etapa 5 (ETA Service)

### Problema
```go
var ErrNoVehicleData = errNoVehicleData("no recent vehicle position data")
```

Al ser `var`, cualquier paquete externo podría reasignar `service.ErrNoVehicleData`,
rompiendo los `errors.Is` que dependen de la identidad del valor. En la práctica
es poco probable, pero no está blindado.

### Solución
Cambiar a una variable de solo lectura efectiva declarando el tipo concreto como
puntero a struct, o documentar explícitamente que no debe reasignarse. En Go no
existe `const` para errores no primitivos, pero la convención idiomática es:

```go
// ErrNoVehicleData should not be reassigned.
var ErrNoVehicleData error = errNoVehicleData("no recent vehicle position data")
```

El type privado `errNoVehicleData` ya previene que código externo construya un
valor igual por accidente, lo cual es la protección más importante para `errors.Is`.
