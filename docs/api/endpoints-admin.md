# Endpoints de Admin

Estos endpoints requieren un **JWT valido** con rol `admin`.

**Header requerido:**
```
Authorization: Bearer <access_token>
```

---

## Gestion de Usuarios

### POST /api/v1/admin/users

Crea un nuevo usuario (conductor o administrador).

**Request Body:**

```json
{
  "username": "conductor3",
  "password": "miPassword123",
  "full_name": "Juan Perez",
  "phone": "987654321",
  "role": "driver"
}
```

| Campo | Tipo | Requerido | Validacion | Descripcion |
|-------|------|-----------|------------|-------------|
| `username` | string | Si | - | Nombre de usuario unico |
| `password` | string | Si | min 6 chars | Contrasena |
| `full_name` | string | Si | - | Nombre completo |
| `phone` | string | No | - | Telefono |
| `role` | string | Si | `driver` o `admin` | Rol del usuario |

**Response 201:**

```json
{
  "id": 4,
  "username": "conductor3",
  "full_name": "Juan Perez",
  "phone": "987654321",
  "role": "driver",
  "active": true
}
```

---

### GET /api/v1/admin/users

Lista usuarios con filtros opcionales.

**Query Parameters:**

| Parametro | Tipo | Requerido | Default | Descripcion |
|-----------|------|-----------|---------|-------------|
| `role` | string | No | - | Filtrar: `driver` o `admin` |
| `active` | string | No | `true` | Filtrar por estado: `true` o `false` |

**Ejemplo:**

```http
GET /api/v1/admin/users?role=driver&active=true
```

**Response 200:**

```json
[
  {
    "id": 2,
    "username": "conductor1",
    "full_name": "Carlos Quispe",
    "phone": "976543210",
    "role": "driver",
    "active": true
  }
]
```

---

### GET /api/v1/admin/users/:id

Obtiene el detalle completo de un usuario.

**Response 200:**

```json
{
  "id": 2,
  "username": "conductor1",
  "full_name": "Carlos Quispe",
  "phone": "976543210",
  "role": "driver",
  "profile_image_path": "",
  "active": true,
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-15T10:00:00Z"
}
```

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | ID invalido |
| 404 | Usuario no existe |

---

### PUT /api/v1/admin/users/:id

Actualiza campos de un usuario. Actualizacion parcial: solo se modifican
los campos enviados (los campos vacios o nulos se ignoran).

**Request Body:**

