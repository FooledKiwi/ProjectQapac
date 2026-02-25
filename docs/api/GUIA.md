# Qapac API - Guia para el Equipo Movil

## Descripcion General

API REST para la aplicacion de transporte publico de **Cajamarca, Peru**.
Permite consultar rutas, paradas, posiciones de buses en tiempo real, y gestionar viajes.

**Base URL**: `http://localhost:8080` (desarrollo) | `https://api.qapac.pe` (produccion)

**Prefijo**: Todos los endpoints estan bajo `/api/v1/`

---

## Indice

1. [Arquitectura para la App Movil](#arquitectura-para-la-app-movil)
2. [Autenticacion](autenticacion.md)
3. [Endpoints Publicos](endpoints-publicos.md) (sin login)
4. [Endpoints de Conductor](endpoints-conductor.md) (requiere JWT)
5. [Endpoints de Admin](endpoints-admin.md) (requiere JWT admin)
6. [Errores y Codigos HTTP](errores-y-codigos.md)
7. [Datos de Prueba](datos-prueba.md)
8. [Modelo de Base de Datos](DIAGRAM.md) (diagrama + explicacion de diseno)
9. [Especificacion OpenAPI](openapi.yaml)

---

## Arquitectura para la App Movil

### Usuarios anonimos (pasajeros)

Los usuarios normales de la app **NO necesitan crear cuenta ni iniciar sesion**.
Todas las funciones de pasajero son publicas:

- Ver rutas y paradas
- Ver posiciones de buses en tiempo real
- Buscar paradas y vehiculos cercanos
- Calcular ruta peatonal a una parada
- Calificar viajes (identificados por `device_id`)
- Guardar rutas favoritas (identificados por `device_id`)

El `device_id` es el UUID unico del dispositivo Android. Se usa para identificar
al usuario anonimo en calificaciones y favoritos.

### Conductores y administradores

Solo los conductores y administradores necesitan login. Sus cuentas son creadas
por un administrador a traves de la API de admin.

### Tiempo real (polling)

No hay WebSocket. Para mostrar posiciones de buses en tiempo real, la app debe
hacer **polling** periodico:

```
GET /api/v1/routes/{id}/vehicles    (cada 5-10 segundos)
GET /api/v1/vehicles/{id}/position  (para un vehiculo especifico)
GET /api/v1/vehicles/nearby         (vehiculos cercanos al usuario)
```

### Polylines (recorridos en mapa)

Los recorridos se envian como **Google Encoded Polyline** (precision 1e-5).
Para decodificar en Android:

```kotlin
// build.gradle.kts
implementation("com.google.maps.android:android-maps-utils:3.8.2")

// Codigo
import com.google.maps.android.PolyUtil
import com.google.android.gms.maps.model.PolylineOptions

val points = PolyUtil.decode(response.polyline)
googleMap.addPolyline(
    PolylineOptions()
        .addAll(points)
        .width(8f)
        .color(Color.BLUE)
)
```

---

## Resumen de Endpoints

### Publicos (sin autenticacion) - 16 endpoints

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| `GET` | `/health` | Estado del servidor |
| `GET` | `/api/v1/stops/nearby` | Paradas cercanas |
| `GET` | `/api/v1/stops/:id` | Detalle de parada + ETA |
| `GET` | `/api/v1/routes` | Listar rutas |
| `GET` | `/api/v1/routes/:id` | Detalle de ruta |
| `GET` | `/api/v1/routes/:id/vehicles` | Vehiculos de ruta con GPS |
| `GET` | `/api/v1/routes/to-stop` | Ruta peatonal a parada |
| `GET` | `/api/v1/vehicles/nearby` | Vehiculos cercanos |
| `GET` | `/api/v1/vehicles/:id/position` | Posicion GPS de vehiculo |
| `GET` | `/api/v1/alerts` | Listar alertas |
| `GET` | `/api/v1/alerts/:id` | Detalle de alerta |
| `POST` | `/api/v1/ratings` | Calificar viaje |
| `GET` | `/api/v1/favorites` | Listar favoritos |
| `POST` | `/api/v1/favorites` | Agregar favorito |
| `DELETE` | `/api/v1/favorites` | Eliminar favorito |
| `GET` | `/api/v1/uploads/images/:filename` | Descargar imagen |

### Autenticacion - 3 endpoints

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| `POST` | `/api/v1/auth/login` | Iniciar sesion |
| `POST` | `/api/v1/auth/refresh` | Renovar tokens |
| `POST` | `/api/v1/auth/logout` | Cerrar sesion |

### Conductor (JWT driver\|admin) - 7 endpoints

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| `POST` | `/api/v1/driver/position` | Reportar posicion GPS |
| `GET` | `/api/v1/driver/profile` | Ver perfil |
| `PUT` | `/api/v1/driver/profile` | Actualizar perfil |
| `GET` | `/api/v1/driver/assignment` | Ver asignacion actual |
| `POST` | `/api/v1/driver/trips/start` | Iniciar viaje |
| `POST` | `/api/v1/driver/trips/end` | Finalizar viaje |
| `POST` | `/api/v1/driver/uploads/images` | Subir imagen |

### Admin (JWT admin) - 25 endpoints

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| `POST` | `/api/v1/admin/users` | Crear usuario |
| `GET` | `/api/v1/admin/users` | Listar usuarios |
| `GET` | `/api/v1/admin/users/:id` | Detalle de usuario |
| `PUT` | `/api/v1/admin/users/:id` | Actualizar usuario |
| `DELETE` | `/api/v1/admin/users/:id` | Desactivar usuario |
| `POST` | `/api/v1/admin/stops` | Crear parada |
| `GET` | `/api/v1/admin/stops` | Listar paradas |
| `GET` | `/api/v1/admin/stops/:id` | Detalle de parada |
| `PUT` | `/api/v1/admin/stops/:id` | Actualizar parada |
| `DELETE` | `/api/v1/admin/stops/:id` | Desactivar parada |
| `POST` | `/api/v1/admin/routes` | Crear ruta |
| `GET` | `/api/v1/admin/routes` | Listar rutas |
| `GET` | `/api/v1/admin/routes/:id` | Detalle de ruta |
| `PUT` | `/api/v1/admin/routes/:id` | Actualizar ruta |
| `DELETE` | `/api/v1/admin/routes/:id` | Desactivar ruta |
| `PUT` | `/api/v1/admin/routes/:id/stops` | Reemplazar paradas de ruta |
| `PUT` | `/api/v1/admin/routes/:id/shape` | Actualizar geometria de ruta |
| `POST` | `/api/v1/admin/vehicles` | Registrar vehiculo |
| `GET` | `/api/v1/admin/vehicles` | Listar vehiculos |
| `GET` | `/api/v1/admin/vehicles/:id` | Detalle de vehiculo |
| `PUT` | `/api/v1/admin/vehicles/:id` | Actualizar vehiculo |
| `POST` | `/api/v1/admin/vehicles/:id/assign` | Asignar conductor |
| `POST` | `/api/v1/admin/alerts` | Crear alerta |
| `DELETE` | `/api/v1/admin/alerts/:id` | Eliminar alerta |
| `POST` | `/api/v1/admin/uploads/images` | Subir imagen |

---

## Configuracion del Servidor

Variables de entorno:

| Variable | Requerida | Default | Descripcion |
|----------|-----------|---------|-------------|
| `DB_DSN` | Si | - | Cadena de conexion PostgreSQL |
| `JWT_SECRET` | Si | - | Clave para firmar tokens JWT (HS256) |
| `GOOGLE_API_KEY` | No | - | API key para Google Routes API v2 |
| `PORT` | No | `8080` | Puerto del servidor |
| `ACCESS_TOKEN_TTL` | No | `15m` | Duracion del access token |
| `REFRESH_TOKEN_TTL` | No | `168h` | Duracion del refresh token (7 dias) |
| `UPLOAD_DIR` | No | `./uploads/images` | Directorio para imagenes subidas |

---

## Inicio Rapido

```bash
# 1. Iniciar la base de datos
docker compose up -d postgres

# 2. Configurar variables de entorno
export DB_DSN="postgres://qapac:qapac@localhost:5432/qapac?sslmode=disable"
export JWT_SECRET="mi-clave-secreta-para-desarrollo"

# 3. Ejecutar el servidor
cd api/qapac-api
go run ./cmd/server

# 4. Verificar que funciona
curl http://localhost:8080/health
# {"status":"ok"}

# 5. Login como admin
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

---

Para documentacion detallada de cada grupo de endpoints, ver los archivos individuales
en este directorio o consultar la [especificacion OpenAPI](openapi.yaml).
