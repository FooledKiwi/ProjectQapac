# Plan de Implementacion API - Conciliacion con App Movil

> Generado: 2026-02-24
> Estado: Pendiente de implementacion

## Principios de Diseno

- **Usuarios normales son anonimos** — no requieren login para navegar rutas, paraderos, ETAs o rastrear buses.
- **Auth (JWT) solo para conductores y administradores** — gestionados via API de admin.
- **REST polling** para posiciones de bus en tiempo real (sin WebSocket).
- **Almacenamiento local en disco** para archivos (fotos de perfil, fotos de alertas).
- **Ciudad objetivo: Cajamarca** — los datos semilla deben reemplazarse.

---

## Endpoints Existentes (Ya Implementados)

| Endpoint | Proposito | Estado |
|---|---|---|
| `GET /health` | Health check | Listo |
| `GET /api/v1/stops/nearby` | Paraderos cercanos por radio | Listo |
| `GET /api/v1/stops/:id` | Detalle de paradero + ETA | Listo |
| `GET /api/v1/routes/to-stop` | Direcciones de manejo a un paradero | Listo |

---

## Fase 1: Migracion de Schema de Base de Datos

Nuevas tablas necesarias (ademas de las existentes `stops`, `routes`, `route_stops`, `route_shapes`):

### Tabla `users`
Conductores y administradores.

| Columna | Tipo | Constraints |
|---|---|---|
| `id` | `SERIAL` | PRIMARY KEY |
| `username` | `VARCHAR(100)` | NOT NULL, UNIQUE |
| `password_hash` | `VARCHAR(255)` | NOT NULL |
| `full_name` | `VARCHAR(255)` | NOT NULL |
| `phone` | `VARCHAR(20)` | |
| `role` | `VARCHAR(20)` | NOT NULL, CHECK (role IN ('driver', 'admin')) |
| `profile_image_path` | `VARCHAR(500)` | |
| `active` | `BOOLEAN` | DEFAULT true |
| `created_at` | `TIMESTAMP` | DEFAULT NOW() |
| `updated_at` | `TIMESTAMP` | DEFAULT NOW() |

### Tabla `vehicles`
Flota de buses.

| Columna | Tipo | Constraints |
|---|---|---|
| `id` | `SERIAL` | PRIMARY KEY |
| `plate_number` | `VARCHAR(20)` | NOT NULL, UNIQUE |
| `route_id` | `INT` | REFERENCES routes(id) |
| `status` | `VARCHAR(20)` | DEFAULT 'inactive', CHECK (status IN ('active', 'inactive', 'maintenance')) |
| `created_at` | `TIMESTAMP` | DEFAULT NOW() |

### Tabla `vehicle_assignments`
Vincula vehiculo con conductor + cobrador.

| Columna | Tipo | Constraints |
|---|---|---|
| `id` | `SERIAL` | PRIMARY KEY |
| `vehicle_id` | `INT` | NOT NULL, REFERENCES vehicles(id) |
| `driver_id` | `INT` | NOT NULL, REFERENCES users(id) |
| `collector_id` | `INT` | REFERENCES users(id) |
| `assigned_at` | `TIMESTAMP` | DEFAULT NOW() |
| `active` | `BOOLEAN` | DEFAULT true |
| UNIQUE | | `(vehicle_id)` WHERE active = true |

### Tabla `vehicle_positions`
Ultima posicion GPS por vehiculo.

| Columna | Tipo | Constraints |
|---|---|---|
| `id` | `SERIAL` | PRIMARY KEY |
| `vehicle_id` | `INT` | NOT NULL, REFERENCES vehicles(id), UNIQUE |
| `geom` | `GEOMETRY(POINT, 4326)` | NOT NULL |
| `heading` | `FLOAT` | |
| `speed` | `FLOAT` | |
| `recorded_at` | `TIMESTAMP` | NOT NULL |

### Tabla `trips`
Viajes completados/activos.

