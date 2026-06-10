# 2026-06-10 · Combobox de intervalo de autorefresco en el panel del operador

## Context

El panel web del operador (`cmd/server/index.html`) refresca la lista de apps
automáticamente cada 15 segundos fijos cuando el toggle "auto" está activo. Se
pide añadir un combobox (`<select>`) que permita elegir ese intervalo entre un
conjunto de valores predefinidos, integrándolo con el toggle existente y
persistiendo la elección en `localStorage`. El cambio es íntegramente frontend,
dentro del único HTML embebido; no hay backend que tocar.

## Decisions

No se realizó interview: la tarea queda resuelta por el código y los docs. Se
toman estos defaults, todos registrados aquí como asunciones:

1. **Valores del combobox**: `5s`, `10s`, `15s`, `30s`, `1m`, `5m`. Default
   `15s` (mantiene el comportamiento actual y evita cambiar la cadencia de carga
   percibida tras desplegar). Razón: cubre desde monitorización activa de un
   deploy (5–10s) hasta vigilancia pasiva (1–5m) sin saturar al agente/cluster.
2. **Persistencia**: `localStorage`, replicando el patrón ya usado para el token
   (`TOKEN_KEY = "kubeops.apiToken"`). Nueva clave `kubeops.refreshInterval` que
   guarda los milisegundos del intervalo. Si el valor guardado no está en la
   lista de opciones, se ignora y se usa el default.
3. **Persistencia del toggle auto**: actualmente el estado del checkbox `auto`
   NO se persiste (siempre arranca `checked`). Se mantiene ese comportamiento
   para acotar el cambio; solo se persiste el intervalo. (Asunción: no se pidió
   persistir el toggle.)
4. **Ubicación del control**: en el header (`.bar`), inmediatamente a la
   izquierda del toggle `auto`, como `<select>` reutilizando los estilos `select`
   ya existentes. El combobox queda visualmente acoplado al toggle que gobierna.
5. **Mecanismo de timer**: se mantiene `setInterval` (no se migra a recursión con
   `setTimeout`), reprogramándolo al cambiar el intervalo. Razón: cambio mínimo
   sobre la implementación actual.

Si algo de lo anterior no encaja con lo esperado, ajustar antes de implementar.

## Approach

El panel es un único `cmd/server/index.html` autocontenido (HTML + CSS + JS
vanilla, `"use strict"`, sin bundler ni dependencias), embebido con `//go:embed`
en `cmd/server/web.go` y servido en `GET /{$}`. Toda la persistencia de
preferencias de cliente ya vive en `localStorage`. Por tanto:

- El combobox se añade como markup inline en el header, usando los estilos
  `select` existentes (no se crean estilos paralelos; alineado con
  `docs/design.md` → "Reuse the existing component classes").
- La lógica de polling (`scheduleRefresh`, líneas ~598-606) se generaliza para
  leer el intervalo seleccionado en vez de la constante `15000`.
- La preferencia se guarda/lee con un par `getRefreshInterval()` /
  `setRefreshInterval()` que imita `getToken()` / `setToken()` (líneas 305-311).

No hay cambios de arquitectura Go: `web.go`, `main.go` y el resto de slices
quedan intactos. No hay endpoint nuevo ni cambio de contrato API. Encaja con la
arquitectura porque el panel es deliberadamente un asset estático embebido y la
funcionalidad es puramente de presentación cliente.

## Steps

Todos los cambios son en `cmd/server/index.html`. Referencias de línea sobre el
estado actual del fichero.

1. **Constante de clave y lista de intervalos (JS, junto a `TOKEN_KEY`, ~línea
   305).** Añadir:
   - `const REFRESH_KEY = "kubeops.refreshInterval";`
   - `const REFRESH_DEFAULT_MS = 15000;`
   - Un array de opciones `[{ ms, label }]`:
     `5000/"5s"`, `10000/"10s"`, `15000/"15s"`, `30000/"30s"`, `60000/"1m"`,
     `300000/"5m"`. Mantenerlo como única fuente de verdad (se usa para poblar el
     `<select>` y para validar el valor persistido).

2. **Helpers de persistencia (JS, justo después de `setToken`, ~línea 311).**
   Reutilizar el patrón de `getToken`/`setToken`:
   - `getRefreshInterval()`: lee `REFRESH_KEY`, lo parsea a entero; si no es un ms
     presente en la lista de opciones, devuelve `REFRESH_DEFAULT_MS`.
   - `setRefreshInterval(ms)`: `localStorage.setItem(REFRESH_KEY, String(ms))`.

