# Modelo de Base de Datos

Este documento contiene el diagrama visual de la base de datos y una explicacion
detallada de cada tabla, relacion y decision de diseno del esquema de Qapac API.

## Indice

1. [Diagrama Visual (DBML)](#diagrama-visual-dbml)
2. [Vision General del Esquema](#vision-general-del-esquema)
3. [PostGIS y Datos Geoespaciales](#postgis-y-datos-geoespaciales)
4. [Grupo 1: Infraestructura de Transporte](#grupo-1-infraestructura-de-transporte)
5. [Grupo 2: Usuarios y Autenticacion](#grupo-2-usuarios-y-autenticacion)
6. [Grupo 3: Flota de Vehiculos](#grupo-3-flota-de-vehiculos)
7. [Grupo 4: Viajes y Operaciones](#grupo-4-viajes-y-operaciones)
8. [Grupo 5: Interaccion de Pasajeros](#grupo-5-interaccion-de-pasajeros)
9. [Grupo 6: Cache de Rendimiento](#grupo-6-cache-de-rendimiento)
10. [Grupo 7: Sistema de Migraciones](#grupo-7-sistema-de-migraciones)
11. [Estrategias de Eliminacion](#estrategias-de-eliminacion)
12. [Indices y Rendimiento](#indices-y-rendimiento)
13. [Sistema de Migraciones Embebidas](#sistema-de-migraciones-embebidas)
14. [Resumen de Estadisticas](#resumen-de-estadisticas)

---

## Diagrama Visual (DBML)

Sintaxis compatible con [dbdiagram.io](https://dbdiagram.io/).
Copiar el bloque de codigo y pegarlo en el editor de dbdiagram.io para visualizar.

```dbml
// =============================================================================
// Qapac API - Diagrama de Base de Datos
// Transporte publico de Cajamarca, Peru
// =============================================================================

// ---------------------------------------------------------------------------
// Infraestructura de transporte
// ---------------------------------------------------------------------------

Table stops {
  id serial [pk, increment]
  name varchar(255) [not null]
  geom geometry(Point,4326) [not null, note: 'PostGIS POINT(lon lat)']
  active boolean [default: true]
  created_at timestamp [default: `NOW()`]

  indexes {
    geom [type: gist, name: 'idx_stops_geom']
  }
}

Table routes {
  id serial [pk, increment]
  name varchar(255) [not null]
  active boolean [default: true]
}

Table route_stops {
  id serial [pk, increment]
  route_id int [not null, ref: > routes.id]
  stop_id int [not null, ref: > stops.id]
  sequence int [not null]

  indexes {
    (route_id, stop_id) [unique]
  }
}

Table route_shapes {
  id serial [pk, increment]
  route_id int [not null, unique, ref: - routes.id, note: 'una geometria por ruta']
  geom geometry(LineString,4326) [not null, note: 'PostGIS LINESTRING del recorrido']
  updated_at timestamp [default: `NOW()`]

  indexes {
    geom [type: gist, name: 'idx_route_shapes_geom']
  }
}

// ---------------------------------------------------------------------------
// Usuarios y autenticacion
// ---------------------------------------------------------------------------

Table users {
  id serial [pk, increment]
  username varchar(100) [not null, unique]
  password_hash varchar(255) [not null]
  full_name varchar(255) [not null]
  phone varchar(20)
  role varchar(20) [not null, note: 'driver | admin']
  profile_image_path varchar(500)
  active boolean [default: true]
  created_at timestamp [default: `NOW()`]
  updated_at timestamp [default: `NOW()`]

  indexes {
    role [name: 'idx_users_role']
    active [name: 'idx_users_active']
  }
}

Table refresh_tokens {
  id serial [pk, increment]
  token_hash varchar(255) [not null, unique]
  user_id int [not null, ref: > users.id]
  expires_at timestamp [not null]
  revoked boolean [default: false]
  created_at timestamp [default: `NOW()`]

  indexes {
    user_id [name: 'idx_refresh_tokens_user_id']
  }

  note: 'ON DELETE CASCADE desde users'
}

// ---------------------------------------------------------------------------
// Flota de vehiculos
// ---------------------------------------------------------------------------

Table vehicles {
  id serial [pk, increment]
  plate_number varchar(20) [not null, unique]
  route_id int [ref: > routes.id, note: 'ON DELETE SET NULL']
  status varchar(20) [default: 'inactive', note: 'active | inactive | maintenance']
  created_at timestamp [default: `NOW()`]

  indexes {
    route_id [name: 'idx_vehicles_route_id']
    status [name: 'idx_vehicles_status']
  }
}

Table vehicle_assignments {
  id serial [pk, increment]
  vehicle_id int [not null, ref: > vehicles.id]
  driver_id int [not null, ref: > users.id]
  collector_id int [ref: > users.id, note: 'ON DELETE SET NULL']
  assigned_at timestamp [default: `NOW()`]
  active boolean [default: true]

  indexes {
    driver_id [name: 'idx_vehicle_assignments_driver']
    collector_id [name: 'idx_vehicle_assignments_collector']
  }

  note: 'Indice parcial UNIQUE en (vehicle_id) WHERE active = true'
}

Table vehicle_positions {
  id serial [pk, increment]
  vehicle_id int [not null, unique, ref: - vehicles.id, note: 'una posicion por vehiculo (upsert)']
  geom geometry(Point,4326) [not null, note: 'PostGIS POINT(lon lat)']
  heading float [note: 'direccion 0-360 grados']
  speed float [note: 'km/h']
  recorded_at timestamp [not null]

  indexes {
    geom [type: gist, name: 'idx_vehicle_positions_geom']
  }

  note: 'ON DELETE CASCADE desde vehicles'
}

// ---------------------------------------------------------------------------
// Viajes y operaciones
// ---------------------------------------------------------------------------

Table trips {
  id serial [pk, increment]
  vehicle_id int [not null, ref: > vehicles.id]
  route_id int [not null, ref: > routes.id]
  driver_id int [not null, ref: > users.id]
  started_at timestamp [not null, default: `NOW()`]
  ended_at timestamp
  status varchar(20) [default: 'active', note: 'active | completed | cancelled']

  indexes {
    vehicle_id [name: 'idx_trips_vehicle_id']
    route_id [name: 'idx_trips_route_id']
    driver_id [name: 'idx_trips_driver_id']
    status [name: 'idx_trips_status']
    started_at [name: 'idx_trips_started_at']
  }

  note: 'ON DELETE CASCADE desde vehicles, routes, users'
}

Table alerts {
  id serial [pk, increment]
  title varchar(255) [not null]
  description text
  route_id int [ref: > routes.id, note: 'ON DELETE SET NULL']
  vehicle_plate varchar(20) [note: 'referencia suave, no FK']
  image_path varchar(500)
  created_by int [ref: > users.id, note: 'ON DELETE SET NULL']
  created_at timestamp [default: `NOW()`]

  indexes {
    route_id [name: 'idx_alerts_route_id']
    (created_at) [name: 'idx_alerts_created_at']
  }
}

// ---------------------------------------------------------------------------
// Interaccion de pasajeros (anonimos por device_id)
// ---------------------------------------------------------------------------

Table ratings {
  id serial [pk, increment]
  trip_id int [not null, ref: > trips.id]
  rating smallint [not null, note: '1 a 5 estrellas']
  device_id varchar(255) [not null]
  created_at timestamp [default: `NOW()`]

  indexes {
    trip_id [name: 'idx_ratings_trip_id']
    (trip_id, device_id) [unique]
  }

  note: 'ON DELETE CASCADE desde trips'
}

Table favorites {
  id serial [pk, increment]
  device_id varchar(255) [not null]
  route_id int [not null, ref: > routes.id]
  created_at timestamp [default: `NOW()`]

  indexes {
    device_id [name: 'idx_favorites_device_id']
    route_id [name: 'idx_favorites_route_id']
    (device_id, route_id) [unique]
  }

  note: 'ON DELETE CASCADE desde routes'
}

// ---------------------------------------------------------------------------
// Cache (tablas UNLOGGED - datos volatiles, se pierden en crash)
// ---------------------------------------------------------------------------

Table stop_eta_cache {
  id serial [pk, increment]
  stop_id int [not null, unique, ref: > stops.id]
  eta_seconds int [not null]
  calc_ts timestamp [default: `NOW()`]
  expires_at timestamp [not null]

  indexes {
    stop_id [name: 'idx_eta_cache_stop_id']
  }

  note: 'UNLOGGED - cache de ETA por parada'
}

Table route_to_stop_cache {
  id serial [pk, increment]
  origin_hash varchar(50) [not null]
  stop_id int [not null, ref: > stops.id]
  polyline text [not null]
  distance_m int [not null]
  duration_s int [not null]
  calc_ts timestamp [default: `NOW()`]
  expires_at timestamp [not null]

  indexes {
    (origin_hash, stop_id) [unique, name: 'idx_route_cache_hash_stop']
  }

  note: 'UNLOGGED - cache de rutas peatonales a paradas'
}

// ---------------------------------------------------------------------------
// Migraciones
// ---------------------------------------------------------------------------

Table schema_migrations {
  version varchar(255) [pk]
  applied_at timestamp [default: `NOW()`]

  note: 'Control de migraciones aplicadas'
}
```

---

## Vision General del Esquema

La base de datos de Qapac esta disenada para soportar una aplicacion de transporte
publico urbano en Cajamarca, Peru. El esquema tiene **16 tablas** (14 regulares +
2 tablas UNLOGGED de cache) que se organizan en los siguientes grupos funcionales:

| Grupo | Tablas | Proposito |
|-------|--------|-----------|
| Infraestructura | `stops`, `routes`, `route_stops`, `route_shapes` | Definen la red de transporte: donde estan las paradas y por donde pasan las rutas |
| Usuarios | `users`, `refresh_tokens` | Cuentas de conductores y administradores, gestion de sesiones JWT |
| Flota | `vehicles`, `vehicle_assignments`, `vehicle_positions` | Registro de buses, asignacion de personal, rastreo GPS en tiempo real |
| Operaciones | `trips`, `alerts` | Viajes activos de los buses, alertas e incidencias |
| Pasajeros | `ratings`, `favorites` | Calificaciones y favoritos de usuarios anonimos (sin cuenta) |
| Cache | `stop_eta_cache`, `route_to_stop_cache` | Datos temporales precalculados para respuestas rapidas |
| Sistema | `schema_migrations` | Control de versiones del esquema de base de datos |

### Motor de base de datos

Se utiliza **PostgreSQL** como motor de base de datos por las siguientes razones:

1. **Extension PostGIS**: PostgreSQL es el unico motor relacional con soporte
   maduro para datos geoespaciales a traves de PostGIS. Esto es fundamental para
   una aplicacion de transporte que necesita calcular distancias, buscar puntos
   cercanos y almacenar geometrias de recorridos.

2. **Rendimiento con datos espaciales**: Los indices GiST (Generalized Search Tree)
   de PostGIS permiten busquedas eficientes de tipo "encontrar las 5 paradas mas
   cercanas a mi ubicacion" sin escanear toda la tabla.

3. **Tablas UNLOGGED**: PostgreSQL soporta tablas sin Write-Ahead Log (WAL) que
   ofrecen escrituras mucho mas rapidas para datos de cache que no necesitan
   durabilidad.

4. **Tipos CHECK y restricciones**: Validaciones a nivel de base de datos para
   asegurar integridad (roles validos, estados de vehiculo, rango de calificaciones).

---

## PostGIS y Datos Geoespaciales

### Que es PostGIS

PostGIS es una extension de PostgreSQL que agrega soporte para objetos geograficos.
Permite almacenar coordenadas GPS, calcular distancias entre puntos, verificar si
un punto esta dentro de un area, y realizar consultas espaciales eficientes.

Se habilita en la primera migracion con:

```sql
CREATE EXTENSION IF NOT EXISTS postgis;
```

### SRID 4326 (WGS84)

Todas las columnas geometricas usan **SRID 4326**, que corresponde al sistema de
coordenadas **WGS84** (World Geodetic System 1984). Este es el mismo sistema que
usan los receptores GPS de los telefonos moviles, Google Maps y practicamente todos
los servicios de mapas del mundo.

En WGS84, una coordenada se expresa como:
- **Longitud** (lon): posicion este-oeste, entre -180 y +180 grados
- **Latitud** (lat): posicion norte-sur, entre -90 y +90 grados

Cajamarca se encuentra aproximadamente en **lon -78.51, lat -7.16**.

**Importante**: En PostGIS, las funciones `ST_Point(lon, lat)` reciben la longitud
primero y la latitud segundo. Esto es contrario a la convencion comun de "latitud,
longitud" que usan muchas APIs (como Google Maps), y es una fuente frecuente de
errores. En nuestro codigo siempre usamos el orden PostGIS: `ST_Point(lon, lat)`.

### Tipos de geometria utilizados

| Tipo | Columna | Tabla | Uso |
|------|---------|-------|-----|
| `POINT(lon, lat)` | `geom` | `stops` | Ubicacion exacta de cada parada |
| `POINT(lon, lat)` | `geom` | `vehicle_positions` | Posicion GPS actual de cada vehiculo |
| `LINESTRING(...)` | `geom` | `route_shapes` | Recorrido completo de una ruta como una linea conectada de puntos |

**POINT** representa una ubicacion unica en el mapa (una parada, un vehiculo).

**LINESTRING** representa una secuencia ordenada de puntos que forman una linea
continua. Se usa para dibujar el recorrido de una ruta de bus en el mapa. Por ejemplo,
una ruta con 7 paradas tendria un LINESTRING con al menos 7 puntos conectados.

### Indices GiST para consultas espaciales

Las tres columnas geometricas tienen indices **GiST** (Generalized Search Tree):

```sql
CREATE INDEX idx_stops_geom             ON stops             USING GIST(geom);
CREATE INDEX idx_route_shapes_geom      ON route_shapes      USING GIST(geom);
CREATE INDEX idx_vehicle_positions_geom ON vehicle_positions  USING GIST(geom);
```

Un indice GiST organiza los datos espaciales en un arbol jerarquico de "cajas
delimitadoras" (bounding boxes). Cuando la aplicacion ejecuta una consulta como
"encontrar las paradas dentro de 500 metros de mi ubicacion", PostgreSQL no necesita
calcular la distancia a cada una de las 28 paradas. En cambio, el indice GiST
descarta rapidamente las regiones del mapa que estan demasiado lejos y solo evalua
las paradas que estan en la zona relevante.

Esto es especialmente importante para las consultas en tiempo real de la aplicacion
movil, como buscar vehiculos cercanos mientras el usuario camina.

### Funciones PostGIS utilizadas en la API

| Funcion | Descripcion | Ejemplo de uso |
|---------|-------------|----------------|
| `ST_SetSRID(ST_Point(lon, lat), 4326)` | Crear un punto con coordenadas GPS | Insertar una nueva parada |
| `ST_DWithin(geom, punto, distancia)` | Verificar si dos geometrias estan dentro de una distancia | Buscar paradas cercanas |
| `ST_Distance(geom1, geom2)` | Calcular distancia entre dos geometrias | Ordenar resultados por cercania |
| `ST_X(geom)`, `ST_Y(geom)` | Extraer longitud y latitud de un punto | Devolver coordenadas en JSON |
| `ST_AsText(geom)` | Convertir geometria a texto legible (WKT) | Devolver recorrido de ruta |
| `ST_MakeLine(ARRAY[...])` | Crear un LINESTRING a partir de un arreglo de puntos | Definir recorrido de ruta |

---

## Grupo 1: Infraestructura de Transporte

Este grupo define la red fisica de transporte: donde estan las paradas, que rutas
existen, por cuales paradas pasa cada ruta (y en que orden), y cual es la geometria
del recorrido de cada ruta.

### Tabla `stops` (Paradas)

```sql
CREATE TABLE stops (
  id         SERIAL PRIMARY KEY,
  name       VARCHAR(255) NOT NULL,
  geom       GEOMETRY(POINT, 4326) NOT NULL,
  active     BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW()
);
```

Cada fila representa una parada fisica de bus en la ciudad. Se almacena su nombre
descriptivo (por ejemplo, "Plaza de Armas", "Hospital Regional de Cajamarca") y su
ubicacion exacta como un punto GPS.

**Decisiones de diseno:**

- **`geom GEOMETRY(POINT, 4326)`**: Se usa el tipo nativo de PostGIS en lugar de
  columnas separadas `latitude`/`longitude` (FLOAT). La ventaja es que PostGIS
  puede crear indices espaciales GiST sobre la columna, permitiendo consultas como
  "paradas dentro de 500 metros" de forma eficiente. Con columnas FLOAT separadas,
  habria que calcular distancias con formulas trigonometricas en cada fila.

- **`active BOOLEAN DEFAULT true`**: Se usa **borrado logico** (soft delete) en
  lugar de eliminar la fila. Cuando un administrador "elimina" una parada, solo
  se marca como `active = false`. Esto preserva la integridad referencial: si hay
  viajes historicos o calificaciones que referencian esa parada, no se pierden.
  Ademas, permite reactivar una parada sin tener que recrearla.

- **`SERIAL PRIMARY KEY`**: Identificador autoincremental. Suficiente para el
  volumen de datos esperado (decenas o cientos de paradas, no millones).

### Tabla `routes` (Rutas)

```sql
CREATE TABLE routes (
  id     SERIAL PRIMARY KEY,
  name   VARCHAR(255) NOT NULL,
  active BOOLEAN DEFAULT true
);
```

Cada fila representa una ruta de transporte publico (por ejemplo, "Ruta 1 - Centro
a Banos del Inca"). La tabla es intencionalmente simple: solo un nombre y un estado
activo/inactivo.

**Decisiones de diseno:**

- **Sin columna de geometria**: La geometria del recorrido se almacena en una tabla
  separada `route_shapes` (ver mas abajo). Esto evita que las consultas de listado
  (`SELECT id, name FROM routes`) arrastren grandes blobs de geometria que no se
  necesitan en la lista.

- **Mismo patron de borrado logico** que `stops`: `active = false` en lugar de DELETE.

### Tabla `route_stops` (Paradas por Ruta)

```sql
CREATE TABLE route_stops (
  id       SERIAL PRIMARY KEY,
  route_id INT NOT NULL REFERENCES routes(id),
  stop_id  INT NOT NULL REFERENCES stops(id),
  sequence INT NOT NULL,
  UNIQUE(route_id, stop_id)
);
```

Esta es una **tabla de union** (join table) que conecta rutas con paradas. Define
por cuales paradas pasa cada ruta y en que orden.

**Decisiones de diseno:**

- **`sequence INT`**: Indica el orden de la parada dentro de la ruta. La parada con
  `sequence = 1` es la primera del recorrido, `sequence = 2` la segunda, y asi
  sucesivamente. Esto permite que la aplicacion muestre las paradas en el orden
  correcto del recorrido.

- **`UNIQUE(route_id, stop_id)`**: Evita que la misma parada aparezca dos veces en
  la misma ruta. Sin embargo, una misma parada puede pertenecer a multiples rutas
  (con diferentes valores de `sequence`). Por ejemplo, "Plaza de Armas" esta en 4
  de las 5 rutas de Cajamarca, porque es un punto central de la ciudad.

- **Sin ON DELETE CASCADE en las FK**: Si se elimina una ruta o parada referenciada,
  PostgreSQL rechazara la operacion. Esto es intencional: se espera que las rutas y
  paradas se desactiven (borrado logico) en lugar de eliminarse fisicamente. Si se
  necesita eliminar fisicamente, primero se deben limpiar las referencias en esta
  tabla.

### Tabla `route_shapes` (Geometria de Ruta)

```sql
CREATE TABLE route_shapes (
  id         SERIAL PRIMARY KEY,
  route_id   INT NOT NULL REFERENCES routes(id) UNIQUE,
  geom       GEOMETRY(LINESTRING, 4326) NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW()
);
```

Almacena el recorrido geografico exacto de cada ruta como una linea conectada de
puntos GPS (LINESTRING).

**Decisiones de diseno:**

- **`route_id UNIQUE`**: Establece una relacion **uno a uno** con `routes`. Cada
  ruta tiene exactamente una geometria de recorrido. Se usa `UNIQUE` en lugar de
  hacer que `route_id` sea la PK para mantener la consistencia del patron `id SERIAL`
  en todas las tablas.

- **Tabla separada de `routes`**: Esta es una decision de rendimiento importante.
  Un LINESTRING tipico contiene decenas o cientos de puntos coordenados y puede
  ocupar varios kilobytes. Si la geometria estuviera en la tabla `routes`, cada
  `SELECT * FROM routes` (listado de rutas) leeria esos datos pesados sin necesidad.
  Al separarlo, el listado de rutas es una lectura ligera y la geometria solo se
  carga cuando se necesita (detalle de una ruta, dibujo en mapa).

- **`updated_at`**: Permite saber cuando se actualizo el recorrido por ultima vez.
  Util para invalidar caches en la aplicacion movil.

---

## Grupo 2: Usuarios y Autenticacion

### Tabla `users` (Usuarios)

```sql
CREATE TABLE users (
  id                 SERIAL PRIMARY KEY,
  username           VARCHAR(100) NOT NULL UNIQUE,
  password_hash      VARCHAR(255) NOT NULL,
  full_name          VARCHAR(255) NOT NULL,
  phone              VARCHAR(20),
  role               VARCHAR(20)  NOT NULL CHECK (role IN ('driver', 'admin')),
  profile_image_path VARCHAR(500),
  active             BOOLEAN      DEFAULT true,
  created_at         TIMESTAMP    DEFAULT NOW(),
  updated_at         TIMESTAMP    DEFAULT NOW()
);
```

Solo los **conductores** y **administradores** tienen cuentas de usuario. Los
pasajeros (usuarios de la app movil) son anonimos y no necesitan registrarse.

**Decisiones de diseno:**

- **`password_hash VARCHAR(255)`**: NUNCA se almacena la contrasena en texto plano.
  Se guarda el hash bcrypt (costo 10) de la contrasena. Bcrypt es un algoritmo de
  hashing disenado especificamente para contrasenas: es lento (por diseno) para
  dificultar ataques de fuerza bruta, e incluye un salt automatico para que dos
  usuarios con la misma contrasena tengan hashes diferentes.

- **`role CHECK (role IN ('driver', 'admin'))`**: La validacion de roles se hace a
  nivel de base de datos con una restriccion CHECK. Si alguien intenta insertar un
  rol invalido como `'superadmin'`, PostgreSQL rechaza la operacion. Esto es una
  capa de seguridad adicional mas alla de la validacion en el codigo Go.

- **`username UNIQUE`**: Garantiza que no existan dos usuarios con el mismo nombre
  de usuario. PostgreSQL crea automaticamente un indice B-tree sobre columnas UNIQUE.

- **Indices en `role` y `active`**: Aceleran las consultas filtradas que son comunes
  en la API de administracion: "listar todos los conductores activos".

- **`profile_image_path`**: Almacena la ruta del archivo de imagen, no la imagen en
  si. Las imagenes se guardan en el sistema de archivos del servidor (directorio
  `uploads/images/`). Almacenar imagenes en la base de datos (BYTEA) seria ineficiente
  para PostgreSQL e impediria servirlas directamente por HTTP.

- **Borrado logico**: `active = false` en lugar de DELETE, por las mismas razones
  que `stops` y `routes` (preservar historial de viajes y asignaciones).

### Tabla `refresh_tokens` (Tokens de Refresco)

```sql
CREATE TABLE refresh_tokens (
  id         SERIAL PRIMARY KEY,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  user_id    INT          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMP    NOT NULL,
  revoked    BOOLEAN      DEFAULT false,
  created_at TIMESTAMP    DEFAULT NOW()
);
```

Gestiona las sesiones de autenticacion JWT. Cuando un usuario inicia sesion, recibe
dos tokens: un **access token** (corta duracion, 15 minutos) y un **refresh token**
(larga duracion, 7 dias). El access token se usa en cada peticion HTTP. Cuando
expira, el cliente envia el refresh token para obtener un nuevo par de tokens sin
tener que ingresar usuario y contrasena otra vez.

**Decisiones de diseno:**

- **`token_hash` en lugar de almacenar el token en texto plano**: Si un atacante
  obtiene acceso a la base de datos (SQL injection, backup filtrado, etc.), no
  podria usar los tokens directamente porque estan hasheados. El token original
  solo existe en el dispositivo del usuario.

- **`revoked BOOLEAN`**: Permite invalidar un token especifico sin eliminarlo. Cuando
  el usuario cierra sesion o cuando se detecta actividad sospechosa, se marca el
  token como `revoked = true`. El token sigue existiendo en la base de datos (para
  auditoria) pero ya no es valido.

- **`ON DELETE CASCADE`**: Si se elimina un usuario, TODOS sus refresh tokens se
  eliminan automaticamente. Esto es correcto porque los tokens de un usuario eliminado
  no tienen razon de existir. Se usa CASCADE (borrado fisico) en lugar de borrado
  logico porque los tokens son datos operacionales, no datos de negocio que se
  necesiten preservar.

- **Estrategia de rotacion**: Cada vez que se usa un refresh token para obtener
  nuevos tokens, el token usado se revoca y se emite uno nuevo. Si un atacante roba
  un refresh token y lo usa, el token del usuario legitimo dejara de funcionar, lo
  que sirve como mecanismo de deteccion de compromiso.

---

## Grupo 3: Flota de Vehiculos

### Tabla `vehicles` (Vehiculos)

```sql
CREATE TABLE vehicles (
  id           SERIAL PRIMARY KEY,
  plate_number VARCHAR(20)  NOT NULL UNIQUE,
  route_id     INT          REFERENCES routes(id) ON DELETE SET NULL,
  status       VARCHAR(20)  DEFAULT 'inactive'
                            CHECK (status IN ('active', 'inactive', 'maintenance')),
  created_at   TIMESTAMP    DEFAULT NOW()
);
```

Registra los vehiculos (buses/combis) de la flota de transporte.

**Decisiones de diseno:**

- **`plate_number UNIQUE`**: La placa es el identificador natural de un vehiculo en
  Peru. Garantiza que no se registre el mismo vehiculo dos veces.

- **`route_id REFERENCES routes(id) ON DELETE SET NULL`**: Cada vehiculo puede estar
  asignado a una ruta, pero esta asignacion es **opcional** (la columna es nullable).
  Si se elimina una ruta, el vehiculo no se elimina; simplemente pierde su asignacion
  de ruta (`route_id` pasa a ser NULL). Esto es diferente al CASCADE usado en tokens
  porque un vehiculo es un recurso fisico valioso que no debe desaparecer solo porque
  se reorganizo una ruta.

- **`status CHECK`**: Tres estados posibles:
  - `active`: El vehiculo esta en servicio y puede ser rastreado
  - `inactive`: El vehiculo no esta operando (fuera de horario, sin conductor)
  - `maintenance`: El vehiculo esta en taller y no debe asignarse a conductores

### Tabla `vehicle_assignments` (Asignaciones de Vehiculos)

```sql
CREATE TABLE vehicle_assignments (
  id           SERIAL PRIMARY KEY,
  vehicle_id   INT       NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
  driver_id    INT       NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
  collector_id INT       REFERENCES users(id)             ON DELETE SET NULL,
  assigned_at  TIMESTAMP DEFAULT NOW(),
  active       BOOLEAN   DEFAULT true
);

CREATE UNIQUE INDEX idx_vehicle_assignments_active
  ON vehicle_assignments(vehicle_id) WHERE active = true;
```

Registra que conductor (y opcionalmente que cobrador) esta asignado a cada vehiculo.
En el transporte publico de Cajamarca, un vehiculo tipicamente tiene un conductor
y puede tener un cobrador que maneja los pagos de pasajeros.

**Decisiones de diseno:**

- **Indice parcial UNIQUE `WHERE active = true`**: Esta es una de las decisiones mas
  importantes del esquema. El indice garantiza que **solo puede haber una asignacion
  activa por vehiculo a la vez**. Sin embargo, al ser un indice parcial (filtrado por
  `active = true`), permite que existan multiples asignaciones inactivas para el mismo
  vehiculo. Esto preserva el historial completo: se puede ver que conductores han
  manejado un vehiculo en el pasado. Un indice UNIQUE normal sobre `vehicle_id` no
  permitiria este historial.

  Ejemplo: Si el vehiculo 1 fue conducido por Carlos lunes a viernes y luego se
  reasigna a Maria el sabado, la asignacion de Carlos se marca como `active = false`
  y se crea una nueva asignacion activa para Maria. El indice parcial permite ambas
  filas porque solo la de Maria tiene `active = true`.

- **`driver_id ON DELETE CASCADE`**: Si se elimina un conductor, sus asignaciones
  desaparecen. Un vehiculo sin conductor no tiene sentido operacionalmente.

- **`collector_id ON DELETE SET NULL`**: El cobrador es opcional. Si se elimina su
  cuenta, la asignacion sigue siendo valida (el conductor puede operar sin cobrador).
  Por eso se usa SET NULL en lugar de CASCADE.

### Tabla `vehicle_positions` (Posiciones GPS)

```sql
CREATE TABLE vehicle_positions (
  id          SERIAL PRIMARY KEY,
  vehicle_id  INT            NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE UNIQUE,
  geom        GEOMETRY(POINT, 4326) NOT NULL,
  heading     FLOAT,
  speed       FLOAT,
  recorded_at TIMESTAMP      NOT NULL
);
```

Almacena la **ultima posicion GPS conocida** de cada vehiculo. Esta tabla se actualiza
constantemente desde la aplicacion del conductor.

**Decisiones de diseno:**

- **`vehicle_id UNIQUE`**: Esta restriccion es fundamental y define el patron de la
  tabla. Cada vehiculo tiene **exactamente una fila** en esta tabla (o ninguna si
  nunca ha reportado posicion). Cuando el conductor envia una nueva posicion GPS,
  se usa un **UPSERT** (INSERT ... ON CONFLICT ... DO UPDATE) que reemplaza la
  posicion anterior. Esto significa que la tabla siempre contiene solo las posiciones
  actuales, no un historial.

  **Por que no almacenar historial**: Un historial de posiciones GPS (time series)
  generaria miles de filas por vehiculo por dia (una cada 5-10 segundos). La tabla
  creceria rapidamente y las consultas espaciales ("vehiculos cercanos") tendrian
  que filtrar enormes cantidades de datos historicos. Al mantener solo la posicion
  actual, la tabla tiene como maximo N filas (donde N es el numero de vehiculos),
  lo que garantiza consultas espaciales rapidas.

  Si en el futuro se necesita historial de posiciones (por ejemplo, para analisis
  de cobertura o reconstruccion de recorridos), se crearia una tabla separada
  `vehicle_position_history` optimizada para escrituras masivas (potencialmente
  usando TimescaleDB o particionamiento por fecha).

- **`heading FLOAT`**: Direccion del vehiculo en grados (0-360, donde 0 es norte).
  Es nullable porque no todos los dispositivos GPS reportan direccion.

- **`speed FLOAT`**: Velocidad en km/h. Tambien nullable.

- **`ON DELETE CASCADE`**: Si se elimina un vehiculo, su posicion se elimina
  automaticamente. Una posicion GPS sin vehiculo asociado no tiene utilidad.

---

## Grupo 4: Viajes y Operaciones

### Tabla `trips` (Viajes)

```sql
CREATE TABLE trips (
  id         SERIAL PRIMARY KEY,
  vehicle_id INT          NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
  route_id   INT          NOT NULL REFERENCES routes(id)   ON DELETE CASCADE,
  driver_id  INT          NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
  started_at TIMESTAMP    NOT NULL DEFAULT NOW(),
  ended_at   TIMESTAMP,
  status     VARCHAR(20)  DEFAULT 'active'
                          CHECK (status IN ('active', 'completed', 'cancelled'))
);
```

Registra cada recorrido que hace un bus. Cuando un conductor inicia su jornada,
se crea un viaje con `status = 'active'`. Cuando termina el recorrido, se actualiza
con `ended_at` y `status = 'completed'` (o `'cancelled'` si se cancelo).

**Decisiones de diseno:**

- **Tres FK con CASCADE**: Un viaje requiere un vehiculo, una ruta y un conductor.
  Si se elimina cualquiera de estos, el viaje pierde contexto y se elimina. Esta
  decision prioriza la integridad sobre la preservacion historica. En una version
  futura se podria cambiar a SET NULL si se necesitan estadisticas de viajes
  historicos independientes de los recursos que ya no existen.

- **`ended_at` nullable**: Es NULL mientras el viaje esta en progreso. Se llena
  cuando el conductor finaliza el viaje. Esto permite consultar facilmente los
  viajes activos: `WHERE ended_at IS NULL AND status = 'active'`.

- **Multiples indices**: La tabla `trips` tiene indices en `vehicle_id`, `route_id`,
  `driver_id`, `status` y `started_at`. Estos cubren las consultas mas comunes:
  "viajes activos de un vehiculo", "todos los viajes de una ruta", "historial de
  un conductor", y "viajes recientes".

### Tabla `alerts` (Alertas)

```sql
CREATE TABLE alerts (
  id            SERIAL PRIMARY KEY,
  title         VARCHAR(255) NOT NULL,
  description   TEXT,
  route_id      INT          REFERENCES routes(id) ON DELETE SET NULL,
  vehicle_plate VARCHAR(20),
  image_path    VARCHAR(500),
  created_by    INT          REFERENCES users(id)  ON DELETE SET NULL,
  created_at    TIMESTAMP    DEFAULT NOW()
);
```

Registra alertas e incidencias del servicio de transporte. Un administrador o
conductor puede crear alertas como "Desvio en Ruta 3 por obras en Av. Hoyos Rubio"
o "Vehiculo CAJ-302 fuera de servicio por averia".

**Decisiones de diseno:**

- **`route_id ON DELETE SET NULL`**: Si se elimina la ruta asociada, la alerta
  sigue existiendo pero sin ruta asociada. Las alertas son informacion temporal
  pero pueden ser utiles para revision historica.

- **`vehicle_plate VARCHAR(20)` en lugar de FK**: Es una **referencia suave** (soft
  reference). No tiene FK a `vehicles.plate_number`. Esto es intencional: permite
  crear alertas sobre vehiculos que quizas no estan registrados en el sistema (por
  ejemplo, un vehiculo de otra empresa que causo un incidente). Tambien evita
  problemas si se cambia la placa del vehiculo.

- **`created_by ON DELETE SET NULL`**: Si se elimina el usuario que creo la alerta,
  la alerta sigue visible pero sin autor. Se prioriza la preservacion de la
  informacion sobre la integridad referencial estricta.

- **`image_path`**: Permite adjuntar una foto a la alerta (por ejemplo, foto de un
  accidente o desvio). La imagen se almacena en el servidor, no en la base de datos.

---

## Grupo 5: Interaccion de Pasajeros

### Patron `device_id` (Usuarios Anonimos)

Una decision de arquitectura fundamental de Qapac es que los **pasajeros no
necesitan crear una cuenta** para usar la aplicacion. Esto reduce la barrera de
entrada (no hay registro, no hay verificacion de correo/telefono) y simplifica
la experiencia del usuario.

Para funcionalidades que requieren identificar al usuario (como calificar un viaje
o guardar rutas favoritas), se usa el **`device_id`**: un UUID unico generado por
el dispositivo Android. Este identificador es estable (no cambia al reiniciar la
app) pero anonimo (no esta vinculado a datos personales).

**Ventajas:**
- No se recopilan datos personales de los pasajeros (cumple con principios de
  privacidad)
- No hay friccion de registro
- Permite personalizar la experiencia (favoritos) sin requerir cuenta

**Limitaciones:**
- Si el usuario cambia de dispositivo, pierde sus favoritos y calificaciones
- Un usuario podria generar multiples `device_id` (reinstalando la app) para
  calificar un viaje varias veces (mitigado por la restriccion UNIQUE)

### Tabla `ratings` (Calificaciones)

```sql
CREATE TABLE ratings (
  id         SERIAL PRIMARY KEY,
  trip_id    INT          NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  rating     SMALLINT     NOT NULL CHECK (rating BETWEEN 1 AND 5),
  device_id  VARCHAR(255) NOT NULL,
  created_at TIMESTAMP    DEFAULT NOW(),
  UNIQUE(trip_id, device_id)
);
```

Permite a los pasajeros calificar un viaje de 1 a 5 estrellas.

**Decisiones de diseno:**

- **`rating SMALLINT CHECK (rating BETWEEN 1 AND 5)`**: La validacion del rango se
  hace a nivel de base de datos. Incluso si hay un bug en el frontend que envia un
  valor de 0 o 6, la base de datos lo rechaza. `SMALLINT` (2 bytes) es suficiente
  y mas eficiente que `INT` (4 bytes) para un rango tan pequeno.

- **`UNIQUE(trip_id, device_id)`**: Cada dispositivo puede calificar un viaje solo
  una vez. Esto previene que un usuario (o un bot) infle las calificaciones. Si el
  mismo dispositivo intenta calificar el mismo viaje otra vez, PostgreSQL rechaza la
  insercion.

- **`ON DELETE CASCADE` desde `trips`**: Si se elimina un viaje, sus calificaciones
  se eliminan. Las calificaciones no tienen sentido sin el viaje que las origino.

### Tabla `favorites` (Favoritos)

```sql
CREATE TABLE favorites (
  id         SERIAL PRIMARY KEY,
  device_id  VARCHAR(255) NOT NULL,
  route_id   INT          NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
  created_at TIMESTAMP    DEFAULT NOW(),
  UNIQUE(device_id, route_id)
);
```

Permite a los pasajeros marcar rutas como favoritas para acceso rapido.

**Decisiones de diseno:**

- **`UNIQUE(device_id, route_id)`**: Un dispositivo solo puede tener cada ruta
  como favorita una vez. Sin este constraint, tocar el boton de "favorito" varias
  veces crearia duplicados.

- **`ON DELETE CASCADE` desde `routes`**: Si se elimina una ruta, los favoritos
  asociados se eliminan. No tiene sentido mantener un favorito a una ruta que ya no
  existe.

- **Indices separados en `device_id` y `route_id`**: El indice en `device_id`
  optimiza la consulta "mostrar mis favoritos" (filtra por el device_id del usuario).
  El indice en `route_id` optimiza "cuantos favoritos tiene esta ruta" (consulta
  administrativa).

---

## Grupo 6: Cache de Rendimiento

### Que son las tablas UNLOGGED

PostgreSQL normalmente escribe cada cambio dos veces: una al **Write-Ahead Log**
(WAL) y otra a los archivos de datos. El WAL garantiza que, si el servidor se apaga
inesperadamente (corte de luz, crash del sistema operativo), los datos se puedan
recuperar al reiniciar.

Las tablas **UNLOGGED** omiten la escritura al WAL. Esto las hace significativamente
mas rapidas para escrituras, pero con una desventaja: **si el servidor se apaga
inesperadamente, los datos de estas tablas se pierden** (se truncan a cero filas).

Para datos de cache, esto es aceptable porque:
1. Los datos son **efimeros**: tienen un `expires_at` y se invalidan periodicamente
2. Los datos son **recalculables**: se pueden regenerar desde los datos fuente
3. La **velocidad de escritura** es critica: el cache se actualiza frecuentemente
4. La **perdida de datos** no afecta la funcionalidad, solo el rendimiento temporal
   (las primeras consultas despues de un reinicio seran mas lentas hasta que el
   cache se repueble)

```sql
CREATE UNLOGGED TABLE stop_eta_cache (...);
CREATE UNLOGGED TABLE route_to_stop_cache (...);
```

### Tabla `stop_eta_cache` (Cache de Tiempo Estimado de Llegada)

```sql
CREATE UNLOGGED TABLE stop_eta_cache (
  id          SERIAL PRIMARY KEY,
  stop_id     INT       NOT NULL REFERENCES stops(id),
  eta_seconds INT       NOT NULL,
  calc_ts     TIMESTAMP DEFAULT NOW(),
  expires_at  TIMESTAMP NOT NULL,
  UNIQUE(stop_id)
);
```

Almacena el tiempo estimado de llegada (ETA) del vehiculo mas cercano a cada parada.
Calcular un ETA en tiempo real requiere consultar posiciones de vehiculos, calcular
rutas y distancias — un proceso costoso. El cache evita repetir este calculo para
cada consulta.

**Decisiones de diseno:**

- **`stop_id UNIQUE`**: Cada parada tiene como maximo un ETA cacheado. Se usa
  UPSERT para actualizar: si ya existe un ETA para esa parada, se reemplaza.

- **`eta_seconds INT`**: Tiempo en segundos (no minutos ni formato HH:MM). Los
  segundos son la unidad mas precisa y el frontend puede formatear como prefiera.

- **`expires_at TIMESTAMP`**: Permite implementar politicas de invalidacion basadas
  en tiempo. El codigo de la API verifica si `expires_at < NOW()` antes de usar el
  valor cacheado.

### Tabla `route_to_stop_cache` (Cache de Rutas Peatonales)

```sql
CREATE UNLOGGED TABLE route_to_stop_cache (
  id          SERIAL PRIMARY KEY,
  origin_hash VARCHAR(50)  NOT NULL,
  stop_id     INT          NOT NULL REFERENCES stops(id),
  polyline    TEXT         NOT NULL,
  distance_m  INT          NOT NULL,
  duration_s  INT          NOT NULL,
  calc_ts     TIMESTAMP    DEFAULT NOW(),
  expires_at  TIMESTAMP    NOT NULL,
  UNIQUE(origin_hash, stop_id)
);
```

Almacena las rutas peatonales calculadas desde la ubicacion del usuario hasta las
paradas cercanas. El calculo de rutas peatonales se hace a traves de la
**Google Routes API** (servicio externo de pago), por lo que cachear los resultados
reduce costos y mejora la velocidad de respuesta.

**Decisiones de diseno:**

- **`origin_hash VARCHAR(50)`**: En lugar de almacenar las coordenadas exactas del
  origen (que cambiarian con cada paso del usuario), se usa un hash que agrupa
  ubicaciones cercanas. Asi, si el usuario se mueve unos pocos metros, se reutiliza
  el cache de la posicion anterior. El hash se calcula redondeando las coordenadas
  a una precision determinada.

- **`UNIQUE(origin_hash, stop_id)`**: Para cada combinacion de "zona de origen" y
  "parada destino", solo hay un resultado cacheado. Esto permite UPSERT eficiente.

- **`polyline TEXT`**: Almacena la ruta peatonal en formato **Google Encoded Polyline**,
  un formato compacto de texto que representa una secuencia de coordenadas. La
  aplicacion movil decodifica este texto para dibujar la ruta en el mapa.

- **`distance_m` y `duration_s`**: Distancia en metros y duracion en segundos de
  la caminata. Se almacenan como enteros porque la precision de metros/segundos es
  suficiente para informacion de transporte urbano.

---

## Grupo 7: Sistema de Migraciones

### Tabla `schema_migrations`

```sql
CREATE TABLE schema_migrations (
  version    VARCHAR(255) PRIMARY KEY,
  applied_at TIMESTAMP DEFAULT NOW()
);
```

Controla que migraciones de esquema se han aplicado a la base de datos. Es la
primera tabla que se crea (migracion 000) y es requisito para todas las demas.

**Decisiones de diseno:**

- **`version VARCHAR(255) PRIMARY KEY`**: El nombre del archivo de migracion sirve
  como identificador unico. Por ejemplo: `"001_initial_schema"`, `"003_users_and_auth"`.
  Se usa VARCHAR en lugar de INT porque es mas descriptivo y permite referenciar
  rapidamente que contiene cada migracion.

- **Sin columnas adicionales**: La tabla es deliberadamente simple. Solo necesita
  saber "esta migracion ya se aplico" (existencia de la fila) y "cuando" (applied_at).

---

## Estrategias de Eliminacion

El esquema utiliza **tres estrategias diferentes** para manejar la eliminacion de
datos, dependiendo de la naturaleza de cada tabla:

### 1. Borrado logico (Soft Delete): `active = false`

Se usa en **entidades de negocio** que representan recursos fisicos o administrativos
con valor historico.

| Tabla | Columna | Justificacion |
|-------|---------|---------------|
| `stops` | `active` | Preservar referencias en viajes historicos y route_stops |
| `routes` | `active` | Preservar viajes y asignaciones de vehiculos historicas |
| `users` | `active` | Preservar historial de viajes del conductor |
| `vehicle_assignments` | `active` | Preservar historial de asignaciones (quien condujo que vehiculo) |

El borrado logico permite:
- **Reactivar** el recurso sin recrearlo
- **Preservar** la integridad referencial (no hay FK rotas)
- **Auditar** quien existio en el sistema

### 2. Borrado en cascada (ON DELETE CASCADE)

Se usa en **datos operacionales** que dependen completamente de su entidad padre
y no tienen sentido sin ella.

| Tabla hija | Tabla padre | Justificacion |
|------------|-------------|---------------|
| `refresh_tokens` | `users` | Tokens de un usuario eliminado son invalidos |
| `vehicle_positions` | `vehicles` | Posicion GPS sin vehiculo es inutil |
| `vehicle_assignments` | `vehicles` | Asignacion sin vehiculo no tiene sentido |
| `vehicle_assignments` | `users` (driver) | Asignacion sin conductor no es valida |
| `trips` | `vehicles`, `routes`, `users` | Viaje sin sus entidades pierde contexto |
| `ratings` | `trips` | Calificacion sin viaje no tiene sentido |
| `favorites` | `routes` | Favorito de ruta eliminada no es util |

### 3. SET NULL (ON DELETE SET NULL)

Se usa en **referencias opcionales** donde la entidad hija sigue siendo valida
sin la referencia.

| Tabla | Columna | Tabla referenciada | Justificacion |
|-------|---------|--------------------|---------------|
| `vehicles` | `route_id` | `routes` | Vehiculo sigue existiendo sin ruta asignada |
| `vehicle_assignments` | `collector_id` | `users` | Asignacion valida sin cobrador |
| `alerts` | `route_id` | `routes` | Alerta sigue siendo informativa sin ruta |
| `alerts` | `created_by` | `users` | Alerta sigue siendo visible sin autor |

---

## Indices y Rendimiento

El esquema define **22 indices explicitamente** ademas de los indices automaticos
creados por PostgreSQL para columnas PRIMARY KEY y UNIQUE.

### Indices espaciales (GiST)

| Indice | Tabla | Columna | Consulta que optimiza |
|--------|-------|---------|-----------------------|
| `idx_stops_geom` | `stops` | `geom` | Buscar paradas cercanas a coordenadas GPS |
| `idx_route_shapes_geom` | `route_shapes` | `geom` | Buscar rutas que pasan por una zona |
| `idx_vehicle_positions_geom` | `vehicle_positions` | `geom` | Buscar vehiculos cercanos al usuario |

### Indices B-tree de filtrado

| Indice | Tabla | Columna(s) | Consulta que optimiza |
|--------|-------|------------|-----------------------|
| `idx_users_role` | `users` | `role` | Listar solo conductores o solo admins |
| `idx_users_active` | `users` | `active` | Filtrar usuarios activos/inactivos |
| `idx_vehicles_route_id` | `vehicles` | `route_id` | Vehiculos de una ruta especifica |
| `idx_vehicles_status` | `vehicles` | `status` | Vehiculos por estado operativo |
| `idx_trips_status` | `trips` | `status` | Viajes activos vs completados |
| `idx_trips_started_at` | `trips` | `started_at` | Viajes recientes, ordenados por fecha |

### Indices B-tree de FK (join)

| Indice | Tabla | Columna | Consulta que optimiza |
|--------|-------|---------|-----------------------|
| `idx_refresh_tokens_user_id` | `refresh_tokens` | `user_id` | Tokens de un usuario |
| `idx_vehicle_assignments_driver` | `vehicle_assignments` | `driver_id` | Asignaciones de un conductor |
| `idx_vehicle_assignments_collector` | `vehicle_assignments` | `collector_id` | Asignaciones con un cobrador |
| `idx_trips_vehicle_id` | `trips` | `vehicle_id` | Viajes de un vehiculo |
| `idx_trips_route_id` | `trips` | `route_id` | Viajes de una ruta |
| `idx_trips_driver_id` | `trips` | `driver_id` | Viajes de un conductor |
| `idx_ratings_trip_id` | `ratings` | `trip_id` | Calificaciones de un viaje |
| `idx_favorites_device_id` | `favorites` | `device_id` | Favoritos de un dispositivo |
| `idx_favorites_route_id` | `favorites` | `route_id` | Favoritos de una ruta |
| `idx_alerts_route_id` | `alerts` | `route_id` | Alertas de una ruta |
| `idx_alerts_created_at` | `alerts` | `created_at DESC` | Alertas recientes (orden descendente) |
| `idx_eta_cache_stop_id` | `stop_eta_cache` | `stop_id` | Buscar ETA cacheado de una parada |

### Indice parcial unico

| Indice | Tabla | Expresion | Proposito |
|--------|-------|-----------|-----------|
| `idx_vehicle_assignments_active` | `vehicle_assignments` | `(vehicle_id) WHERE active = true` | Garantizar una sola asignacion activa por vehiculo |

### Por que tantos indices en FK

PostgreSQL **no crea automaticamente indices en columnas de foreign key** (a
diferencia de MySQL/InnoDB). Sin indices explicititos en FK, las operaciones de JOIN
y las verificaciones de CASCADE escanearian tablas completas. Como la API hace JOINs
frecuentemente (por ejemplo, listar vehiculos con sus rutas, mostrar viajes con
datos del conductor), estos indices son criticos para el rendimiento.

---

## Sistema de Migraciones Embebidas

Las migraciones no se ejecutan con una herramienta externa (como `golang-migrate` o
`flyway`). En cambio, los archivos SQL estan **embebidos en el binario** de Go
usando la directiva `//go:embed`:

```go
//go:embed *.sql
var migrationFiles embed.FS
```

Al iniciar el servidor, el sistema de migraciones:

1. Crea la tabla `schema_migrations` si no existe (migracion 000)
2. Lee todos los archivos `*.sql` embebidos y los ordena alfabeticamente
3. Para cada archivo, verifica si ya esta registrado en `schema_migrations`
4. Si no esta registrado, ejecuta el SQL y registra la version

**Ventajas de este enfoque:**

- **Despliegue simple**: No se necesita instalar herramientas de migracion en el
  servidor. El binario Go incluye todo lo necesario.
- **Idempotencia**: Se puede reiniciar el servidor multiples veces sin que las
  migraciones se apliquen dos veces (verificacion por `schema_migrations`).
- **Todas las tablas usan `IF NOT EXISTS`**: Capa adicional de seguridad. Incluso
  si `schema_migrations` tuviera un error, las migraciones no fallan al intentar
  crear tablas que ya existen.
- **Orden determinista**: Los archivos se nombran con prefijo numerico (`000_`,
  `001_`, ..., `009_`) para garantizar el orden de ejecucion.

### Orden de migraciones

| # | Archivo | Contenido |
|---|---------|-----------|
| 000 | `000_migrations_table.sql` | Tabla de control `schema_migrations` |
| 001 | `001_initial_schema.sql` | PostGIS + paradas, rutas, route_stops, route_shapes, tablas de cache |
| 002 | `002_seed_data.sql` | Datos iniciales de Lima (demo) |
| 003 | `003_users_and_auth.sql` | Usuarios (conductores, admins) y tokens de refresco |
| 004 | `004_vehicles.sql` | Vehiculos, asignaciones de personal, posiciones GPS |
| 005 | `005_trips.sql` | Registro de viajes |
| 006 | `006_alerts.sql` | Alertas e incidencias |
| 007 | `007_ratings.sql` | Calificaciones de viajes (anonimas) |
| 008 | `008_favorites.sql` | Rutas favoritas (anonimas) |
| 009 | `009_seed_cajamarca.sql` | Reemplaza datos de Lima con datos reales de Cajamarca |

---

## Resumen de Estadisticas

| Metrica | Valor |
|---------|-------|
| Total de tablas | 16 (14 regulares + 2 UNLOGGED) |
| Total de columnas | 87 |
| Foreign keys | 16 |
| Indices explicitos | 22 |
| Restricciones CHECK | 4 (`role`, `status`, `rating`, `trip status`) |
| Restricciones UNIQUE | 10 (incluyendo 1 parcial) |
| Tipos de geometria | 2 (POINT, LINESTRING) |
| Indices GiST | 3 |
| Tablas con borrado logico | 4 (`stops`, `routes`, `users`, `vehicle_assignments`) |
| Tablas con ON DELETE CASCADE | 7 |
| Tablas con ON DELETE SET NULL | 3 |
| Migraciones SQL | 10 (000–009) |