| Columna | Tipo | Constraints |
|---|---|---|
| `id` | `SERIAL` | PRIMARY KEY |
| `vehicle_id` | `INT` | NOT NULL, REFERENCES vehicles(id) |
| `route_id` | `INT` | NOT NULL, REFERENCES routes(id) |
| `driver_id` | `INT` | NOT NULL, REFERENCES users(id) |
| `started_at` | `TIMESTAMP` | NOT NULL, DEFAULT NOW() |
| `ended_at` | `TIMESTAMP` | |
| `status` | `VARCHAR(20)` | DEFAULT 'active', CHECK (status IN ('active', 'completed', 'cancelled')) |

### Tabla `alerts`
Notificaciones de cambios de ruta.

| Columna | Tipo | Constraints |
|---|---|---|
| `id` | `SERIAL` | PRIMARY KEY |
| `title` | `VARCHAR(255)` | NOT NULL |
| `description` | `TEXT` | |
| `route_id` | `INT` | REFERENCES routes(id) |
| `vehicle_plate` | `VARCHAR(20)` | |
| `image_path` | `VARCHAR(500)` | |
| `created_by` | `INT` | REFERENCES users(id) |
| `created_at` | `TIMESTAMP` | DEFAULT NOW() |

### Tabla `ratings`
Calificaciones de viajes por usuarios anonimos.

| Columna | Tipo | Constraints |
|---|---|---|
| `id` | `SERIAL` | PRIMARY KEY |
| `trip_id` | `INT` | NOT NULL, REFERENCES trips(id) |
| `rating` | `SMALLINT` | NOT NULL, CHECK (rating BETWEEN 1 AND 5) |
| `device_id` | `VARCHAR(255)` | NOT NULL |
| `created_at` | `TIMESTAMP` | DEFAULT NOW() |
| UNIQUE | | `(trip_id, device_id)` |

### Tabla `favorites`
Rutas favoritas por usuarios anonimos (identificados por device_id).

| Columna | Tipo | Constraints |
|---|---|---|
| `id` | `SERIAL` | PRIMARY KEY |
| `device_id` | `VARCHAR(255)` | NOT NULL |
| `route_id` | `INT` | NOT NULL, REFERENCES routes(id) |
| `created_at` | `TIMESTAMP` | DEFAULT NOW() |
| UNIQUE | | `(device_id, route_id)` |

### Tabla `refresh_tokens`
Seguimiento de tokens JWT de refresco.

| Columna | Tipo | Constraints |
|---|---|---|
| `id` | `SERIAL` | PRIMARY KEY |
| `token_hash` | `VARCHAR(255)` | NOT NULL, UNIQUE |
| `user_id` | `INT` | NOT NULL, REFERENCES users(id) |
| `expires_at` | `TIMESTAMP` | NOT NULL |
| `revoked` | `BOOLEAN` | DEFAULT false |
| `created_at` | `TIMESTAMP` | DEFAULT NOW() |

---

## Fase 2: Autenticacion (Solo Conductores/Admins)

| # | Metodo | Ruta | Proposito | Request Body | Response Body |
|---|---|---|---|---|---|
| 1 | `POST` | `/api/v1/auth/login` | Login conductor/admin | `{username, password}` | `{access_token, refresh_token, user: {id, username, full_name, role}}` |
| 2 | `POST` | `/api/v1/auth/refresh` | Refrescar access token | `{refresh_token}` | `{access_token, refresh_token}` |
| 3 | `POST` | `/api/v1/auth/logout` | Revocar refresh token | `{refresh_token}` | `204 No Content` |

### Middleware JWT
- Validar `Authorization: Bearer <access_token>` en rutas protegidas.
- Access token TTL: 15 minutos.
- Refresh token TTL: 7 dias.
- Almacenar hash del refresh token en tabla `refresh_tokens`.

---

## Fase 3: Endpoints de Admin (Protegidos, rol admin)

### Gestion de Usuarios (Conductores/Admins)

