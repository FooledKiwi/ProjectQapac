# Endpoints de Conductor

Estos endpoints requieren un **JWT valido** con rol `driver` o `admin`.

**Header requerido:**
```
Authorization: Bearer <access_token>
```

---

## POST /api/v1/driver/position

Reporta la posicion GPS actual del conductor. Se asocia automaticamente al
vehiculo que tiene asignado.

**Request Body:**

```json
{
  "lat": -7.1640,
  "lon": -78.5010,
  "heading": 180.5,
  "speed": 25.3
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `lat` | float64 | Si | Latitud WGS-84 |
| `lon` | float64 | Si | Longitud WGS-84 |
| `heading` | float64 | No | Direccion en grados (0-360) |
| `speed` | float64 | No | Velocidad en km/h |

**Response 200:**

```json
{
  "status": "ok"
}
```

**Errores:**

| Codigo | Mensaje | Causa |
|--------|---------|-------|
| 400 | (validacion) | Falta lat o lon |
| 401 | `authentication required` | Sin token o token invalido |
| 409 | `no active vehicle assignment` | Conductor sin vehiculo asignado |

**Uso en Android:**

Se recomienda enviar la posicion cada 5-10 segundos mientras el viaje este activo:

```kotlin
// Usar FusedLocationProviderClient para obtener ubicacion
fusedLocationClient.lastLocation.addOnSuccessListener { location ->
    val body = ReportPositionRequest(
        lat = location.latitude,
        lon = location.longitude,
        heading = location.bearing.toDouble(),
        speed = location.speed.toDouble() * 3.6  // m/s a km/h
    )
    apiService.reportPosition(body).enqueue(...)
}
```

---

## GET /api/v1/driver/profile

Obtiene el perfil del conductor autenticado.

**Response 200:**

```json
{
  "id": 2,
  "username": "conductor1",
  "full_name": "Carlos Quispe",
  "phone": "976543210",
  "profile_image_path": ""
}
```

---

## PUT /api/v1/driver/profile

Actualiza el perfil del conductor. Solo se actualizan los campos enviados
(los campos vacios se ignoran).

**Request Body:**

```json
{
  "full_name": "Carlos Quispe Mendoza",
  "phone": "976543210"
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `full_name` | string | No | Nombre completo (se ignora si vacio) |
| `phone` | string | No | Telefono (se ignora si vacio) |

**Response 200:**

```json
{
  "id": 2,
  "username": "conductor1",
  "full_name": "Carlos Quispe Mendoza",
  "phone": "976543210"
}
```

---

## GET /api/v1/driver/assignment

Obtiene la asignacion actual del conductor (vehiculo y cobrador).

**Response 200:**

```json
{
  "vehicle": {
    "id": 1,
    "plate": "ABC-123",
    "route_name": "R01 - Centro a Baños del Inca"
  },
  "collector_name": "Maria Lopez",
  "assigned_at": "2025-01-15T06:00:00Z"
}
```

> **collector_name** puede ser `null` si no hay cobrador asignado.

**Errores:**

| Codigo | Mensaje | Causa |
|--------|---------|-------|
| 401 | `authentication required` | Sin token |
| 404 | `no active assignment` | Conductor sin vehiculo asignado |

---

## POST /api/v1/driver/trips/start

Inicia un nuevo viaje. Requiere:
1. Tener un vehiculo asignado
2. El vehiculo debe tener una ruta asignada
3. No tener un viaje activo previo

**Request Body:** No requiere cuerpo.

**Response 201:**

```json
{
  "trip_id": 1,
  "vehicle_id": 1,
  "route_id": 1,
  "started_at": "2025-01-15T06:30:00Z"
}
```

**Errores:**

| Codigo | Mensaje | Causa |
|--------|---------|-------|
| 401 | `authentication required` | Sin token |
| 409 | `an active trip already exists` | Ya tiene viaje activo |
| 409 | `no active vehicle assignment` | Sin vehiculo asignado |
| 409 | `vehicle has no route assigned` | Vehiculo sin ruta |

> **Nota**: Si ya tiene un viaje activo, la respuesta 409 incluye `trip_id`
> del viaje existente:
> ```json
> {"error": "an active trip already exists", "trip_id": 5}
> ```

---

## POST /api/v1/driver/trips/end

Finaliza el viaje activo del conductor.

**Request Body:** No requiere cuerpo.

**Response:** `204 No Content` (sin cuerpo)

**Errores:**

| Codigo | Mensaje | Causa |
|--------|---------|-------|
| 401 | `authentication required` | Sin token |
| 404 | `no active trip to end` | No hay viaje activo |

---

## POST /api/v1/driver/uploads/images

Sube una imagen desde la app del conductor (ej. foto del vehiculo, incidencia).

**Request:** `multipart/form-data`

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `file` | binary | Si | Imagen JPEG, PNG o WebP (max 5 MB) |

**Ejemplo con curl:**

```bash
curl -X POST http://localhost:8080/api/v1/driver/uploads/images \
  -H "Authorization: Bearer <token>" \
  -F "file=@foto.jpg"
```

**Ejemplo en Android (Kotlin + Retrofit):**

```kotlin
// Definir la interfaz
@Multipart
@POST("api/v1/driver/uploads/images")
suspend fun uploadImage(@Part file: MultipartBody.Part): UploadResponse

// Usar
val file = File(imagePath)
val requestBody = file.asRequestBody("image/jpeg".toMediaType())
val part = MultipartBody.Part.createFormData("file", file.name, requestBody)
val response = apiService.uploadImage(part)
// response.filename = "uuid.jpg"
// response.url = "http://host/api/v1/uploads/images/uuid.jpg"
```

**Response 201:**

```json
{
  "filename": "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg",
  "url": "http://localhost:8080/api/v1/uploads/images/a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg"
}
```

**Errores:**

| Codigo | Mensaje | Causa |
|--------|---------|-------|
| 400 | `missing or invalid 'file' field` | No se envio archivo |
| 400 | `unsupported file type "..."` | Tipo no permitido |
| 401 | `authentication required` | Sin token |
| 413 | `file must not exceed 5 MB` | Archivo muy grande |

---

## Flujo Tipico del Conductor

```
1. Login
   POST /api/v1/auth/login

2. Ver asignacion (vehiculo + ruta)
   GET /api/v1/driver/assignment

3. Iniciar viaje
   POST /api/v1/driver/trips/start

4. Reportar posicion GPS (cada 5-10 seg)
   POST /api/v1/driver/position  (en loop)

5. Finalizar viaje
   POST /api/v1/driver/trips/end

6. Logout
   POST /api/v1/auth/logout
```
