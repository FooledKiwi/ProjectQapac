# Endpoints Publicos

Estos endpoints **NO requieren autenticacion**. Son para uso de la app movil
de pasajeros (usuarios anonimos).

---

## Paradas

### GET /api/v1/stops/nearby

Busca paradas de bus cercanas a una ubicacion.

**Query Parameters:**

| Parametro | Tipo | Requerido | Default | Descripcion |
|-----------|------|-----------|---------|-------------|
| `lat` | float | Si | - | Latitud WGS-84 |
| `lon` | float | Si | - | Longitud WGS-84 |
| `radius` | float | No | 1000 | Radio en metros (max 50000) |

**Ejemplo:**

```http
GET /api/v1/stops/nearby?lat=-7.1637&lon=-78.5003&radius=500
```

**Response 200:**

```json
[
  {
    "id": 1,
    "name": "Plaza de Armas",
    "lat": -7.1637,
    "lon": -78.5003
  },
  {
    "id": 2,
    "name": "Mercado Central",
    "lat": -7.1612,
    "lon": -78.5128
  }
]
```

---

### GET /api/v1/stops/:id

Obtiene el detalle de una parada, incluyendo el tiempo estimado de llegada
del proximo bus (en segundos).

**Path Parameters:**

| Parametro | Tipo | Descripcion |
|-----------|------|-------------|
| `id` | int32 | ID de la parada |

**Ejemplo:**

```http
GET /api/v1/stops/1
```

**Response 200:**

```json
{
  "id": 1,
  "name": "Plaza de Armas",
  "lat": -7.1637,
  "lon": -78.5003,
  "eta_seconds": 180
}
```

> **Nota**: `eta_seconds` sera `0` si no hay buses activos cerca o si no se
> puede calcular el ETA.

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | ID no es un entero positivo |
| 404 | Parada no existe |

---

## Rutas

### GET /api/v1/routes

Lista todas las rutas de transporte con la cantidad de vehiculos activos.

**Ejemplo:**

```http
GET /api/v1/routes
```

**Response 200:**

```json
[
  {
    "id": 1,
    "name": "R01 - Centro a Baños del Inca",
    "active": true,
    "vehicle_count": 3
  },
  {
    "id": 2,
    "name": "R02 - Fonavi a Centro",
    "active": true,
    "vehicle_count": 2
  }
]
```

---

### GET /api/v1/routes/:id

Detalle de una ruta con sus paradas ordenadas, vehiculos asignados y polyline.

**Path Parameters:**

| Parametro | Tipo | Descripcion |
|-----------|------|-------------|
| `id` | int32 | ID de la ruta |

**Response 200:**

```json
{
  "id": 1,
  "name": "R01 - Centro a Baños del Inca",
  "active": true,
  "stops": [
    {
      "id": 1,
      "name": "Plaza de Armas",
      "lat": -7.1637,
      "lon": -78.5003,
      "sequence": 1
    },
    {
      "id": 5,
      "name": "Av. Independencia",
      "lat": -7.1580,
      "lon": -78.5120,
      "sequence": 2
    }
  ],
  "vehicles": [
    {
      "id": 1,
      "plate": "ABC-123",
      "driver": "Carlos Quispe",
      "collector": "Maria Lopez",
      "status": "active"
    }
  ],
  "shape_polyline": "~fvnL~{bqM..."
}
```

> **shape_polyline**: Puede ser `null` si la ruta no tiene forma definida.
> Decodificar con `PolyUtil.decode()` en Android.

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | ID invalido |
| 404 | Ruta no existe |

---

### GET /api/v1/routes/:id/vehicles

Vehiculos de una ruta con su posicion GPS mas reciente. **Este es el endpoint
principal para mostrar buses en el mapa en tiempo real.**

**Path Parameters:**

| Parametro | Tipo | Descripcion |
|-----------|------|-------------|
| `id` | int32 | ID de la ruta |

**Response 200:**

```json
[
  {
    "id": 1,
    "plate": "ABC-123",
    "driver_name": "Carlos Quispe",
    "collector_name": "Maria Lopez",
    "status": "active",
    "position": {
      "lat": -7.1640,
      "lon": -78.5010,
      "heading": 180.5,
      "speed": 25.3,
      "recorded_at": "2025-01-15T10:30:00Z"
    }
  },
  {
    "id": 3,
    "plate": "DEF-456",
    "driver_name": null,
    "collector_name": null,
    "status": "inactive",
    "position": null
  }
]
```

> **position** es `null` si el vehiculo no ha reportado posicion GPS.
> **heading** es la direccion en grados (0-360). **speed** es en km/h.

---

### GET /api/v1/routes/to-stop

Calcula la ruta peatonal desde la ubicacion del usuario hasta una parada.
Usa Google Routes API v2 internamente.

**Query Parameters:**

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `lat` | float | Si | Latitud del usuario |
| `lon` | float | Si | Longitud del usuario |
| `stop_id` | int32 | Si | ID de la parada destino |

**Ejemplo:**

```http
GET /api/v1/routes/to-stop?lat=-7.1640&lon=-78.5010&stop_id=5
```

**Response 200:**