3. **Markup del combobox (HTML, header `.bar`, antes del `<label class="toggle">`
   de `#autorefresh`, ~línea 227).** Añadir:
   ```html
   <select id="refreshInterval" title="Intervalo de autorefresco"></select>
   ```
   Usa el estilo `select` global ya definido (líneas 94-97); no requiere CSS
   nuevo. Mantener las opciones vacías en el markup y poblarlas desde JS (paso 4)
   para no duplicar la lista. Quitar el `15s` hardcodeado del `title` del toggle
   `auto` (línea 227: `title="Refrescar automáticamente cada 15s"`) y dejarlo
   genérico (p. ej. "Refrescar automáticamente"), porque la cadencia ya no es
   fija.

4. **Poblado e inicialización del `<select>` (JS, en el bloque "Wiring", junto al
   resto de `addEventListener`, ~línea 577-606).** Antes de `scheduleRefresh()`:
   - Generar las `<option>` desde la lista del paso 1 (mismo patrón que
     `populateEnvFilter`, líneas 457-464, con `esc()` por consistencia aunque las
     labels sean estáticas).
   - Fijar `$("refreshInterval").value = String(getRefreshInterval())`.

5. **Generalizar el timer (JS, `scheduleRefresh`, líneas 598-606).** Sustituir el
   literal `15000` por `getRefreshInterval()`:
   ```js
   function scheduleRefresh() {
     if (timer) clearInterval(timer);
     timer = setInterval(() => {
       if ($("autorefresh").checked && document.visibilityState === "visible") load(false);
     }, getRefreshInterval());
   }
   ```
   El gating por checkbox y por `visibilityState` se mantiene intacto.

6. **Listener del combobox (JS, junto a `$("autorefresh").addEventListener(...)`,
   línea 606).** Al cambiar:
   ```js
   $("refreshInterval").addEventListener("change", () => {
     setRefreshInterval(parseInt($("refreshInterval").value, 10));
     scheduleRefresh();
   });
   ```
   `scheduleRefresh()` ya hace `clearInterval` + `setInterval`, así que el cambio
   en caliente reinicia el ciclo con el nuevo periodo. No se dispara un `load`
   inmediato (se respeta el ritmo elegido; el botón "Refrescar" sigue disponible
   para forzarlo).

## Risks

- **Comportamiento por defecto inalterado**: con `15s` como default y el toggle
  arrancando `checked`, un usuario sin preferencia guardada ve exactamente el
  comportamiento previo. Riesgo bajo; verificar manualmente.
- **Valor persistido inválido o de una versión futura** (p. ej. se elimina una
  opción): `getRefreshInterval()` debe degradar al default si el ms no está en la
  lista. Cubierto en el paso 2; es el punto a no olvidar.
- **Cambio de intervalo con auto desactivado**: el `setInterval` sigue
  programado pero su callback no hace nada porque el checkbox está sin marcar; al
  reactivar `auto` ya usa el último intervalo elegido. Sin acción extra; es el
  comportamiento esperado.
- **Estilo del `<select>` en el header**: el header puede estrecharse en el
  breakpoint de 560px. El `<select>` hereda el estilo global y debería ajustarse;
  revisar visualmente en móvil que no rompa el wrap de `.bar`. Si molesta, es un
  ajuste CSS menor, no estructural.
- **Test existente** (`cmd/server/web_test.go`) solo comprueba que el HTML se
  sirve y contiene el marcador "kubeops-agent"; no se rompe. No hay framework de
  test JS en el proyecto, así que la verificación del combobox es manual.

## Verification

- `gofmt -w .` y `go vet ./...` (no debería haber cambios Go, pero confirma que
  nada se desformatea).
- `go test ./...` debe seguir pasando (cubre el `serveIndex` embebido).
- Manual, arrancando `CONFIG_FILE=config.yaml go run ./cmd/server/main.go` y
  abriendo el panel:
  1. El combobox aparece en el header junto al toggle `auto` y muestra `15s` por
     defecto en una sesión limpia.
  2. Con `auto` activo, cambiar a `5s` acelera el refresco (observable en la red
     del navegador / DevTools, peticiones a `api/v1/apps`); cambiar a `1m` lo
     espacia.
  3. Recargar la página conserva el intervalo elegido (localStorage
     `kubeops.refreshInterval`).
  4. Con `auto` desactivado, ningún intervalo dispara refrescos automáticos; el
     botón "Refrescar" sigue funcionando.
  5. Inyectar un valor inválido en localStorage
     (`localStorage.setItem("kubeops.refreshInterval","999")`) y recargar: el
     combobox vuelve al default `15s`.
  6. Comprobar en modo claro y oscuro y en el breakpoint de 560px que el header
     no se rompe.

## Missing docs

Ninguno: `docs/architecture.md`, `docs/code-style.md` y `docs/design.md` existen
y se han usado para alinear el plan.
