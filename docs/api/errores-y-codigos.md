# Errores y Codigos HTTP

## Formato de Error

Todos los errores de la API retornan un JSON con la siguiente estructura:

```json
{
  "error": "<mensaje descriptivo>"
}
```

El campo `error` siempre es un string. No hay codigos de error numericos
adicionales; el codigo HTTP es suficiente para determinar el tipo de error.

---

## Codigos HTTP Utilizados

### Exito

| Codigo | Descripcion | Cuando se usa |
|--------|-------------|---------------|
| `200 OK` | Operacion exitosa | GET, PUT, POST (posicion) |
| `201 Created` | Recurso creado | POST (crear) |
| `204 No Content` | Exito sin cuerpo | DELETE, logout, end trip |

### Errores del Cliente

| Codigo | Descripcion | Cuando se usa |
|--------|-------------|---------------|
| `400 Bad Request` | Solicitud mal formada | Parametros faltantes, tipos invalidos, validacion fallida |
| `401 Unauthorized` | No autenticado | Token JWT faltante, invalido o expirado |
| `404 Not Found` | Recurso no existe | ID no encontrado en BD |
| `409 Conflict` | Conflicto de estado | Duplicados (rating), sin asignacion (conductor), viaje activo |
| `413 Payload Too Large` | Cuerpo muy grande | Upload de archivo > 5 MB |

### Errores del Servidor

| Codigo | Descripcion | Cuando se usa |
|--------|-------------|---------------|
| `500 Internal Server Error` | Error interno | Fallo de BD, error no manejado |

---

## Mensajes de Error Comunes

### Autenticacion

| Endpoint | Codigo | Mensaje |
|----------|--------|---------|
| `POST /auth/login` | 400 | `username and password are required` |
| `POST /auth/login` | 401 | `invalid username or password` |
| `POST /auth/refresh` | 400 | `refresh_token is required` |
| `POST /auth/refresh` | 401 | `invalid or expired refresh token` |
| `POST /auth/logout` | 400 | `refresh_token is required` |
| Cualquier endpoint protegido | 401 | `authentication required` |

### Parametros

| Situacion | Codigo | Mensaje |
|-----------|--------|---------|
| Falta lat/lon | 400 | `lat query parameter is required` |
| lat/lon invalido | 400 | `lat must be a valid number` |
| Radio negativo | 400 | `radius must be a positive number` |
| Radio > 50km | 400 | `radius must not exceed 50000 metres` |
| ID invalido | 400 | `id must be a positive integer` |
| Falta device_id | 400 | `device_id query parameter is required` |
| route_id invalido | 400 | `route_id must be a valid integer` |
| stop_id invalido | 400 | `stop_id must be a positive integer` |

### Recursos no encontrados

| Recurso | Codigo | Mensaje |
|---------|--------|---------|
| Parada | 404 | `stop not found` |
| Ruta | 404 | `route not found` |
| Vehiculo (posicion) | 404 | `no position recorded for this vehicle` |
| Alerta | 404 | `alert not found` |
| Usuario | 404 | `user not found` |
| Vehiculo | 404 | `vehicle not found` |
| Asignacion | 404 | `no active assignment` |
| Viaje activo | 404 | `no active trip to end` |

### Conflictos

| Situacion | Codigo | Mensaje |
|-----------|--------|---------|
| Rating duplicado | 409 | `device has already rated this trip` |
| Sin asignacion (posicion) | 409 | `no active vehicle assignment` |
| Viaje ya activo | 409 | `an active trip already exists` |
| Vehiculo sin ruta | 409 | `vehicle has no route assigned` |

### Subida de archivos

| Situacion | Codigo | Mensaje |
|-----------|--------|---------|
| Sin archivo | 400 | `missing or invalid 'file' field` |
| Tipo no soportado | 400 | `unsupported file type "..."; allowed: JPEG, PNG, WebP` |
| Muy grande | 413 | `file must not exceed 5 MB` |

---

## Manejo de Errores en Android (Kotlin)

```kotlin
// Modelo de error
data class ApiError(val error: String)

// Extension para extraer el mensaje de error
fun Response<*>.errorMessage(): String {
    return try {
        val body = errorBody()?.string() ?: return "Error desconocido"
        Gson().fromJson(body, ApiError::class.java).error
    } catch (e: Exception) {
        "Error desconocido"
    }
}

// Uso
val response = apiService.getStop(1)
if (!response.isSuccessful) {
    when (response.code()) {
        400 -> showError("Solicitud invalida: ${response.errorMessage()}")
        401 -> navigateToLogin()
        404 -> showError("No encontrado")
        409 -> showError(response.errorMessage())
        413 -> showError("Archivo muy grande")
        else -> showError("Error del servidor")
    }
}
```
