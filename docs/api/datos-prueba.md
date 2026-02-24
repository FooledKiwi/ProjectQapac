# Datos de Prueba (Seed Data)

La base de datos se inicializa con datos de prueba para Cajamarca. Estos datos
se cargan automaticamente al ejecutar las migraciones.

---

## Usuarios

| ID | Username | Password | Rol | Nombre |
|----|----------|----------|-----|--------|
| 1 | `admin` | `admin123` | admin | Administrador Qapac |
| 2 | `conductor1` | `driver123` | driver | Carlos Quispe |
| 3 | `conductor2` | `driver123` | driver | Maria Lopez |

> Las contrasenas estan hasheadas con bcrypt en la BD.

---

## Rutas

| ID | Nombre |
|----|--------|
| 1 | R01 - Centro a Baños del Inca |
| 2 | R02 - Fonavi a Centro |
| 3 | R03 - Mollepampa a UNC |
| 4 | R04 - Samanacruz a Centro |
| 5 | R05 - Baños del Inca a Llacanora |

---

## Paradas (28 en total)

Las paradas usan coordenadas reales de Cajamarca. Ejemplos:

| ID | Nombre | Lat | Lon |
|----|--------|-----|-----|
| 1 | Plaza de Armas | -7.1637 | -78.5003 |
| 2 | Mercado Central | -7.1612 | -78.5128 |
| 3 | Av. Independencia | -7.1580 | -78.5120 |
| 4 | Ovalo Musical | -7.1560 | -78.5070 |
| 5 | Complejo Qhapaq Nan | -7.1510 | -78.4950 |
| ... | ... | ... | ... |

> Las coordenadas completas estan en el archivo de migracion
> `api/qapac-api/internal/migrations/009_seed_cajamarca.sql`

---

## Vehiculos (8 en total)

| ID | Placa | Ruta | Estado |
|----|-------|------|--------|
| 1 | AAA-111 | R01 | active |
| 2 | BBB-222 | R01 | active |
| 3 | CCC-333 | R02 | active |
| 4 | DDD-444 | R02 | inactive |
| 5 | EEE-555 | R03 | active |
| 6 | FFF-666 | R03 | maintenance |
| 7 | GGG-777 | R04 | active |
| 8 | HHH-888 | R05 | active |

---

## Asignaciones Activas

| Vehiculo | Conductor | Cobrador |
|----------|-----------|----------|
| AAA-111 (ID 1) | conductor1 (ID 2) | conductor2 (ID 3) |
| CCC-333 (ID 3) | conductor2 (ID 3) | - |

---

## Posiciones GPS de Ejemplo

| Vehiculo | Lat | Lon | Heading | Speed |
|----------|-----|-----|---------|-------|
| AAA-111 | -7.1640 | -78.5010 | 180.0 | 25.0 |
| CCC-333 | -7.1600 | -78.5100 | 90.0 | 15.0 |

---

## Centro de Cajamarca (para pruebas)

Coordenadas utiles para probar los endpoints de busqueda:

| Lugar | Lat | Lon |
|-------|-----|-----|
| Plaza de Armas (centro) | -7.1637 | -78.5003 |
| Baños del Inca | -7.1614 | -78.4688 |
| UNC (universidad) | -7.1720 | -78.4930 |
| Terminal terrestre | -7.1555 | -78.5240 |

**Ejemplo de prueba con curl:**

```bash
# Paradas cercanas a la Plaza de Armas (radio 1km)
curl "http://localhost:8080/api/v1/stops/nearby?lat=-7.1637&lon=-78.5003&radius=1000"

# Vehiculos cercanos
curl "http://localhost:8080/api/v1/vehicles/nearby?lat=-7.1637&lon=-78.5003&radius=5000"

# Listar rutas
curl http://localhost:8080/api/v1/routes

# Detalle de ruta con paradas y vehiculos
curl http://localhost:8080/api/v1/routes/1

# Vehiculos de la ruta 1 con posicion GPS
curl http://localhost:8080/api/v1/routes/1/vehicles

# Login como conductor
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"conductor1","password":"driver123"}'

# Reportar posicion (con token)
TOKEN="<access_token del login>"
curl -X POST http://localhost:8080/api/v1/driver/position \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"lat":-7.1645,"lon":-78.5015,"heading":270.0,"speed":30.0}'
```
