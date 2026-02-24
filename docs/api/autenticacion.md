# Autenticacion

## Descripcion General

La API usa **JWT (JSON Web Tokens)** con algoritmo **HS256** para autenticar conductores y administradores.

**Los usuarios normales (pasajeros) NO necesitan autenticarse.**

## Flujo de Autenticacion

```
1. POST /api/v1/auth/login       → access_token + refresh_token
2. Usar access_token en header    → Authorization: Bearer <token>
3. Cuando expire (15 min):
   POST /api/v1/auth/refresh      → nuevo access_token + refresh_token
4. Al cerrar sesion:
   POST /api/v1/auth/logout       → revoca el refresh token
```

## Detalles Tecnicos

| Propiedad | Valor |
|-----------|-------|
| Algoritmo | HS256 |
| Access Token TTL | 15 minutos (configurable via `ACCESS_TOKEN_TTL`) |
| Refresh Token TTL | 7 dias (configurable via `REFRESH_TOKEN_TTL`) |
| Refresh Token formato | 64 caracteres hexadecimales |
| Almacenamiento | Refresh token se guarda como hash SHA-256 en BD |
| Rotacion | Cada refresh emite un nuevo par y revoca el anterior |

## Claims del JWT

El access token contiene estos claims:

```json
{
  "user_id": 2,
  "username": "conductor1",
  "role": "driver",
  "exp": 1705312800,
  "iat": 1705311900
}
```

## Roles

| Rol | Acceso |
|-----|--------|
| `driver` | Endpoints `/api/v1/driver/*` y `/api/v1/auth/*` |
| `admin` | Endpoints `/api/v1/admin/*`, `/api/v1/driver/*` y `/api/v1/auth/*` |

> **Nota**: Un admin puede acceder a los endpoints de conductor tambien.

---

## POST /api/v1/auth/login

Inicia sesion y retorna tokens.

**Request:**

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "conductor1",
  "password": "driver123"
}
```

**Response 200:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
  "user": {
    "id": 2,
    "username": "conductor1",
    "full_name": "Carlos Quispe",
    "role": "driver"
  }
}
```

**Errores:**

| Codigo | Mensaje | Causa |
|--------|---------|-------|
| 400 | `username and password are required` | Falta username o password |
| 401 | `invalid username or password` | Credenciales incorrectas |
| 500 | `login failed` | Error interno |

---

## POST /api/v1/auth/refresh

Renueva el par de tokens. El refresh token anterior queda invalidado.

**Request:**

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "a1b2c3d4e5f6..."
}
```

**Response 200:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "f6e5d4c3b2a1f6e5d4c3b2a1..."
}
```

**Errores:**

| Codigo | Mensaje | Causa |
|--------|---------|-------|
| 400 | `refresh_token is required` | Falta el refresh token |
| 401 | `invalid or expired refresh token` | Token invalido, expirado o ya revocado |
| 500 | `token refresh failed` | Error interno |

---

## POST /api/v1/auth/logout

Revoca el refresh token. El access token sigue siendo valido hasta que expire.

**Request:**

```http
POST /api/v1/auth/logout
Content-Type: application/json

{
  "refresh_token": "a1b2c3d4e5f6..."
}
```

**Response:** `204 No Content` (sin cuerpo)

**Errores:**

| Codigo | Mensaje | Causa |
|--------|---------|-------|
| 400 | `refresh_token is required` | Falta el refresh token |
| 500 | `logout failed` | Error interno |

---

## Uso del Token en Android (Kotlin)

```kotlin
// Guardar tokens despues del login
val prefs = getSharedPreferences("qapac_auth", MODE_PRIVATE)
prefs.edit()
    .putString("access_token", loginResponse.accessToken)
    .putString("refresh_token", loginResponse.refreshToken)
    .apply()

// Interceptor para agregar el token a cada request
class AuthInterceptor(private val prefs: SharedPreferences) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val token = prefs.getString("access_token", null)
        val request = chain.request().newBuilder()
        if (token != null) {
            request.addHeader("Authorization", "Bearer $token")
        }
        return chain.proceed(request.build())
    }
}

// Interceptor para renovar token automaticamente al recibir 401
class TokenRefreshInterceptor(
    private val prefs: SharedPreferences,
    private val apiService: AuthApiService
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val response = chain.proceed(chain.request())
        if (response.code == 401) {
            val refreshToken = prefs.getString("refresh_token", null) ?: return response
            val newTokens = apiService.refresh(RefreshRequest(refreshToken)).execute()
            if (newTokens.isSuccessful) {
                val body = newTokens.body()!!
                prefs.edit()
                    .putString("access_token", body.accessToken)
                    .putString("refresh_token", body.refreshToken)
                    .apply()
                // Reintentar request original con nuevo token
                val newRequest = chain.request().newBuilder()
                    .header("Authorization", "Bearer ${body.accessToken}")
                    .build()
                response.close()
                return chain.proceed(newRequest)
            }
        }
        return response
    }
}
```