```json
{
  "full_name": "Carlos Quispe Mendoza",
  "role": "admin",
  "active": false
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `full_name` | string | No | Nombre completo |
| `phone` | string | No | Telefono |
| `role` | string | No | `driver` o `admin` |
| `profile_image_path` | string | No | Ruta de imagen de perfil |
| `active` | bool | No | `true` o `false` para activar/desactivar |

**Response 200:**

```json
{
  "id": 2,
  "username": "conductor1",
  "full_name": "Carlos Quispe Mendoza",
  "phone": "976543210",
  "role": "admin",
  "active": false
}
```

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | ID invalido o rol no valido |
| 404 | Usuario no existe |

---

### DELETE /api/v1/admin/users/:id

Desactiva un usuario (soft delete). El usuario no podra iniciar sesion pero
sus datos se conservan en la base de datos.

**Response:** `204 No Content` (sin cuerpo)

---

## Gestion de Vehiculos

### POST /api/v1/admin/vehicles

Registra un nuevo vehiculo.

**Request Body:**

```json
{
  "plate_number": "XYZ-789",
  "route_id": 1
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `plate_number` | string | Si | Numero de placa |
| `route_id` | int32 | No | ID de ruta (puede asignarse despues) |

> Los vehiculos nuevos se crean con estado `inactive`.

**Response 201:**

```json
{
  "id": 9,
  "plate_number": "XYZ-789",
  "route_id": 1,
  "status": "inactive"
}
```

---

### GET /api/v1/admin/vehicles

Lista vehiculos con filtros opcionales.

**Query Parameters:**

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `route_id` | int32 | No | Filtrar por ruta |
| `status` | string | No | Filtrar: `active`, `inactive`, `maintenance` |

**Ejemplo:**

```http
GET /api/v1/admin/vehicles?status=active&route_id=1
```

**Response 200:**

```json
[
  {
    "id": 1,
    "plate_number": "ABC-123",
    "route_id": 1,
    "status": "active"
  }
]
```

---

### GET /api/v1/admin/vehicles/:id

Detalle de un vehiculo con su asignacion actual (conductor y cobrador).

**Response 200:**

```json
{
  "id": 1,
  "plate_number": "ABC-123",
  "route_id": 1,
  "status": "active",
  "created_at": "2025-01-01T00:00:00Z",
  "assignment": {
    "driver_id": 2,
    "collector_id": 3,
    "assigned_at": "2025-01-15T06:00:00Z"
  }
}
```

> **assignment** es `null` si el vehiculo no tiene conductor asignado.

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | ID invalido |
| 404 | Vehiculo no existe |

---

### PUT /api/v1/admin/vehicles/:id

Actualiza campos de un vehiculo (parcial).

**Request Body:**

```json
{
  "plate_number": "XYZ-999",
  "route_id": 2,
  "status": "active"
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `plate_number` | string | No | Nueva placa |
| `route_id` | int32 | No | Nueva ruta |
| `status` | string | No | `active`, `inactive`, `maintenance` |

**Response 200:**

```json
{
  "id": 1,
  "plate_number": "XYZ-999",
  "route_id": 2,
  "status": "active"
}
```

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | ID invalido o estado no valido |
| 404 | Vehiculo no existe |

---

### POST /api/v1/admin/vehicles/:id/assign

Asigna un conductor (y opcionalmente un cobrador) a un vehiculo.

**Request Body:**

```json
{
  "driver_id": 2,
  "collector_id": 3
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `driver_id` | int32 | Si | ID del conductor |
| `collector_id` | int32 | No | ID del cobrador (puede ser nulo) |

**Response 201:**

```json
{
  "id": 1,
  "vehicle_id": 1,
  "driver_id": 2,
  "collector_id": 3,
  "assigned_at": "2025-01-15T06:00:00Z"
}
```

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | Falta driver_id o ID de vehiculo invalido |
| 404 | Vehiculo no existe |

---

## Gestion de Alertas

### POST /api/v1/admin/alerts

Crea una alerta de servicio. El `created_by` se extrae automaticamente del JWT.

**Request Body:**

```json
{
  "title": "Desvio en Av. Independencia",
  "description": "Por obras municipales, la ruta R01 desvia por Jr. Amazonas",
  "route_id": 1,
  "vehicle_plate": "",
  "image_path": ""
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `title` | string | Si | Titulo de la alerta |
| `description` | string | No | Descripcion detallada |
| `route_id` | int32 | No | ID de la ruta afectada |
| `vehicle_plate` | string | No | Placa del vehiculo afectado |
| `image_path` | string | No | Nombre de imagen subida previamente |

> Para adjuntar una imagen, primero subirla con `POST /api/v1/admin/uploads/images`
> y luego usar el `filename` retornado en el campo `image_path`.

**Response 201:**

```json
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
```

---

### DELETE /api/v1/admin/alerts/:id

Elimina una alerta.

**Response:** `204 No Content` (sin cuerpo)

---

## Subida de Imagenes (Admin)

### POST /api/v1/admin/uploads/images

Igual que el endpoint de conductor. Sube una imagen via `multipart/form-data`.

**Request:** `multipart/form-data` con campo `file`

- Tipos permitidos: JPEG, PNG, WebP
- Tamano maximo: 5 MB

**Ejemplo con curl:**

```bash
curl -X POST http://localhost:8080/api/v1/admin/uploads/images \
  -H "Authorization: Bearer <token>" \
  -F "file=@alerta.jpg"
```

**Response 201:**

```json
{
  "filename": "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg",
  "url": "http://localhost:8080/api/v1/uploads/images/a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg"
}
```

**Errores:**

| Codigo | Causa |
|--------|-------|
| 400 | Archivo faltante o tipo no soportado |
| 413 | Archivo excede 5 MB |