| # | Metodo | Ruta | Proposito | Request Body |
|---|---|---|---|---|
| 4 | `POST` | `/api/v1/admin/users` | Crear cuenta conductor/admin | `{username, password, full_name, phone, role}` |
| 5 | `GET` | `/api/v1/admin/users` | Listar conductores/admins | Query: `?role=driver&active=true` |
| 6 | `GET` | `/api/v1/admin/users/:id` | Detalle de usuario | |
| 7 | `PUT` | `/api/v1/admin/users/:id` | Actualizar usuario | `{full_name, phone, role, active}` |
| 8 | `DELETE` | `/api/v1/admin/users/:id` | Desactivar usuario (soft delete) | |

### Gestion de Vehiculos

| # | Metodo | Ruta | Proposito | Request Body |
|---|---|---|---|---|
| 9 | `POST` | `/api/v1/admin/vehicles` | Registrar vehiculo | `{plate_number, route_id}` |
| 10 | `GET` | `/api/v1/admin/vehicles` | Listar vehiculos | Query: `?route_id=X&status=active` |
| 11 | `GET` | `/api/v1/admin/vehicles/:id` | Detalle de vehiculo | |
| 12 | `PUT` | `/api/v1/admin/vehicles/:id` | Actualizar vehiculo | `{plate_number, route_id, status}` |
| 13 | `POST` | `/api/v1/admin/vehicles/:id/assign` | Asignar conductor+cobrador | `{driver_id, collector_id}` |

### Gestion de Alertas (Admin)

| # | Metodo | Ruta | Proposito | Request Body |
|---|---|---|---|---|
| 14 | `POST` | `/api/v1/admin/alerts` | Crear alerta | `{title, description, route_id, vehicle_plate, image_path}` |
| 15 | `DELETE` | `/api/v1/admin/alerts/:id` | Eliminar alerta | |

---

## Fase 4: Endpoints Publicos (Sin Auth - Para Usuarios de la App)

### Rutas

| # | Metodo | Ruta | Proposito | Response |
|---|---|---|---|---|
| 16 | `GET` | `/api/v1/routes` | Listar rutas activas | `[{id, name, status, vehicle_count}]` |
| 17 | `GET` | `/api/v1/routes/:id` | Detalle de ruta | `{id, name, stops: [...], vehicles: [{plate, driver, collector, status}], shape_polyline}` |
| 18 | `GET` | `/api/v1/routes/:id/vehicles` | Vehiculos activos en ruta | `[{id, plate, driver_name, collector_name, status, position: {lat, lon}, eta}]` |

### Vehiculos / Posiciones

| # | Metodo | Ruta | Proposito | Response |
|---|---|---|---|---|
| 19 | `GET` | `/api/v1/vehicles/:id/position` | Ultima posicion de un vehiculo | `{lat, lon, heading, speed, recorded_at}` |
| 20 | `GET` | `/api/v1/vehicles/nearby` | Vehiculos cerca de un punto | Query: `?lat=X&lon=Y&radius=Z` -> `[{id, plate, route_name, lat, lon}]` |

### Alertas

| # | Metodo | Ruta | Proposito | Response |
|---|---|---|---|---|
| 21 | `GET` | `/api/v1/alerts` | Listar alertas recientes | Query: `?route_id=X` -> `[{id, title, description, route_name, vehicle_plate, image_url, created_at}]` |
| 22 | `GET` | `/api/v1/alerts/:id` | Detalle de alerta | `{id, title, description, route_name, vehicle_plate, image_url, created_at}` |

### Calificaciones

| # | Metodo | Ruta | Proposito | Request Body |
|---|---|---|---|---|
| 23 | `POST` | `/api/v1/ratings` | Enviar calificacion | `{trip_id, rating, device_id}` |

### Favoritos (por device_id, anonimo)

| # | Metodo | Ruta | Proposito | Notas |
|---|---|---|---|---|
| 24 | `GET` | `/api/v1/favorites` | Obtener favoritos | Query: `?device_id=X` |
| 25 | `POST` | `/api/v1/favorites` | Agregar favorito | `{device_id, route_id}` |
| 26 | `DELETE` | `/api/v1/favorites` | Eliminar favorito | `{device_id, route_id}` |

---

## Fase 5: Endpoints de Conductor (Protegidos, rol driver)