```json
{
  "polyline": "~fvnL~{bqM...",
  "distance_m": 450,
  "duration_s": 360,
  "is_fallback": false
}
```

| Campo | Descripcion |
|-------|-------------|
| `polyline` | Google Encoded Polyline (vacia si `is_fallback=true`) |
| `distance_m` | Distancia en metros |
| `duration_s` | Duracion estimada en segundos |
| `is_fallback` | `true` = no se pudo usar Google API, valores son estimacion en linea recta |

**Decodificar en Android:**

```kotlin
val points = PolyUtil.decode(response.polyline)
if (points.isNotEmpty()) {
    googleMap.addPolyline(
        PolylineOptions().addAll(points).width(8f).color(Color.BLUE)
    )
}
```

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | Parametros faltantes o invalidos |
| 404 | Parada no existe |

---

## Vehiculos

### GET /api/v1/vehicles/nearby

Busca vehiculos con posicion GPS registrada dentro de un radio.

**Query Parameters:**

| Parametro | Tipo | Requerido | Default | Descripcion |
|-----------|------|-----------|---------|-------------|
| `lat` | float | Si | - | Latitud WGS-84 |
| `lon` | float | Si | - | Longitud WGS-84 |
| `radius` | float | No | 1000 | Radio en metros (max 50000) |

**Ejemplo:**

```http
GET /api/v1/vehicles/nearby?lat=-7.1637&lon=-78.5003&radius=2000
```

**Response 200:**

```json
[
  {
    "id": 1,
    "plate": "ABC-123",
    "route_name": "R01 - Centro a Baños del Inca",
    "lat": -7.1640,
    "lon": -78.5010
  }
]
```

---

### GET /api/v1/vehicles/:id/position

Obtiene la ultima posicion GPS de un vehiculo especifico.

**Path Parameters:**

| Parametro | Tipo | Descripcion |
|-----------|------|-------------|
| `id` | int32 | ID del vehiculo |

**Response 200:**

```json
{
  "lat": -7.1640,
  "lon": -78.5010,
  "heading": 180.5,
  "speed": 25.3,
  "recorded_at": "2025-01-15T10:30:00Z"
}
```

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | ID invalido |
| 404 | Vehiculo no tiene posicion registrada |

---

## Alertas

### GET /api/v1/alerts

Lista las alertas de servicio activas.

**Query Parameters:**

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `route_id` | int32 | No | Filtrar por ID de ruta |

**Response 200:**

```json
[
  {
    "id": 1,
    "title": "Desvio en Av. Independencia",
    "description": "Por obras municipales, la ruta R01 desvia por Jr. Amazonas",
    "route_id": 1,
    "vehicle_plate": "",
    "image_path": "",
    "created_by": 1,
    "created_at": "2025-01-15T08:00:00Z"
  }
]
```

---

### GET /api/v1/alerts/:id

Obtiene el detalle de una alerta especifica.

**Response 200:** Mismo formato que un elemento del array de `GET /alerts`.

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | ID invalido |
| 404 | Alerta no existe |

---

## Calificaciones

### POST /api/v1/ratings

Permite a un usuario anonimo calificar un viaje (1-5 estrellas).
Se identifica al usuario por el UUID del dispositivo Android.

**Request Body:**

```json
{
  "trip_id": 1,
  "rating": 4,
  "device_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `trip_id` | int32 | Si | ID del viaje |
| `rating` | int | Si | Estrellas (1 a 5) |
| `device_id` | string | Si | UUID del dispositivo |

**Response 201:**

```json
{
  "id": 1,
  "trip_id": 1,
  "rating": 4,
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-01-15T10:30:00Z"
}
```

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | Campos faltantes o rating fuera de rango |
| 409 | El dispositivo ya califico este viaje |

---

## Favoritos

### GET /api/v1/favorites

Lista las rutas favoritas de un dispositivo.

**Query Parameters:**

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `device_id` | string | Si | UUID del dispositivo |

**Ejemplo:**

```http
GET /api/v1/favorites?device_id=550e8400-e29b-41d4-a716-446655440000
```

**Response 200:**

```json
[
  {
    "id": 1,
    "device_id": "550e8400-e29b-41d4-a716-446655440000",
    "route_id": 1,
    "route_name": "R01 - Centro a Baños del Inca",
    "created_at": "2025-01-15T08:00:00Z"
  }
]
```

---

### POST /api/v1/favorites

Agrega una ruta a favoritos.

**Request Body:**

```json
{
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "route_id": 1
}
```

**Response 201:**

```json
{
  "id": 1,
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "route_id": 1
}
```

---

### DELETE /api/v1/favorites

Elimina una ruta de favoritos.

**Request Body:**

```json
{
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "route_id": 1
}
```

**Response:** `204 No Content` (sin cuerpo)

---

## Imagenes

### GET /api/v1/uploads/images/:filename

Descarga una imagen previamente subida. Retorna el archivo binario con el
Content-Type correspondiente (image/jpeg, image/png, image/webp).

**Ejemplo:**

```http
GET /api/v1/uploads/images/a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg
```

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | Nombre de archivo invalido |
| 404 | Archivo no existe |
