# BiletFlow Scanner

The Expo app venue staff use at the door. This week it signs in against the Go
API and keeps the session on the device; the event list and the camera
scanner come next.

## Run it

The API has to be running first (`make up` and `make api-run` from the
repository root). Then, from this directory:

```bash
npm install
npx expo start
```

Scan the QR code with Expo Go on a phone on the **same Wi-Fi** as your computer,
or press `i` / `a` for a simulator.

### Which API does the phone talk to?

A phone cannot reach `localhost` - that would be the phone itself. The app
takes the development machine's LAN address from Expo and talks to port 8080
there, so on the same Wi-Fi it works with no configuration. The address is
printed at the bottom of the sign-in screen.

To point it somewhere else, copy `.env.example` to `.env.local` and set:

```bash
EXPO_PUBLIC_API_BASE_URL=http://192.168.1.24:8080/api/v1
```

## Layout

```
app/
  _layout.tsx        stack navigator, AuthProvider
  index.tsx          waits for the session, then shows who is signed in
  login.tsx          email + password
lib/
  api.ts             fetch wrapper with a 10-second timeout
  auth-context.tsx   restores the session on launch and confirms it with /auth/me
  session.ts         the token, kept in the keychain (expo-secure-store)
  config.ts          where the API is
  theme.ts           colours
  types.ts           mirrors of the API's JSON
```

### The token is in the keychain

A scanner is a shared device on a table at the entrance, so the access token is
stored with `expo-secure-store` (iOS Keychain, Android Keystore) rather than in
plain storage.

## Checks

```bash
npx tsc --noEmit     # or `make scan-check` from the repository root
```