| # | Metodo | Ruta | Proposito | Request/Response |
|---|---|---|---|---|
| 27 | `POST` | `/api/v1/driver/position` | Reportar posicion GPS | Request: `{lat, lon, heading, speed}` |
| 28 | `GET` | `/api/v1/driver/profile` | Obtener perfil propio | Response: `{id, username, full_name, phone, profile_image_url}` |
| 29 | `PUT` | `/api/v1/driver/profile` | Actualizar perfil propio | Request: `{full_name, phone}` |
| 30 | `GET` | `/api/v1/driver/assignment` | Obtener asignacion actual | Response: `{vehicle: {plate, route_name}, schedule}` |
| 31 | `POST` | `/api/v1/driver/trips/start` | Iniciar viaje | Response: `{trip_id, route, vehicle}` |
| 32 | `POST` | `/api/v1/driver/trips/end` | Finalizar viaje actual | Response: `204 No Content` |

---

## Fase 6: Carga de Archivos

| # | Metodo | Ruta | Proposito | Notas |
|---|---|---|---|---|
| 33 | `POST` | `/api/v1/uploads/images` | Subir imagen | Multipart form, devuelve `{filename, url}` |
| 34 | `GET` | `/api/v1/uploads/images/:filename` | Servir imagen | Archivo estatico desde disco |

### Configuracion de Almacenamiento
- Directorio: `./uploads/images/`
- Tamano maximo: 5MB
- Formatos permitidos: JPEG, PNG, WebP
- Nombres generados con UUID para evitar colisiones

---

## Fase 7: Datos Semilla de Cajamarca

- Reemplazar paraderos de Lima con paraderos de Cajamarca.
- Crear rutas de buses de Cajamarca (ej: P13, etc. segun la app movil).
- Actualizar polylines de route_shapes con geometria de Cajamarca.
- Crear vehiculos de ejemplo con placas y asignaciones.

---

## Orden de Implementacion Sugerido

| Prioridad | Tarea | Dependencias | Complejidad |
|---|---|---|---|
| **1** | Migracion DB (nuevas tablas) | Ninguna | Media |
| **2** | Auth (login/refresh/logout + middleware JWT) | Fase 1 | Alta |
| **3** | Admin CRUD usuarios/vehiculos | Fases 1, 2 | Media |
| **4** | Rutas publicas (listar, detalle) #16, #17 | Fase 1 | Baja |
| **5** | Reporte de posicion del conductor #27 | Fases 1, 2 | Media |
| **6** | Posiciones publicas de vehiculos #18, #19, #20 | Fases 1, 5 | Media |
| **7** | Alertas #21, #22, #14, #15 | Fase 1 | Baja |
| **8** | Favoritos #24, #25, #26 | Fase 1 | Baja |
| **9** | Calificaciones #23 | Fase 1 | Baja |
| **10** | Perfil y viajes del conductor #28-32 | Fases 1, 2 | Media |
| **11** | Carga de archivos #33, #34 | Ninguna | Media |
| **12** | Datos semilla de Cajamarca | Fase 1 | Baja |

---

## Dependencias Go Nuevas Necesarias

| Paquete | Proposito |
|---|---|
| `golang.org/x/crypto/bcrypt` | Hash de contrasenas |
| `github.com/golang-jwt/jwt/v5` | Generacion/validacion JWT |
| `github.com/google/uuid` | Generacion UUID para nombres de archivo |

---

## Notas de Compatibilidad con App Movil

- La app usa `device_id` (Android ID o similar) para identificar usuarios anonimos -> usado en favoritos y ratings.
- La app espera datos en espanol (nombres de rutas, estados, etc.).
- La app muestra: nombre de ruta, placa de vehiculo, conductor asignado, cobrador, ETA, ubicacion actual.
- Estados de vehiculo en la app: "Activo", "Inactivo", "Pendiente" — mapear desde los valores de la DB.
- La app muestra precio del viaje (ej: "S/ 02.00") — considerar agregar campo `price` a `routes` en el futuro.
- La app tiene un trip planner (fragment vacio) — dejado fuera de este plan, implementar despues.
