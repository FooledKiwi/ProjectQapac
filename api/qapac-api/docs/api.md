# QAPAC API — Referencia de Endpoints (v1)

**Base URL:** `http://localhost:8080`  
**Prefijo:** `/api/v1`  
**Formato:** JSON (`Content-Type: application/json`)  
**Autenticación:** ninguna (endpoints públicos, solo lectura)

---

## Esquemas de datos

### `Stop`

| Campo | Tipo | Descripción |
|---|---|---|
| `id` | `integer` | Identificador único del paradero |
| `name` | `string` | Nombre descriptivo |
| `lat` | `number` | Latitud WGS-84 |
| `lon` | `number` | Longitud WGS-84 |

### `StopWithETA`

Extiende `Stop` con un campo adicional:

| Campo | Tipo | Descripción |
|---|---|---|
| `eta_seconds` | `integer` | Segundos estimados hasta la llegada del próximo bus. `0` si el servicio de ETA no está disponible |

### `Route`

| Campo | Tipo | Descripción |
|---|---|---|
| `polyline` | `string` | Ruta codificada en [Google Encoded Polyline Algorithm](https://developers.google.com/maps/documentation/utilities/polylinealgorithm). Vacío (`""`) cuando `is_fallback` es `true` |
| `distance_m` | `integer` | Distancia en metros |
| `duration_s` | `integer` | Duración estimada en segundos |
| `is_fallback` | `boolean` | `true` cuando la respuesta fue calculada con el estimador de línea recta en lugar de Google Routes API. El cliente puede usar este campo para mostrar un aviso al usuario |

### `Error`

| Campo | Tipo | Descripción |
|---|---|---|
| `error` | `string` | Descripción del error |

---

## Endpoints

---

### `GET /health`

Verifica que el servidor esté operativo. No requiere parámetros.

**Respuesta `200`**
```json
{"status": "ok"}
```

**curl**
```bash
curl http://localhost:8080/health
```

---

### `GET /api/v1/stops/nearby`

Devuelve todos los paraderos activos dentro de un radio a partir de una coordenada, ordenados por distancia ascendente.

#### Parámetros de query

| Parámetro | Tipo | Requerido | Default | Descripción |
|---|---|---|---|---|
| `lat` | `float` | si | — | Latitud del punto de origen (WGS-84) |
| `lon` | `float` | si | — | Longitud del punto de origen (WGS-84) |
| `radius` | `float` | no | `1000` | Radio de búsqueda en metros. Debe ser positivo y no mayor a `50000` |

#### Respuestas

| Código | Descripción | Cuerpo |
|---|---|---|
| `200` | Lista de paraderos cercanos (puede ser `[]`) | `Stop[]` |
| `400` | Parámetro faltante o inválido | `Error` |
| `500` | Error interno de base de datos | `Error` |

#### Ejemplo — paraderos en radio de 500 m alrededor de Plaza Mayor (Lima)

```bash
curl "http://localhost:8080/api/v1/stops/nearby?lat=-12.0464&lon=-77.0282&radius=500"
```

```json
[
  {
    "id": 1,
    "name": "Paradero Plaza Mayor",
    "lat": -12.0464,
    "lon": -77.0282
  },
  {
    "id": 2,
    "name": "Paradero Breña",
    "lat": -12.058,
    "lon": -77.045
  }
]
```

#### Ejemplo — radio de 5 km

```bash
curl "http://localhost:8080/api/v1/stops/nearby?lat=-12.0464&lon=-77.0282&radius=5000"
```

#### Ejemplo — error por parámetro faltante

```bash
curl "http://localhost:8080/api/v1/stops/nearby?lat=-12.0464"
```

```json
{"error": "lon query parameter is required"}
```

---

### `GET /api/v1/stops/:id`

Devuelve un paradero por ID junto con el ETA estimado del próximo bus.

#### Parámetros de ruta

| Parámetro | Tipo | Descripción |
|---|---|---|
| `id` | `integer` | ID del paradero. Debe ser un entero positivo |

#### Respuestas

| Código | Descripción | Cuerpo |
|---|---|---|
| `200` | Paradero encontrado con ETA | `StopWithETA` |
| `400` | `id` no es un entero positivo | `Error` |
| `404` | Paradero no encontrado | `Error` |
| `500` | Error interno de base de datos | `Error` |

> **Nota sobre `eta_seconds`:** en MVP v1, el ETA es una estimación simulada basada en la hora del día (off-peak: ~180 s, peak 7–9 h / 17–19 h: ~360 s) más un offset determinístico por paradero. Si el servicio de ETA falla, `eta_seconds` devuelve `0` y el resto de los datos del paradero se incluyen igualmente.

#### Ejemplo — paradero existente

```bash
curl http://localhost:8080/api/v1/stops/5
```

```json
{
  "id": 5,
  "name": "Paradero Miraflores Centro",
  "lat": -12.117,
  "lon": -77.03,
  "eta_seconds": 185
}
```

#### Ejemplo — paradero no encontrado

```bash
curl http://localhost:8080/api/v1/stops/999
```

```json
{"error": "stop not found"}
```

#### Ejemplo — ID inválido

```bash
curl http://localhost:8080/api/v1/stops/abc
```

```json
{"error": "id must be a positive integer"}
```

---

### `GET /api/v1/routes/to-stop`

Calcula la ruta en auto desde la ubicación del usuario hasta un paradero específico. En producción usa Google Routes API v2; si la API no está disponible (o `GOOGLE_API_KEY` está vacío), devuelve una estimación de línea recta con `polyline: ""`.

#### Parámetros de query

| Parámetro | Tipo | Requerido | Descripción |
|---|---|---|---|
| `lat` | `float` | si | Latitud del usuario (WGS-84) |
| `lon` | `float` | si | Longitud del usuario (WGS-84) |
| `stop_id` | `integer` | si | ID del paradero destino. Debe ser un entero positivo |

#### Respuestas

| Código | Descripción | Cuerpo |
|---|---|---|
| `200` | Ruta calculada | `Route` |
| `400` | Parámetro faltante o inválido | `Error` |
| `404` | El paradero no existe | `Error` |
| `500` | Error interno del router | `Error` |

> **Nota sobre caché:** las rutas se cachean 120 segundos en `route_to_stop_cache`. Una segunda llamada con el mismo origen (±76 m) y `stop_id` se sirve desde la base de datos sin invocar Google.

#### Ejemplo — ruta desde Plaza Mayor al paradero 5

```bash
curl "http://localhost:8080/api/v1/routes/to-stop?lat=-12.0464&lon=-77.0282&stop_id=5"
```

```json
{
  "polyline": "abcdEfghiJ...",
  "distance_m": 7840,
  "duration_s": 1020,
  "is_fallback": false
}
```

#### Ejemplo — sin API key (fallback de línea recta)

```json
{
  "polyline": "",
  "distance_m": 7612,
  "duration_s": 912,
  "is_fallback": true
}
```

#### Ejemplo — parámetro inválido

```bash
curl "http://localhost:8080/api/v1/routes/to-stop?lat=-12.0464&lon=-77.0282&stop_id=0"
```

```json
{"error": "stop_id must be a positive integer"}
```

---

## Datos de demo (seed)

El servidor carga automáticamente los siguientes fixtures al arrancar por primera vez:

### Paraderos (Lima metropolitana)

| ID | Nombre | Lat | Lon |
|---|---|---|---|
| 1 | Paradero Plaza Mayor | -12.0464 | -77.0282 |
| 2 | Paradero Breña | -12.0580 | -77.0450 |
| 3 | Paradero La Victoria | -12.0650 | -77.0196 |
| 4 | Paradero San Borja Norte | -12.0870 | -77.0050 |
| 5 | Paradero Miraflores Centro | -12.1170 | -77.0300 |
| 6 | Paradero Ovalo Gutierrez | -12.1050 | -77.0350 |
| 7 | Paradero San Isidro | -12.0960 | -77.0400 |
| 8 | Paradero Surquillo | -12.1080 | -77.0270 |

### Rutas

| ID | Nombre | Paraderos (secuencia) |
|---|---|---|
| 1 | Ruta A — Centro a Miraflores | 1 → 2 → 3 → 4 → 5 |
| 2 | Ruta B — Miraflores a San Isidro | 5 → 6 → 7 → 8 |

---

## Flujo típico de integración

```
1. GET /api/v1/stops/nearby?lat=<user_lat>&lon=<user_lon>&radius=1000
       → Obtener lista de paraderos cercanos

2. GET /api/v1/stops/<stop_id>
       → Ver detalle del paradero elegido + ETA del bus

3. GET /api/v1/routes/to-stop?lat=<user_lat>&lon=<user_lon>&stop_id=<stop_id>
       → Obtener ruta para llegar caminando/en auto al paradero
```

---

## Configuración del servidor

| Variable de entorno | Requerida | Default | Descripción |
|---|---|---|---|
| `DB_DSN` | si | — | Connection string de PostgreSQL. Ej: `postgres://user:pass@localhost:5432/qapac_db?sslmode=disable` |
| `GOOGLE_API_KEY` | no | `""` | API key de Google Cloud con Routes API habilitada. Sin ella, el endpoint `/routes/to-stop` usa fallback de línea recta |
| `PORT` | no | `8080` | Puerto HTTP en que escucha el servidor |

### Arranque rápido (local)

```bash
# 1. Levantar PostGIS
mise run db:up

# 2. Iniciar API (aplica migraciones y seed automáticamente)
mise run api:dev
```
